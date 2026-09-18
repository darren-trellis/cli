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
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/pelletier/go-toml/v2"
)

const defaultThemeName = "default"

//go:embed themes/*.toml
var embeddedThemeFS embed.FS

type Theme struct {
	Name          string
	Background    tcell.Color
	Accent        tcell.Color
	ActiveEnv     tcell.Color
	Border        tcell.Color
	Title         tcell.Color
	Dim           tcell.Color
	Dirty         tcell.Color
	Delete        tcell.Color
	Error         tcell.Color
	Text          tcell.Color
	SelectionBg   tcell.Color
	SelectionFg   tcell.Color
	SearchMatchFg tcell.Color
	SearchMatchBg tcell.Color
}

type themeFile struct {
	Name   string          `toml:"name"`
	Colors themeFileColors `toml:"colors"`
	Levels themeFileLevels `toml:"levels"`
	UI     themeFileUI     `toml:"ui"`
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
	SearchMatch       any    `toml:"search_match"`
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

// parseColor turns a theme file colour into a tcell colour. An empty string
// means "inherit the terminal default"; bare digits are ANSI palette indexes
// (the built-in themes use those so they follow the user's own palette).
func parseColor(s string) tcell.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return tcell.ColorDefault
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 255 {
		return tcell.PaletteColor(n)
	}
	c := tcell.GetColor(s)
	if c == tcell.ColorDefault {
		// GetColor returns ColorDefault for anything it cannot parse; treat
		// that as "unset" rather than silently rendering an invalid colour.
		return tcell.ColorDefault
	}
	return c
}

func registerBuiltinThemes() {
	themes["default"] = Theme{
		Name:        "default",
		Background:  tcell.ColorDefault,
		Accent:      tcell.PaletteColor(5),
		ActiveEnv:   tcell.PaletteColor(2),
		Border:      tcell.PaletteColor(8),
		Title:       tcell.PaletteColor(15),
		Dim:         tcell.PaletteColor(8),
		Dirty:       tcell.PaletteColor(3),
		Delete:      tcell.PaletteColor(1),
		Error:       tcell.PaletteColor(1),
		Text:        tcell.PaletteColor(15),
		SelectionBg: tcell.ColorDefault,
		SelectionFg: tcell.PaletteColor(5),
	}
	themes["cool"] = Theme{
		Name:        "cool",
		Background:  tcell.ColorDefault,
		Accent:      tcell.PaletteColor(6),
		ActiveEnv:   tcell.PaletteColor(2),
		Border:      tcell.PaletteColor(8),
		Title:       tcell.PaletteColor(15),
		Dim:         tcell.PaletteColor(8),
		Dirty:       tcell.PaletteColor(4),
		Delete:      tcell.PaletteColor(1),
		Error:       tcell.PaletteColor(1),
		Text:        tcell.PaletteColor(15),
		SelectionBg: tcell.ColorDefault,
		SelectionFg: tcell.PaletteColor(6),
	}
	themes["warm"] = Theme{
		Name:        "warm",
		Background:  tcell.ColorDefault,
		Accent:      tcell.PaletteColor(208),
		ActiveEnv:   tcell.PaletteColor(2),
		Border:      tcell.PaletteColor(8),
		Title:       tcell.PaletteColor(15),
		Dim:         tcell.PaletteColor(8),
		Dirty:       tcell.PaletteColor(3),
		Delete:      tcell.PaletteColor(1),
		Error:       tcell.PaletteColor(1),
		Text:        tcell.PaletteColor(15),
		SelectionBg: tcell.ColorDefault,
		SelectionFg: tcell.PaletteColor(208),
	}
	themes["mono"] = Theme{
		Name:        "mono",
		Background:  tcell.ColorDefault,
		Accent:      tcell.PaletteColor(15),
		ActiveEnv:   tcell.PaletteColor(7),
		Border:      tcell.PaletteColor(8),
		Title:       tcell.PaletteColor(15),
		Dim:         tcell.PaletteColor(8),
		Dirty:       tcell.PaletteColor(7),
		Delete:      tcell.PaletteColor(1),
		Error:       tcell.PaletteColor(1),
		Text:        tcell.PaletteColor(15),
		SelectionBg: tcell.ColorDefault,
		SelectionFg: tcell.PaletteColor(15),
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
	searchFg, searchBg := colorPairFromAny(file.Colors.SearchMatch, bg, accent)

	return Theme{
		Name:          name,
		Background:    parseColor(bg),
		Accent:        parseColor(accent),
		ActiveEnv:     parseColor(activeEnv),
		Border:        parseColor(border),
		Title:         parseColor(fg),
		Dim:           parseColor(dim),
		Dirty:         parseColor(warn),
		Delete:        parseColor(errColor),
		Error:         parseColor(errColor),
		Text:          parseColor(fg),
		SelectionBg:   parseColor(selBg),
		SelectionFg:   parseColor(selFg),
		SearchMatchFg: parseColor(searchFg),
		SearchMatchBg: parseColor(searchBg),
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

func colorPairFromAny(v any, fallbackFg, fallbackBg string) (string, string) {
	fg, bg := fallbackFg, fallbackBg
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) != "" {
			bg = strings.TrimSpace(t)
		}
	case map[string]any:
		if f, ok := t["fg"].(string); ok && strings.TrimSpace(f) != "" {
			fg = strings.TrimSpace(f)
		}
		if b, ok := t["bg"].(string); ok && strings.TrimSpace(b) != "" {
			bg = strings.TrimSpace(b)
		}
	}
	return fg, bg
}

func themeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func resolveTheme(name string) (Theme, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		name = defaultThemeName
	}
	theme, ok := themes[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(themeNames(), ", "))
	}
	return theme, nil
}

// CheckTheme reports whether name is a known TUI theme. An empty name is the default.
func CheckTheme(name string) error {
	_, err := resolveTheme(name)
	return err
}

func applyTheme(name string) error {
	theme, err := resolveTheme(name)
	if err != nil {
		return err
	}
	rebuildStyles(theme)
	return nil
}
