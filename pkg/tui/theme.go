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
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const defaultThemeName = "default"

type Theme struct {
	Name   string
	Accent lipgloss.Color
	Title  lipgloss.Color
	Dim    lipgloss.Color
	Dirty  lipgloss.Color
	Delete lipgloss.Color
	Error  lipgloss.Color
	Text   lipgloss.Color
}

var themes = map[string]Theme{
	"default": {
		Name:   "default",
		Accent: lipgloss.Color("5"),  // magenta
		Title:  lipgloss.Color("15"), // white
		Dim:    lipgloss.Color("8"),
		Dirty:  lipgloss.Color("3"), // yellow
		Delete: lipgloss.Color("1"), // red
		Error:  lipgloss.Color("1"),
		Text:   lipgloss.Color("15"),
	},
	"cool": {
		Name:   "cool",
		Accent: lipgloss.Color("6"), // cyan
		Title:  lipgloss.Color("15"),
		Dim:    lipgloss.Color("8"),
		Dirty:  lipgloss.Color("4"), // blue
		Delete: lipgloss.Color("1"),
		Error:  lipgloss.Color("1"),
		Text:   lipgloss.Color("15"),
	},
	"warm": {
		Name:   "warm",
		Accent: lipgloss.Color("208"), // orange
		Title:  lipgloss.Color("15"),
		Dim:    lipgloss.Color("8"),
		Dirty:  lipgloss.Color("3"),
		Delete: lipgloss.Color("1"),
		Error:  lipgloss.Color("1"),
		Text:   lipgloss.Color("15"),
	},
	"mono": {
		Name:   "mono",
		Accent: lipgloss.Color("15"),
		Title:  lipgloss.Color("15"),
		Dim:    lipgloss.Color("8"),
		Dirty:  lipgloss.Color("7"),
		Delete: lipgloss.Color("1"),
		Error:  lipgloss.Color("1"),
		Text:   lipgloss.Color("15"),
	},
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
