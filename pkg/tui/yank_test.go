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
	"github.com/gdamore/tcell/v2"
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
	m.focus = focusProjects
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")
	m.secrets = []secretRow{newSecretRow("FOO", "bar", "masked")}

	next, _ := m.handleNavKey(runeKey('y'))
	mod := next
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "y", mod.statusMsg)

	next, _ = mod.handleNavKey(runeKey('n'))
	mod = next
	assert.False(t, mod.pendingYank)
	assert.Equal(t, "Copied name", mod.statusMsg)
}

func TestSecretsYStartsYankOperator(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{newSecretRow("FOO", "bar", "masked")}
	m.secretIdx = 0
	m.secretCol = colValue

	cmd, ok := m.keys.Resolve(focusSecrets, "y")
	require.True(t, ok)
	assert.Equal(t, "yank", cmd)

	next, _ := m.handleNavKey(runeKey('y'))
	mod := next
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "y", mod.statusMsg)
}

func TestSecretsYankCell(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{newSecretRow("FOO", "bar", "masked")}
	m.secretIdx = 0
	m.secretCol = colValue

	next, _ := m.handleNavKey(runeKey('y'))
	mod := next
	next, _ = mod.handleNavKey(runeKey('n'))
	mod = next
	assert.False(t, mod.pendingYank)
	assert.Equal(t, "Copied to clipboard", mod.statusMsg)
}

func TestYankOperatorStartsInSecrets(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets

	next, _ := m.executeCommand("yank")
	mod := next
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "y", mod.statusMsg)
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
	mod := next
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
	m.focus = focusProjects
	m.secrets = []secretRow{newSecretRow("A", "x", "masked")}

	next, _ := m.executeCommand("yank env")
	mod := next
	assert.Contains(t, mod.statusMsg, "Copied 1 secrets (env)")
}

func TestSecretYankWindow(t *testing.T) {
	lo, hi := secretYankWindow(0, 5, 1, 10)
	assert.Equal(t, 0, lo)
	assert.Equal(t, 4, hi)

	lo, hi = secretYankWindow(2, 3, -1, 10)
	assert.Equal(t, 0, lo)
	assert.Equal(t, 2, hi)

	lo, hi = secretYankWindow(8, 5, 1, 10)
	assert.Equal(t, 8, lo)
	assert.Equal(t, 9, hi)
}

func TestSecretOpWindowMotionIsInclusive(t *testing.T) {
	lo, hi := secretOpWindow(0, 2, 1, 10, false)
	assert.Equal(t, 0, lo)
	assert.Equal(t, 2, hi)

	lo, hi = secretOpWindow(0, 1, 1, 10, false)
	assert.Equal(t, 0, lo)
	assert.Equal(t, 1, hi)

	lo, hi = secretOpWindow(0, 1, 1, 10, true)
	assert.Equal(t, 0, lo)
	assert.Equal(t, 0, hi)

	lo, hi = secretOpWindow(3, 2, -1, 10, false)
	assert.Equal(t, 1, lo)
	assert.Equal(t, 3, hi)
}

func TestFormatSecretRowsPreservesOrder(t *testing.T) {
	rows := []secretRow{
		newSecretRow("ZED", "9", "masked"),
		newSecretRow("ALPHA", "1", "masked"),
	}
	env, err := formatSecretRows(rows, "env")
	require.NoError(t, err)
	assert.True(t, strings.Index(env, "ZED") < strings.Index(env, "ALPHA"))

	js, err := formatSecretRows(rows, "json")
	require.NoError(t, err)
	assert.True(t, strings.Index(js, "ZED") < strings.Index(js, "ALPHA"))
}

func TestSecretsYankCurrentLineYAML(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "y")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 1 secrets (yaml)")
}

func TestSecretsYankCountDown(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "5")
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "y5", mod.statusMsg)

	mod = sendKeys(mod, "down")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 6 secrets (yaml)")
}

func TestSecretsYankJSONLineAndRange(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "j")
	assert.True(t, mod.pendingYank)
	assert.Equal(t, "yj", mod.statusMsg)

	mod = sendKeys(mod, "y")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 1 secrets (json)")

	m = secretsYankModel()
	mod = sendKeys(m, "y", "j", "5", "down")
	assert.Contains(t, mod.statusMsg, "Copied 6 secrets (json)")
}

func TestSecretsYankEnvLine(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "e", "y")
	assert.Contains(t, mod.statusMsg, "Copied 1 secrets (env)")
}

func TestSecretsYankCountThenJIsMotion(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "3", "j")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 4 secrets (yaml)")
}

func TestSecretsYankTwoJCopiesThree(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "2", "j")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 3 secrets (yaml)")
}

func TestSecretsYankCountYIsLinewise(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "y", "5", "y")
	assert.False(t, mod.pendingYank)
	assert.Contains(t, mod.statusMsg, "Copied 5 secrets (yaml)")
}

func secretsYankModel() Model {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("A", "1", "masked"),
		newSecretRow("B", "2", "masked"),
		newSecretRow("C", "3", "masked"),
		newSecretRow("D", "4", "masked"),
		newSecretRow("E", "5", "masked"),
		newSecretRow("F", "6", "masked"),
	}
	m.secretIdx = 0
	return m
}

func sendKeys(m Model, chords ...string) Model {
	for _, chord := range chords {
		var msg keyMsg
		switch chord {
		case "down":
			msg = namedKey(tcell.KeyDown)
		case "up":
			msg = namedKey(tcell.KeyUp)
		default:
			msg = chordKey(chord)
		}
		next, _ := m.Update(msg)
		m = next
	}
	return m
}
