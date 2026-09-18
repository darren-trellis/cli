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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsDStartsDeleteOperator(t *testing.T) {
	m := secretsYankModel()
	cmd, ok := m.keys.Resolve(focusSecrets, "d")
	require.True(t, ok)
	assert.Equal(t, "delete", cmd)

	mod := sendKeys(m, "d")
	assert.True(t, mod.pendingSecretDelete)
	assert.Equal(t, "d", mod.statusMsg)
}

func TestSecretsDDDeletesCurrentLine(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "d", "d")
	assert.False(t, mod.pendingSecretDelete)
	assert.Equal(t, 1, markedDeleteCount(mod))
	assert.True(t, mod.secrets[0].shouldDelete)
	assert.Contains(t, mod.statusMsg, "Deleted 1 secret")
}

func TestSecretsD2JDeletesThree(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "d", "2", "j")
	assert.False(t, mod.pendingSecretDelete)
	assert.Equal(t, 3, markedDeleteCount(mod))
	assert.True(t, mod.secrets[0].shouldDelete)
	assert.True(t, mod.secrets[1].shouldDelete)
	assert.True(t, mod.secrets[2].shouldDelete)
	assert.False(t, mod.secrets[3].shouldDelete)
	assert.Contains(t, mod.statusMsg, "Deleted 3 secrets")
}

func TestSecretsDJDeletesTwo(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "d", "j")
	assert.Equal(t, 2, markedDeleteCount(mod))
}

func TestSecretsCountDDIsLinewise(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "2", "d")
	assert.True(t, mod.pendingSecretDelete)
	assert.Equal(t, "d2", mod.statusMsg)

	mod = sendKeys(mod, "d")
	assert.False(t, mod.pendingSecretDelete)
	assert.Equal(t, 2, markedDeleteCount(mod))
	assert.True(t, mod.secrets[0].shouldDelete)
	assert.True(t, mod.secrets[1].shouldDelete)
	assert.False(t, mod.secrets[2].shouldDelete)
}

func TestSecretsD5DDeletesFive(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "d", "5", "d")
	assert.Equal(t, 5, markedDeleteCount(mod))
}

func TestSecretsDKDeletesInclusiveUp(t *testing.T) {
	m := secretsYankModel()
	m.secretIdx = 2
	mod := sendKeys(m, "d", "k")
	assert.Equal(t, 2, markedDeleteCount(mod))
	assert.False(t, mod.secrets[0].shouldDelete)
	assert.True(t, mod.secrets[1].shouldDelete)
	assert.True(t, mod.secrets[2].shouldDelete)
	assert.Equal(t, 1, mod.secretIdx)
}

func TestSecretsDeleteCommandDeletesCurrent(t *testing.T) {
	m := secretsYankModel()
	next, _ := m.executeCommand("secret delete")
	mod := next
	assert.False(t, mod.pendingSecretDelete)
	assert.Equal(t, 1, markedDeleteCount(mod))
	assert.True(t, mod.secrets[0].shouldDelete)
}

func TestSecretsDDRemovesUnsavedRow(t *testing.T) {
	m := secretsYankModel()
	m.secrets[1] = newEmptySecretRow()
	m.secrets[1].name = "NEW"
	m.secretIdx = 1

	mod := sendKeys(m, "d", "d")
	assert.Equal(t, 5, len(mod.secrets))
	assert.Equal(t, 0, markedDeleteCount(mod))
	assert.Equal(t, "A", mod.secrets[0].name)
	assert.Equal(t, "C", mod.secrets[1].name)
}

func TestSecretsDeleteInvalidKeyCancels(t *testing.T) {
	m := secretsYankModel()
	mod := sendKeys(m, "d", "x")
	assert.False(t, mod.pendingSecretDelete)
	assert.Equal(t, 0, markedDeleteCount(mod))
	assert.Contains(t, mod.errMsg, "delete:")
}

func TestProjectsDStillDeletesConfig(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	cmd, ok := m.keys.Resolve(focusProjects, "d")
	require.True(t, ok)
	assert.Equal(t, "delete", cmd)
}

func TestSecretsDeleteRangeRemovesUnsavedHighToLow(t *testing.T) {
	m := secretsYankModel()
	m.secrets[1] = newEmptySecretRow()
	m.secrets[1].name = "NEW"
	mod := sendKeys(m, "d", "2", "j")
	assert.Equal(t, 5, len(mod.secrets))
	assert.True(t, mod.secrets[0].shouldDelete)
	assert.Equal(t, "C", mod.secrets[1].name)
	assert.True(t, mod.secrets[1].shouldDelete)
	assert.False(t, mod.secrets[2].shouldDelete)
}

func markedDeleteCount(m Model) int {
	n := 0
	for _, s := range m.secrets {
		if s.shouldDelete {
			n++
		}
	}
	return n
}
