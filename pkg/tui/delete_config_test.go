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
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginDeleteConfigRejectsRoot(t *testing.T) {
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

	next, cmd := m.beginDeleteConfig()
	mod := next
	assert.Nil(t, cmd)
	assert.Equal(t, focusProjects, mod.focus)
	assert.Contains(t, mod.errMsg, "root")
}

func TestBeginDeleteConfigOpensConfirm(t *testing.T) {
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

	next, cmd := m.Update(runeKey('d'))
	mod := next
	assert.Nil(t, cmd)
	assert.Equal(t, focusDeleteConfirm, mod.focus)
	assert.Equal(t, "api", mod.pendingDeleteProject)
	assert.Equal(t, "dev_personal", mod.pendingDeleteConfig)
}

func TestDeleteConfirmEscCancels(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusDeleteConfirm
	m.pendingDeleteProject = "api"
	m.pendingDeleteConfig = "dev_personal"

	next, _ := m.handleDeleteConfirmKey(namedKey(tcell.KeyEsc))
	mod := next
	assert.Equal(t, focusProjects, mod.focus)
	assert.Empty(t, mod.pendingDeleteConfig)
}

func TestDeleteConfirmDStartsDelete(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusDeleteConfirm
	m.pendingDeleteProject = "api"
	m.pendingDeleteConfig = "dev_personal"

	next, cmd := m.handleDeleteConfirmKey(runeKey('d'))
	mod := next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingDeleteConfig)
}

func TestBeginDeleteProjectOpensConfirm(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api", "web"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeProject, "api", "")

	next, cmd := m.Update(runeKey('d'))
	mod := next
	assert.Nil(t, cmd)
	assert.Equal(t, focusDeleteConfirm, mod.focus)
	assert.Equal(t, "api", mod.pendingDeleteProject)
	assert.Empty(t, mod.pendingDeleteConfig)
	assert.True(t, mod.deletingProject())
	assert.Contains(t, modalText(mod), "Delete Project")
}

func TestProjectDeletedMsgRemovesProject(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api", "web"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.projectConfigs["web"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.activeProject = "web"
	m.activeConfig = "dev"
	m.rebuildTree()

	next, _ := m.Update(projectDeletedMsg{
		projects:  []string{"web"},
		deleted:   "api",
		highlight: "web",
	})
	mod := next
	assert.Equal(t, []string{"web"}, mod.projects)
	assert.Nil(t, mod.projectConfigs["api"])
	assert.Equal(t, "web", mod.activeProject)
	assert.Equal(t, "Deleted project api", mod.statusMsg)
	assert.Equal(t, findTreeIndex(mod.tree, treeProject, "web", ""), mod.treeIdx)
}

func TestPickProjectAfterDelete(t *testing.T) {
	assert.Equal(t, "web", pickProjectAfterDelete([]string{"web", "jobs"}, []string{"api", "web", "jobs"}, "api"))
	assert.Equal(t, "api", pickProjectAfterDelete([]string{"api"}, []string{"api", "web"}, "web"))
	assert.Equal(t, "", pickProjectAfterDelete(nil, []string{"api"}, "api"))
}

func TestPickConfigAfterDeletePrefersEnv(t *testing.T) {
	configs := buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "prd", Environment: "prd", Root: true},
	})
	assert.Equal(t, "dev", pickConfigAfterDelete(configs, "dev"))
	assert.Equal(t, "prd", pickConfigAfterDelete(configs, "prd"))
	assert.Equal(t, "dev", pickConfigAfterDelete(configs, "missing"))
}
