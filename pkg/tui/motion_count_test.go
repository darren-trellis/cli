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
)

func TestMotionCountMovesList(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("A", "1", "masked"),
		newSecretRow("B", "2", "masked"),
		newSecretRow("C", "3", "masked"),
		newSecretRow("D", "4", "masked"),
		newSecretRow("E", "5", "masked"),
	}
	m.secretIdx = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	mod := next.(Model)
	assert.Equal(t, 3, mod.motionCount)
	assert.Equal(t, 0, mod.secretIdx)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mod = next.(Model)
	assert.Equal(t, 0, mod.motionCount)
	assert.Equal(t, 3, mod.secretIdx)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	mod = next.(Model)
	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	mod = next.(Model)
	assert.Equal(t, 1, mod.secretIdx)
}

func TestMotionCountGJumpsToRow(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("A", "1", "masked"),
		newSecretRow("B", "2", "masked"),
		newSecretRow("C", "3", "masked"),
		newSecretRow("D", "4", "masked"),
	}
	m.secretIdx = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	mod := next.(Model)
	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	mod = next.(Model)
	assert.Equal(t, 1, mod.secretIdx)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	mod = next.(Model)
	assert.Equal(t, 3, mod.secretIdx)
}
