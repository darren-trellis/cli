/*
Copyright © 2023 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tui

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
)

const defaultThemeName = "default"

//go:embed themes/*.toml
var embeddedThemeFS embed.FS

type Theme struct {
	Name        string
	Background  lipgloss.Color
	Accent      lipgloss.Color
	ActiveEnv   lipgloss.Color
	Border      lipgloss.Color
	Title       lipgloss.Color
	Dim         lipgloss.Color
	Dirty       lipgloss.Color
	Delete      lipgloss.Color
	Error       lipgloss.Color
	Text        lipgloss.Color
	SelectionBg lipgloss.Color
	SelectionFg lipgloss.Color
}

type themeFile struct {
	Name   string            `toml:"name"`
	Colors themeFileColors   `toml:"colors"`
	Levels themeFileLevels   `toml:"levels"`
	UI     themeFileUI       `toml:"ui"`
}

type themeFileColors struct {
	Background        string `toml:"background"`
	Foreground        any    `toml:"foreground"`
	SelectionBg       string `toml:"selection_bg"`
	SelectionFg       string `toml:"selection_fg"`
	Border            any    `toml:"border"`
	WindowFocusBorder any    `toml:"window_focus_border"`
	ActiveEnv         any    `toml:"active_env"`
	Dim               any    `toml:"dim"`
}

type themeFileLevels struct {
	Info  any `toml:"info"`
	Warn  any `toml:"warn"`
	Error any `toml:"error"`
}

type themeFileUI struct{}

var themes = map[string]Theme{}

func init() {
	registerBuiltinThemes()
	_ = loadEmbeddedThemes()
	_ = applyTheme(defaultThemeName)
}

func registerBuiltinThemes() {
	themes["default"] = Theme{
		Name:        "default",
		Background:  "",
		Accent:      lipgloss.Color("5"),
		ActiveEnv:   lipgloss.Color("2"),
		Border:      lipgloss.Color("8"),
		Title:       lipgloss.Color("15"),
		Dim:         lipgloss.Color("8"),
		Dirty:       lipgloss.Color("3"),
		Delete:      lipgloss.Color("1"),
		Error:       lipgloss.Color("1"),
		Text:        lipgloss.Color("15"),
		SelectionBg: "",
		SelectionFg: lipgloss.Color("5"),
	}
	themes["cool"] = Theme{
		Name:        "cool",
		Accent:      lipgloss.Color("6"),
		ActiveEnv:   lipgloss.Color("2"),
		Border:      lipgloss.Color("8"),
		Title:       lipgloss.Color("15"),
		Dim:         lipgloss.Color("8"),
		Dirty:       lipgloss.Color("4"),
		Delete:      lipgloss.Color("1"),
		Error:       lipgloss.Color("1"),
		Text:        lipgloss.Color("15"),
		SelectionFg: lipgloss.Color("6"),
	}
	themes["warm"] = Theme{
		Name:        "warm",
		Accent:      lipgloss.Color("208"),
		ActiveEnv:   lipgloss.Color("2"),
		Border:      lipgloss.Color("8"),
		Title:       lipgloss.Color("15"),
		Dim:         lipgloss.Color("8"),
		Dirty:       lipgloss.Color("3"),
		Delete:      lipgloss.Color("1"),
		Error:       lipgloss.Color("1"),
		Text:        lipgloss.Color("15"),
		SelectionFg: lipgloss.Color("208"),
	}
	themes["mono"] = Theme{
		Name:        "mono",
		Accent:      lipgloss.Color("15"),
		ActiveEnv:   lipgloss.Color("7"),
		Border:      lipgloss.Color("8"),
		Title:       lipgloss.Color("15"),
		Dim:         lipgloss.Color("8"),
		Dirty:       lipgloss.Color("7"),
		Delete:      lipgloss.Color("1"),
		Error:       lipgloss.Color("1"),
		Text:        lipgloss.Color("15"),
		SelectionFg: lipgloss.Color("15"),
	}
}

func loadEmbeddedThemes() error {
	entries, err := fs.ReadDir(embeddedThemeFS, "themes")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		data, err := embeddedThemeFS.ReadFile("themes/" + entry.Name())
		if err != nil {
			return err
		}
		theme, err := parseTeleminatorTheme(data)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
		themes[theme.Name] = theme
	}
	return nil
}

func parseTeleminatorTheme(data []byte) (Theme, error) {
	var file themeFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return Theme{}, err
	}
	name := strings.TrimSpace(file.Name)
	if name == "" {
		return Theme{}, fmt.Errorf("theme missing name")
	}

	fg := colorFromAny(file.Colors.Foreground, "#cdd6f4")
	dim := colorFromAny(file.Colors.Dim, "#6c7086")
	border := colorFromAny(file.Colors.Border, dim)
	accent := colorFromAny(file.Colors.WindowFocusBorder, fg)
	info := colorFromAny(file.Levels.Info, accent)
	activeEnv := colorFromAny(file.Colors.ActiveEnv, info)
	warn := colorFromAny(file.Levels.Warn, "#f9e2af")
	errColor := colorFromAny(file.Levels.Error, "#f38ba8")
	selBg := strings.TrimSpace(file.Colors.SelectionBg)
	selFg := strings.TrimSpace(file.Colors.SelectionFg)
	if selFg == "" {
		selFg = fg
	}
	bg := strings.TrimSpace(file.Colors.Background)

	return Theme{
		Name:        name,
		Background:  lipgloss.Color(bg),
		Accent:      lipgloss.Color(accent),
		ActiveEnv:   lipgloss.Color(activeEnv),
		Border:      lipgloss.Color(border),
		Title:       lipgloss.Color(fg),
		Dim:         lipgloss.Color(dim),
		Dirty:       lipgloss.Color(warn),
		Delete:      lipgloss.Color(errColor),
		Error:       lipgloss.Color(errColor),
		Text:        lipgloss.Color(fg),
		SelectionBg: lipgloss.Color(selBg),
		SelectionFg: lipgloss.Color(selFg),
	}, nil
}

func colorFromAny(v any, fallback string) string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
	case map[string]any:
		if fg, ok := t["fg"].(string); ok && strings.TrimSpace(fg) != "" {
			return strings.TrimSpace(fg)
		}
	}
	return fallback
}

func themeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func applyTheme(name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		name = defaultThemeName
	}
	theme, ok := themes[name]
	if !ok {
		return fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(themeNames(), ", "))
	}
	rebuildStyles(theme)
	return nil
}
