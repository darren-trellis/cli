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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigGetAndSet(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, applyTheme(defaultThemeName)) })

	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Theme: "default", Sidebar: true})

	mod, _ := m.executeCommand("config get theme")
	assert.Equal(t, "theme=default", mod.statusMsg)
	assert.Empty(t, mod.errMsg)

	mod, _ = m.executeCommand("config get")
	assert.Contains(t, mod.statusMsg, "theme=default")
	assert.Contains(t, mod.statusMsg, "sidebar=on")
	assert.Empty(t, mod.errMsg)

	mod, _ = m.executeCommand("config get nope")
	assert.Contains(t, mod.errMsg, "unknown setting")

	mod, _ = m.executeCommand("config set theme cool")
	assert.Equal(t, "theme=cool", mod.statusMsg)
	assert.Empty(t, mod.errMsg)
	assert.Equal(t, "cool", mod.cfg.Theme)
	assert.Equal(t, themes["cool"].Accent, accent)

	mod, _ = mod.executeCommand("config set theme nope")
	assert.Contains(t, mod.errMsg, "unknown theme")
	assert.Equal(t, "cool", mod.cfg.Theme)

	mod, _ = m.executeCommand("config set sidebar off")
	assert.Equal(t, "sidebar=off", mod.statusMsg)
	assert.False(t, mod.cfg.Sidebar)

	mod, _ = m.executeCommand("config set")
	assert.Equal(t, "usage: config set <name> [value]", mod.errMsg)

	mod, _ = m.executeCommand("config set sidebar-width")
	assert.Equal(t, "usage: config set sidebar-width <value>", mod.errMsg)
}

func TestConfigSetWithoutValueOpensPicker(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Theme: "default", Sidebar: true, Border: true})
	m.fetching = false
	m.width = 80
	m.height = 24

	mod, _ := m.executeCommand("config set sidebar")
	assert.Equal(t, focusConfigPick, mod.focus)
	assert.Equal(t, "sidebar", mod.configPickName)
	assert.Equal(t, []string{"on", "off"}, mod.configPickValues)
	assert.Equal(t, "on", mod.configPickValues[mod.configPickIdx])
	assert.Contains(t, modalText(mod), "● on")
	assert.Contains(t, modalText(mod), "Set sidebar")

	next, _ := mod.Update(runeKey('j'))
	mod = next
	assert.Equal(t, "off", mod.configPickValues[mod.configPickIdx])
	assert.False(t, mod.cfg.Sidebar, "the highlighted value is previewed")

	next, _ = mod.Update(namedKey(tcell.KeyEnter))
	mod = next
	assert.Equal(t, focusSecrets, mod.focus)
	assert.False(t, mod.cfg.Sidebar)
	assert.Equal(t, "sidebar=off", mod.statusMsg)
	assert.Empty(t, mod.configPickName)
}

func TestConfigPickCancelRestoresPreview(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, applyTheme(defaultThemeName)) })

	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Theme: "default", Sidebar: true})
	m.fetching = false
	m.width = 80
	m.height = 24
	require.NoError(t, applyTheme("default"))

	mod, _ := m.executeCommand("config set theme")
	require.Equal(t, focusConfigPick, mod.focus)
	start := mod.configPickIdx
	next, _ := mod.Update(runeKey('j'))
	mod = next
	require.NotEqual(t, start, mod.configPickIdx)
	assert.Equal(t, mod.configPickValues[mod.configPickIdx], mod.cfg.Theme)

	next, _ = mod.Update(namedKey(tcell.KeyEsc))
	mod = next
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Equal(t, "default", mod.cfg.Theme)
	assert.Equal(t, themes["default"].Accent, accent)
}

func TestConfigSettingSuggestions(t *testing.T) {
	items := suggestionsFor("config ")
	texts := suggestionTexts(items)
	assert.True(t, texts["get"])
	assert.True(t, texts["set"])
	assert.True(t, texts["create"])
	assert.False(t, texts["theme"])

	items = suggestionsFor("config get ")
	texts = suggestionTexts(items)
	assert.True(t, texts["theme"])
	assert.True(t, texts["sidebar"])
	assert.True(t, texts["case-mode"])
	assert.False(t, texts["create"])

	items = suggestionsFor("config set th")
	texts = suggestionTexts(items)
	assert.True(t, texts["theme"])
	assert.False(t, texts["sidebar"])

	items = suggestionsFor("config set theme ")
	texts = suggestionTexts(items)
	assert.True(t, texts["cool"])
	assert.True(t, texts["catppuccin"])
	assert.True(t, texts["default"])
	assert.False(t, texts["theme"])

	items = suggestionsFor("config set sidebar ")
	texts = suggestionTexts(items)
	assert.True(t, texts["on"])
	assert.True(t, texts["off"])
}
