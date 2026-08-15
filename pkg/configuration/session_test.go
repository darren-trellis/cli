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
package configuration

import (
	"path/filepath"
	"testing"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTempConfigDir(t *testing.T) {
	t.Helper()
	origDir := UserConfigDir
	origFile := UserConfigFile
	t.Cleanup(func() {
		UserConfigDir = origDir
		UserConfigFile = origFile
		UserFallbackDir = filepath.Join(origDir, "fallback")
		UserMetadataDir = UserFallbackDir
	})
	SetConfigDir(t.TempDir())
}

func TestShouldRestoreTUISession(t *testing.T) {
	setup := models.ScopedOptions{
		EnclaveProject: models.ScopedOption{Value: "api", Source: models.ConfigFileSource.String()},
		EnclaveConfig:  models.ScopedOption{Value: "dev", Source: models.ConfigFileSource.String()},
	}
	assert.True(t, ShouldRestoreTUISession(setup))

	flagged := setup
	flagged.EnclaveConfig.Source = models.FlagSource.String()
	assert.False(t, ShouldRestoreTUISession(flagged))

	env := setup
	env.EnclaveProject.Source = models.EnvironmentSource.String()
	assert.False(t, ShouldRestoreTUISession(env))
}

func TestApplyTUISession(t *testing.T) {
	opts := models.ScopedOptions{
		EnclaveProject: models.ScopedOption{Value: "api", Source: models.ConfigFileSource.String()},
		EnclaveConfig:  models.ScopedOption{Value: "dev", Source: models.ConfigFileSource.String()},
	}
	got := ApplyTUISession(opts, TUISession{Project: "web", Config: "prd"})
	assert.Equal(t, "web", got.EnclaveProject.Value)
	assert.Equal(t, "prd", got.EnclaveConfig.Value)
	assert.Equal(t, "TUI Session", got.EnclaveProject.Source)
}

func TestTUISessionRoundTripAndScopeInherit(t *testing.T) {
	withTempConfigDir(t)

	parent := filepath.Join(UserConfigDir, "repo")
	child := filepath.Join(parent, "pkg")
	saveTUISessionFor(parent, TUISession{Project: "api", Config: "dev"})

	got, ok := tuiSessionFor(child)
	require.True(t, ok)
	assert.Equal(t, "api", got.Project)
	assert.Equal(t, "dev", got.Config)

	saveTUISessionFor(child, TUISession{Project: "api", Config: "prd"})
	got, ok = tuiSessionFor(child)
	require.True(t, ok)
	assert.Equal(t, "prd", got.Config)

	got, ok = tuiSessionFor(parent)
	require.True(t, ok)
	assert.Equal(t, "dev", got.Config)
}
