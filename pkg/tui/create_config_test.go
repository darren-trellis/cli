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

func TestInferEnvironmentFromConfigName(t *testing.T) {
	assert.Equal(t, "dev", inferEnvironmentFromConfigName("dev_personal"))
	assert.Equal(t, "stg", inferEnvironmentFromConfigName("stg_feature_x"))
	assert.Equal(t, "", inferEnvironmentFromConfigName("dev"))
	assert.Equal(t, "", inferEnvironmentFromConfigName("_personal"))
	assert.Equal(t, "", inferEnvironmentFromConfigName(""))
}

func TestBeginCreateConfigPrefillsEnvironment(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
	})
	m.expanded["api"] = true
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")

	next, _ := m.beginCreateConfig()
	mod := next.(Model)
	assert.Equal(t, focusCreateConfig, mod.focus)
	assert.Equal(t, "api", mod.createConfigProject)
	assert.Equal(t, "dev", mod.createConfigEnv)
	assert.Equal(t, "dev_", mod.createConfigInput.Value())
}

func TestCreateConfigKeybindingFromProjects(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.rebuildTree()
	m.treeIdx = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mod := next.(Model)
	assert.Equal(t, focusCreateConfig, mod.focus)
	assert.Equal(t, "api", mod.createConfigProject)
}

func TestSubmitCreateConfigRequiresEnvironment(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusCreateConfig
	m.createConfigProject = "api"
	m.createConfigEnv = ""
	m.createConfigInput.SetValue("orphan")

	next, cmd := m.submitCreateConfig()
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Contains(t, mod.errMsg, "environment")
	require.False(t, mod.fetching)
}

func TestSubmitCreateConfigInfersEnvironment(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusCreateConfig
	m.createConfigProject = "api"
	m.createConfigEnv = ""
	m.createConfigInput.SetValue("dev_feature")

	next, cmd := m.submitCreateConfig()
	mod := next.(Model)
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Equal(t, focusProjects, mod.focus)
	assert.Equal(t, "", mod.errMsg)
}

func TestBeginRenameConfig(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
	})
	m.expanded["api"] = true
	m.expandedEnvs[envKey("api", "dev")] = true
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev_personal")

	next, _ := m.beginRenameConfig()
	mod := next.(Model)
	assert.Equal(t, focusCreateConfig, mod.focus)
	assert.Equal(t, configPromptRename, mod.configPromptMode)
	assert.Equal(t, "api", mod.createConfigProject)
	assert.Equal(t, "dev_personal", mod.renameFromConfig)
	assert.Equal(t, "dev_personal", mod.createConfigInput.Value())
	assert.Equal(t, "~ ", mod.createConfigInput.Prompt)
}

func TestRenameConfigKeybindingFromProjects(t *testing.T) {
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

	next, _ := m.handleNavKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mod := next.(Model)
	assert.Equal(t, focusCreateConfig, mod.focus)
	assert.Equal(t, configPromptRename, mod.configPromptMode)
	assert.Equal(t, "dev", mod.renameFromConfig)
}

func TestSubmitRenameConfigNoopSameName(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusCreateConfig
	m.configPromptMode = configPromptRename
	m.createConfigProject = "api"
	m.renameFromConfig = "dev"
	m.createConfigInput.SetValue("dev")

	next, cmd := m.submitRenameConfig()
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusProjects, mod.focus)
	assert.Equal(t, configPromptCreate, mod.configPromptMode)
}
