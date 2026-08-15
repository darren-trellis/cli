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

func TestInputHistoryPrevNextAndPrefix(t *testing.T) {
	h := newInputHistory([]string{"help", "nav down", "config delete"})

	got, ok := h.Prev("")
	require.True(t, ok)
	assert.Equal(t, "config delete", got)
	got, ok = h.Prev("")
	require.True(t, ok)
	assert.Equal(t, "nav down", got)
	got, ok = h.Next("")
	require.True(t, ok)
	assert.Equal(t, "config delete", got)
	got, ok = h.Next("")
	require.True(t, ok)
	assert.Equal(t, "", got)

	h.Reset()
	got, ok = h.Prev("nav")
	require.True(t, ok)
	assert.Equal(t, "nav down", got)
	_, ok = h.Prev("nav")
	assert.False(t, ok)
}

func TestInputHistoryPushDedupesAndCaps(t *testing.T) {
	h := newInputHistory([]string{"a", "b"})
	h.Push("a")
	assert.Equal(t, []string{"b", "a"}, h.Items())

	items := make([]string, 0, maxInputHistory+5)
	for i := 0; i < maxInputHistory+5; i++ {
		items = append(items, string(rune('a'+i%26))+string(rune('0'+i%10)))
	}
	h = newInputHistory(nil)
	for _, item := range items {
		h.Push(item)
	}
	assert.LessOrEqual(t, len(h.Items()), maxInputHistory)
}

func TestCommandHistoryUpDown(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.commandHistory = newInputHistory([]string{"help", "sidebar toggle"})
	m.beginCommand()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod := next.(Model)
	assert.Equal(t, "sidebar toggle", mod.commandInput.Value())

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod = next.(Model)
	assert.Equal(t, "help", mod.commandInput.Value())

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod = next.(Model)
	assert.Equal(t, "sidebar toggle", mod.commandInput.Value())

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod = next.(Model)
	assert.Equal(t, "", mod.commandInput.Value())
}

func TestCommandDownFocusesSuggestions(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.commandHistory = newInputHistory([]string{"help"})
	m.beginCommand()
	require.NotEmpty(t, m.completions.Items)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod := next.(Model)
	assert.True(t, mod.completions.Browsed)
	require.NotNil(t, mod.completions.Selected)
	assert.Equal(t, 0, *mod.completions.Selected)
	assert.Equal(t, "", mod.commandInput.Value())

	first := *mod.completions.Selected
	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod = next.(Model)
	require.NotNil(t, mod.completions.Selected)
	assert.Equal(t, first+1, *mod.completions.Selected)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod = next.(Model)
	require.NotNil(t, mod.completions.Selected)
	assert.Equal(t, first, *mod.completions.Selected)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod = next.(Model)
	assert.False(t, mod.completions.Browsed)
	assert.Nil(t, mod.completions.Selected)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod = next.(Model)
	assert.Equal(t, "help", mod.commandInput.Value())
}

func TestSearchHistoryUpAppliesQuery(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
	}
	m.searchHistory = newInputHistory([]string{"BETA"})
	m.beginSearch()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod := next.(Model)
	assert.Equal(t, "BETA", mod.searchInput.Value())
	assert.Equal(t, []int{1}, mod.searchMatches)
}

func TestCommandEnterRecordsHistory(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.beginCommand()
	m.commandInput.SetValue("help")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod := next.(Model)
	assert.Equal(t, []string{"help"}, mod.commandHistory.Items())
	assert.Equal(t, focusHelp, mod.focus)
}
