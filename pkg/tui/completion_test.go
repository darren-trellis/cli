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

func TestCommandClearClearsStatusBar(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.statusMsg = "Copied to clipboard"
	m.errMsg = "boom"
	m.searchQuery = "foo"
	m.searchRe = nil

	mod, _ := m.executeCommand("command clear")
	out := mod
	assert.Empty(t, out.statusMsg)
	assert.Empty(t, out.errMsg)
	assert.Empty(t, out.searchQuery)
}

func TestEscRunsCommandClear(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.width = 80
	m.height = 24
	m.fetching = false
	m.statusMsg = "hello"
	m.focus = focusSecrets

	mod, _ := m.handleNavKey(namedKey(tcell.KeyEsc))
	out := mod
	assert.Empty(t, out.statusMsg)
}

func TestSuggestionsPrefixAndSubcommand(t *testing.T) {
	items := suggestionsFor("hel")
	require.NotEmpty(t, items)
	assert.Equal(t, "help", items[0].Text)

	items = suggestionsFor("n")
	texts := suggestionTexts(items)
	assert.True(t, texts["nav"])
	assert.False(t, texts["up"])
	assert.False(t, texts["nav up"])

	items = suggestionsFor("")
	texts = suggestionTexts(items)
	assert.True(t, texts["nav"])
	assert.True(t, texts["config"])
	assert.False(t, texts["up"])
	assert.False(t, texts["lock"])
	assert.False(t, texts["create"])

	items = suggestionsFor("nav ")
	require.NotEmpty(t, items)
	texts = suggestionTexts(items)
	assert.True(t, texts["up"])
	assert.True(t, texts["down"])
	assert.True(t, texts["top"])
	assert.True(t, texts["page"])
	assert.False(t, texts["page up"])
	assert.False(t, texts["page down"])

	items = suggestionsFor("nav page")
	texts = suggestionTexts(items)
	assert.True(t, texts["up"])
	assert.True(t, texts["down"])
	assert.False(t, texts["page"])
	assert.False(t, texts["page up"])

	items = suggestionsFor("nav page ")
	texts = suggestionTexts(items)
	assert.True(t, texts["up"])
	assert.True(t, texts["down"])
	assert.False(t, texts["page up"])
}

func TestSuggestionsHideNestedUntilParentTyped(t *testing.T) {
	items := suggestionsFor("config")
	texts := suggestionTexts(items)
	assert.True(t, texts["create"] || texts["lock"] || texts["load"])
	assert.False(t, texts["on"])
	assert.False(t, texts["off"])
	assert.False(t, texts["toggle"])

	items = suggestionsFor("config ")
	texts = suggestionTexts(items)
	assert.True(t, texts["create"])
	assert.True(t, texts["rename"])
	assert.True(t, texts["lock"])
	assert.True(t, texts["load"])
	assert.True(t, texts["delete"])
	assert.True(t, texts["get"])
	assert.True(t, texts["set"])
	assert.False(t, texts["on"])
	assert.False(t, texts["off"])
	assert.False(t, texts["toggle"])
	assert.False(t, texts["lock toggle"])
	assert.False(t, texts["unlock"])

	items = suggestionsFor("config lock")
	texts = suggestionTexts(items)
	assert.True(t, texts["on"])
	assert.True(t, texts["off"])
	assert.True(t, texts["toggle"])
	assert.False(t, texts["create"])

	items = suggestionsFor("config lock ")
	texts = suggestionTexts(items)
	assert.True(t, texts["on"])
	assert.True(t, texts["off"])
	assert.True(t, texts["toggle"])

	items = suggestionsFor("config load ")
	texts = suggestionTexts(items)
	assert.True(t, texts["on"])
	assert.True(t, texts["off"])
	assert.True(t, texts["toggle"])
}

func suggestionTexts(items []Suggestion) map[string]bool {
	texts := map[string]bool{}
	for _, s := range items {
		texts[s.Label] = true
		texts[s.Text] = true
	}
	return texts
}

func TestTabCompleteAppliesSuggestion(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.beginCommand()
	m.commandInput.SetValue("qui")
	m.commandInput.CursorEnd()
	m.refreshCompletions()
	require.NotEmpty(t, m.completions.Items)

	m.tabComplete(true)
	assert.Equal(t, "quit", m.commandInput.Value())
}

func TestCommandModeShowsCompletions(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.width = 80
	m.height = 24
	m.beginCommand()
	require.NotEmpty(t, m.completions.Items)
	assert.Greater(t, m.completions.DesiredHeight(20), 0)

	view := renderModel(t, m)
	assert.Contains(t, view, "suggestions")
	assert.NotContains(t, view, "nav up")
}
