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
	tea "github.com/charmbracelet/bubbletea"
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
	out := mod.(Model)
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

	mod, _ := m.handleNavKey(tea.KeyMsg{Type: tea.KeyEsc})
	out := mod.(Model)
	assert.Empty(t, out.statusMsg)
}

func TestSuggestionsPrefixAndSubcommand(t *testing.T) {
	items := suggestionsFor("hel")
	require.NotEmpty(t, items)
	assert.Equal(t, "help", items[0].Text)

	items = suggestionsFor("nav ")
	require.NotEmpty(t, items)
	texts := map[string]bool{}
	for _, s := range items {
		texts[s.Text] = true
	}
	assert.True(t, texts["up"])
	assert.True(t, texts["down"])
	assert.True(t, texts["top"])
}

func TestTabCompleteAppliesSuggestion(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.beginCommand()
	m.commandInput.SetValue("qui")
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

	view := m.View()
	assert.Contains(t, view, "completions")
}
