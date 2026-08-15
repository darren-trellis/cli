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

func TestFormatSecretsCopy(t *testing.T) {
	secrets := map[string]string{
		"BETA":  "2",
		"ALPHA": "1",
	}

	env, err := formatSecretsCopy(secrets, "env")
	require.NoError(t, err)
	assert.Contains(t, env, `ALPHA="1"`)
	assert.Contains(t, env, `BETA="2"`)
	assert.True(t, strings.Index(env, "ALPHA") < strings.Index(env, "BETA"))

	js, err := formatSecretsCopy(secrets, "json")
	require.NoError(t, err)
	assert.Contains(t, js, `"ALPHA"`)
	assert.Contains(t, js, `"1"`)

	ym, err := formatSecretsCopy(secrets, "yaml")
	require.NoError(t, err)
	assert.Contains(t, ym, "ALPHA:")
	assert.Contains(t, ym, "BETA:")
}

func TestYankOperatorSequence(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("FOO", "bar", "masked"),
	}
	m.secretIdx = 0

	next, _ := m.handleNavKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mod := next.(Model)
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "y", mod.statusMsg)

	next, _ = mod.handleNavKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	mod = next.(Model)
	assert.False(t, mod.pendingYank)
	assert.Equal(t, "Copied name", mod.statusMsg)
}

func TestYankNameConfig(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")

	next, _ := m.yankName()
	mod := next.(Model)
	assert.Equal(t, "Copied name", mod.statusMsg)
}

func TestYankYamlSkipsRestricted(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("OK", "1", "masked"),
		newSecretRow("SECRET", "hidden", "restricted"),
	}

	secrets, skipped := m.secretsMapForCopy()
	assert.Equal(t, 1, len(secrets))
	assert.Equal(t, 1, skipped)
	assert.Equal(t, "1", secrets["OK"])
}

func TestYankCommandDirect(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{newSecretRow("A", "x", "masked")}

	next, _ := m.executeCommand("yank env")
	mod := next.(Model)
	assert.Contains(t, mod.statusMsg, "Copied 1 secrets (env)")
}
