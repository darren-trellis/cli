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

func TestParseSecretsClipboardJSON(t *testing.T) {
	m, format, err := parseSecretsClipboard(`{"FOO":"bar","BAZ":1}`)
	require.NoError(t, err)
	assert.Equal(t, "json", format)
	assert.Equal(t, "bar", m["FOO"])
	assert.Equal(t, "1", m["BAZ"])
}

func TestParseSecretsClipboardYAML(t *testing.T) {
	m, format, err := parseSecretsClipboard("FOO: bar\nHELLO: world\n")
	require.NoError(t, err)
	assert.Equal(t, "yaml", format)
	assert.Equal(t, "bar", m["FOO"])
	assert.Equal(t, "world", m["HELLO"])
}

func TestParseSecretsClipboardEnv(t *testing.T) {
	m, format, err := parseSecretsClipboard("FOO=\"bar\"\nBAZ=qux\n")
	require.NoError(t, err)
	assert.Equal(t, "env", format)
	assert.Equal(t, "bar", m["FOO"])
	assert.Equal(t, "qux", m["BAZ"])
}

func TestApplyPastedSecretsAddAndUpdate(t *testing.T) {
	mod := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	mod.secrets = []secretRow{
		newSecretRow("EXISTING", "old", "masked"),
	}

	added, updated := mod.applyPastedSecrets(map[string]string{
		"EXISTING": "new",
		"NEW":      "val",
	})
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, updated)
	assert.Equal(t, "new", mod.secrets[0].value)
	assert.True(t, mod.secrets[0].isDirty())
	require.Len(t, mod.secrets, 2)
	assert.Equal(t, "NEW", mod.secrets[1].name)
	assert.True(t, mod.secrets[1].isDirty())
}

func TestPasteRejectedOutsideSecrets(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.focus = focusProjects
	next, _ := m.pasteSecrets()
	mod := next
	assert.Contains(t, mod.errMsg, "Secrets")
}
