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
	"gopkg.in/yaml.v3"
)

func TestEncodeKey(t *testing.T) {
	chord, ok := encodeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	require.True(t, ok)
	assert.Equal(t, "j", chord)

	chord, ok = encodeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	require.True(t, ok)
	assert.Equal(t, "G", chord)

	chord, ok = encodeKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.True(t, ok)
	assert.Equal(t, "C-c", chord)

	chord, ok = encodeKey(tea.KeyMsg{Type: tea.KeyUp})
	require.True(t, ok)
	assert.Equal(t, "up", chord)

	chord, ok = encodeKey(tea.KeyMsg{Type: tea.KeyPgDown})
	require.True(t, ok)
	assert.Equal(t, "pagedown", chord)

	chord, ok = encodeKey(tea.KeyMsg{Type: tea.KeySpace})
	require.True(t, ok)
	assert.Equal(t, "space", chord)
}

func TestMergeKeysUnbindAndOverlay(t *testing.T) {
	user := &models.TUIKeysOptions{
		Bindings: map[string]string{
			"q": "",
			"x": "quit",
		},
		Projects: map[string]string{
			"o": "",
		},
		Secrets: map[string]string{
			"o": "secret yank",
		},
	}
	keys := MergeKeys(user)
	_, ok := keys.Bindings["q"]
	assert.False(t, ok)
	assert.Equal(t, "quit", keys.Bindings["x"])

	_, ok = keys.Resolve(focusProjects, "o")
	assert.False(t, ok)

	cmd, ok := keys.Resolve(focusSecrets, "o")
	require.True(t, ok)
	assert.Equal(t, "secret yank", cmd)

	cmd, ok = keys.Resolve(focusProjects, "enter")
	require.True(t, ok)
	assert.Equal(t, "config load on", cmd)

	cmd, ok = keys.Resolve(focusProjects, "backspace")
	require.True(t, ok)
	assert.Equal(t, "config load off", cmd)

	cmd, ok = keys.Resolve(focusSecrets, "enter")
	require.True(t, ok)
	assert.Equal(t, "edit", cmd)
}

func TestResolveDefaultOverlays(t *testing.T) {
	keys := MergeKeys(nil)
	cmd, ok := keys.Resolve(focusSecrets, "o")
	require.True(t, ok)
	assert.Equal(t, "secret add", cmd)

	cmd, ok = keys.Resolve(focusProjects, "o")
	require.True(t, ok)
	assert.Equal(t, "config create", cmd)
}

func TestTUIKeysOptionsYAMLRoundTrip(t *testing.T) {
	raw := []byte(`
j: "nav down"
q: "quit"
projects:
  o: "config create"
secrets:
  o: "secret add"
`)
	var keys models.TUIKeysOptions
	require.NoError(t, yaml.Unmarshal(raw, &keys))
	assert.Equal(t, "nav down", keys.Bindings["j"])
	assert.Equal(t, "config create", keys.Projects["o"])
	assert.Equal(t, "secret add", keys.Secrets["o"])

	out, err := yaml.Marshal(&keys)
	require.NoError(t, err)
	var again models.TUIKeysOptions
	require.NoError(t, yaml.Unmarshal(out, &again))
	assert.Equal(t, keys.Bindings["j"], again.Bindings["j"])
	assert.Equal(t, keys.Projects["o"], again.Projects["o"])
}

func TestExecuteCommandNavAndRemap(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.width = 80
	m.height = 24
	m.secrets = make([]secretRow, 10)
	for i := range m.secrets {
		m.secrets[i] = newSecretRow(string(rune('A'+i)), "1", "masked")
	}
	m.secretIdx = 3

	next, _ := m.executeCommand("nav top")
	assert.Equal(t, 0, next.(Model).secretIdx)

	next, _ = next.(Model).executeCommand("nav bottom")
	assert.Equal(t, 9, next.(Model).secretIdx)

	m.keys = MergeKeys(&models.TUIKeysOptions{
		Bindings: map[string]string{"x": "nav top"},
	})
	m.secretIdx = 5
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, 0, next.(Model).secretIdx)
}

func TestHelpIncludesRemappedBinding(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{
		Border:  true,
		Sidebar: true,
		Keys: &models.TUIKeysOptions{
			Bindings: map[string]string{"x": "quit"},
		},
	})
	help := m.renderHelpText()
	assert.Contains(t, help, "x")
	assert.Contains(t, help, "Exit the TUI")
}

func TestCommandPrompt(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.width = 80
	m.height = 24

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mod := next.(Model)
	assert.Equal(t, focusCommand, mod.focus)

	mod.commandInput.SetValue("help")
	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod = next.(Model)
	assert.Equal(t, focusHelp, mod.focus)
}

func TestCommandModeRestoresSidebarFocus(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mod := next.(Model)
	assert.Equal(t, focusCommand, mod.focus)
	assert.Equal(t, focusProjects, mod.commandReturnFocus)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod = next.(Model)
	assert.Equal(t, focusProjects, mod.focus)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	mod = next.(Model)
	mod.commandInput.SetValue("help")
	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod = next.(Model)
	assert.Equal(t, focusHelp, mod.focus)
}
