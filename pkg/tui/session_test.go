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
)

func TestCurrentSessionRoundTrip(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.activeProject = "api"
	m.activeConfig = "dev_personal"
	m.expandedEnvs[envKey("api", "dev")] = false
	m.expandedEnvs[envKey("api", "prd")] = true

	session := m.currentSession()
	assert.Equal(t, "api", session.Project)
	assert.Equal(t, "dev_personal", session.Config)
	assert.Equal(t, map[string]bool{"api/dev": false, "api/prd": true}, session.ExpandedEnvs)

	restored := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	restored.applySession(session)
	assert.False(t, isEnvExpanded(restored.expandedEnvs, "api", "dev"))
	assert.True(t, isEnvExpanded(restored.expandedEnvs, "api", "prd"))
}

func TestPersistSessionRequiresEnable(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{})
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.persistSession()
}
