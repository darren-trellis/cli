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
	"testing"

	"github.com/charmbracelet/lipgloss"
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
[levels]
info = "#00ff00"
warn = "#f0c000"
error = { fg = "#ff0000", bg = "#200000" }
`))
	require.NoError(t, err)
	assert.Equal(t, "example", theme.Name)
	assert.Equal(t, lipgloss.Color("#ffcc00"), theme.Accent)
	assert.Equal(t, lipgloss.Color("#00ff00"), theme.ActiveEnv)
	assert.Equal(t, lipgloss.Color("#ff0000"), theme.Error)
	assert.Equal(t, lipgloss.Color("#f0c000"), theme.Dirty)

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
	assert.Equal(t, lipgloss.Color("#abcdef"), theme.ActiveEnv)
}
