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
	"io/fs"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTheme(t *testing.T) {
	require.NoError(t, applyTheme("cool"))
	assert.Equal(t, themes["cool"].Accent, accent)

	require.NoError(t, applyTheme("DEFAULT"))
	assert.Equal(t, themes["default"].Accent, accent)

	err := applyTheme("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown theme")

	assert.NoError(t, CheckTheme("cool"))
	assert.NoError(t, CheckTheme(""))
	assert.Error(t, CheckTheme("nope"))
}

func TestTeleminatorThemesLoaded(t *testing.T) {
	names := themeNames()
	assert.Contains(t, names, "catppuccin")
	assert.Contains(t, names, "tokyo-night")
	assert.Contains(t, names, "nord")
	assert.Contains(t, names, "gruvbox")
	assert.Contains(t, names, "default")

	require.NoError(t, applyTheme("catppuccin"))
	assert.Equal(t, themes["catppuccin"].Accent, accent)
	assert.Equal(t, themes["catppuccin"].Background, background)
	assert.Equal(t, tcell.GetColor("#1e1e2e"), themes["catppuccin"].SearchMatchFg)
	assert.Equal(t, tcell.GetColor("#f9e2af"), themes["catppuccin"].SearchMatchBg)

	require.NoError(t, applyTheme("tokyo-night"))
	assert.Equal(t, themes["tokyo-night"].Accent, accent)
}

func TestParseTeleminatorTheme(t *testing.T) {
	theme, err := parseTeleminatorTheme([]byte(`
name = "example"
[colors]
background = "#111111"
foreground = "#eeeeee"
selection_bg = "#222222"
selection_fg = "#ffffff"
border = "#333333"
window_focus_border = "#ffcc00"
dim = "#666666"
search_match = { fg = "#111111", bg = "#ffcc00" }
[levels]
info = "#00ff00"
warn = "#f0c000"
error = { fg = "#ff0000", bg = "#200000" }
`))
	require.NoError(t, err)
	assert.Equal(t, "example", theme.Name)
	assert.Equal(t, tcell.GetColor("#ffcc00"), theme.Accent)
	assert.Equal(t, tcell.GetColor("#00ff00"), theme.ActiveEnv)
	assert.Equal(t, tcell.GetColor("#ff0000"), theme.Error)
	assert.Equal(t, tcell.GetColor("#f0c000"), theme.Dirty)
	assert.Equal(t, tcell.GetColor("#111111"), theme.SearchMatchFg)
	assert.Equal(t, tcell.GetColor("#ffcc00"), theme.SearchMatchBg)

	theme, err = parseTeleminatorTheme([]byte(`
name = "override"
[colors]
foreground = "#eeeeee"
window_focus_border = "#ffcc00"
active_env = "#abcdef"
[levels]
info = "#00ff00"
`))
	require.NoError(t, err)
	assert.Equal(t, tcell.GetColor("#abcdef"), theme.ActiveEnv)
}

func TestEmbeddedThemesAllParse(t *testing.T) {
	entries, err := fs.ReadDir(embeddedThemeFS, "themes")
	require.NoError(t, err)

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		count++
		data, err := embeddedThemeFS.ReadFile("themes/" + entry.Name())
		require.NoError(t, err, entry.Name())

		theme, err := parseTeleminatorTheme(data)
		require.NoError(t, err, entry.Name())

		// The file name is the name users type for --theme, so it must match.
		assert.Equal(t, strings.TrimSuffix(entry.Name(), ".toml"), theme.Name)
		assert.Contains(t, themes, theme.Name)

		// Every theme needs enough colour to render a legible pane.
		for name, c := range map[string]tcell.Color{
			"background":   theme.Background,
			"text":         theme.Text,
			"accent":       theme.Accent,
			"dim":          theme.Dim,
			"selection_bg": theme.SelectionBg,
			"error":        theme.Error,
		} {
			assert.NotEqual(t, tcell.ColorDefault, c, "%s: %s unset", entry.Name(), name)
		}

		require.NoError(t, applyTheme(theme.Name), entry.Name())
	}
	assert.Greater(t, count, 40, "embedded themes should still be present")
	require.NoError(t, applyTheme(defaultThemeName))
}
