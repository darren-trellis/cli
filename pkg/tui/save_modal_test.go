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
	"strings"
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveModalCopyAndSize(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}, {Name: "BAR"}}

	view := m.renderSaveModal()
	assert.Contains(t, view, "Modified:")
	assert.Contains(t, view, "FOO")
	assert.Contains(t, view, "Save (y)")
	assert.Contains(t, view, "Cancel (n)")
	assert.NotContains(t, view, "The following secrets")
	assert.Less(t, strings.Count(view, "\n")+1, 12)
}

func TestSaveModalYConfirms(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}

	next, cmd := m.handleSaveKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mod := next.(Model)
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingChanges)
}

func TestSaveModalNAndEscCancel(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}

	next, cmd := m.handleSaveKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingChanges)

	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}
	next, cmd = m.handleSaveKey(tea.KeyMsg{Type: tea.KeyEsc})
	mod = next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingChanges)
}

func TestSaveModalEnterUsesHighlightedButton(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}
	m.modalBtnIdx = 1

	next, cmd := m.handleSaveKey(tea.KeyMsg{Type: tea.KeyEnter})
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingChanges)
}
