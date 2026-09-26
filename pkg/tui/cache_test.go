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

func cachedSidebarModel() Model {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
		{Name: "prd", Environment: "prd", Root: true},
	})
	m.expanded["api"] = true
	m.secrets = []secretRow{newSecretRow("DEV", "1", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.putSecretsCache("api", "prd", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("PRD", "9", "masked")},
	})
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")
	return m
}

func TestHighlightCachedConfigShowsSecrets(t *testing.T) {
	m := cachedSidebarModel()
	m.moveList(findTreeIndex(m.tree, treeConfig, "api", "prd") - m.treeIdx)

	assert.Equal(t, "prd", m.activeConfig)
	require.Len(t, m.secrets, 1)
	assert.Equal(t, "PRD", m.secrets[0].name)
	assert.True(t, m.configIsCached("api", "dev"))
}

func TestHighlightUncachedConfigKeepsCurrentSecrets(t *testing.T) {
	m := cachedSidebarModel()
	m.moveList(findTreeIndex(m.tree, treeConfig, "api", "dev_personal") - m.treeIdx)

	assert.Equal(t, "dev", m.activeConfig)
	require.Len(t, m.secrets, 1)
	assert.Equal(t, "DEV", m.secrets[0].name)
	assert.False(t, m.configIsCached("api", "dev_personal"))
}

func TestHighlightUncachedConfigLoadsAfterDelay(t *testing.T) {
	m := cachedSidebarModel()
	require.Equal(t, focusProjects, m.focus)

	next, cmd := m.Update(runeKey('j'))
	mod := next
	require.True(t, mod.highlightIs("api", "dev_personal"))
	require.NotNil(t, cmd, "landing on an unloaded config schedules a load")
	assert.Equal(t, "dev", mod.activeConfig, "the load is delayed, not immediate")

	due := highlightLoadDueMsg{seq: mod.highlightLoadSeq, project: "api", config: "dev_personal"}
	next, cmd = mod.Update(due)
	mod = next
	require.NotNil(t, cmd)
	assert.False(t, mod.fetching, "background loads do not block input")
	assert.Equal(t, secretsCacheKey("api", "dev_personal"), mod.highlightLoading)

	next, _ = mod.Update(highlightLoadedMsg{
		seq:     due.seq,
		project: "api",
		config:  "dev_personal",
		secrets: []secretRow{newSecretRow("PERSONAL", "1", "masked")},
	})
	mod = next
	assert.Empty(t, mod.highlightLoading)
	assert.Equal(t, "dev_personal", mod.activeConfig)
	assert.Equal(t, "PERSONAL", mod.secrets[0].name)
	assert.True(t, mod.configIsCached("api", "dev"))
}

func TestHighlightLoadIgnoresStaleTick(t *testing.T) {
	m := cachedSidebarModel()
	next, _ := m.Update(runeKey('j'))
	mod := next
	stale := highlightLoadDueMsg{seq: mod.highlightLoadSeq - 1, project: "api", config: "dev_personal"}

	next, cmd := mod.Update(stale)
	assert.Nil(t, cmd)
	assert.Empty(t, next.highlightLoading)
}

func TestHighlightLoadAfterMovingAwayOnlyCaches(t *testing.T) {
	m := cachedSidebarModel()
	next, _ := m.Update(runeKey('j'))
	mod := next
	seq := mod.highlightLoadSeq
	next, _ = mod.Update(runeKey('k'))
	mod = next
	require.True(t, mod.highlightIs("api", "dev"))

	next, _ = mod.Update(highlightLoadedMsg{
		seq:     seq,
		project: "api",
		config:  "dev_personal",
		secrets: []secretRow{newSecretRow("PERSONAL", "1", "masked")},
	})
	mod = next
	assert.Equal(t, "dev", mod.activeConfig)
	assert.True(t, mod.configIsCached("api", "dev_personal"))
}

func TestSecretsLoadedKeepsSidebarFocus(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusProjects
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")

	next, _ := m.Update(secretsLoadedMsg{
		secrets:       []secretRow{newSecretRow("PRD", "9", "masked")},
		activeProject: "api",
		activeConfig:  "prd",
	})
	mod := next
	assert.Equal(t, focusProjects, mod.focus)
	assert.Equal(t, "prd", mod.activeConfig)
}

func TestActivateCachedConfigDoesNotFetch(t *testing.T) {
	m := cachedSidebarModel()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")

	next, cmd := m.activateSelection()
	mod := next
	assert.Nil(t, cmd)
	assert.Equal(t, focusProjects, mod.focus)
	assert.Equal(t, "prd", mod.activeConfig)
	assert.Equal(t, "PRD", mod.secrets[0].name)
}

func TestStepCachedConfigSkipsUncached(t *testing.T) {
	m := cachedSidebarModel()
	m.stepCachedConfig(1)
	assert.Equal(t, "prd", m.activeConfig)
	assert.Equal(t, "PRD", m.secrets[0].name)

	m.stepCachedConfig(1)
	assert.Equal(t, "dev", m.activeConfig)
	assert.Equal(t, "DEV", m.secrets[0].name)
}

func TestSearchNextWithoutQueryHopsCached(t *testing.T) {
	m := cachedSidebarModel()
	next, _ := m.executeCommand("search next")
	mod := next
	assert.Equal(t, "prd", mod.activeConfig)
}

func TestSearchNextWithQueryKeepsMatches(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.searchPane = focusSecrets
	require.NoError(t, m.applySearch("DEV", true))

	next, _ := m.executeCommand("search next")
	mod := next
	assert.Equal(t, "dev", mod.activeConfig)
	assert.Contains(t, mod.statusMsg, "Match")
}

func TestDirtyCacheSurvivesSwitch(t *testing.T) {
	m := cachedSidebarModel()
	m.secrets[0].value = "changed"
	m.stashCurrentSecrets()
	require.True(t, m.applyCachedSecrets("api", "prd"))
	assert.Equal(t, "PRD", m.secrets[0].name)
	assert.True(t, m.configIsDirty("api", "dev"))
	assert.False(t, m.configIsDirty("api", "prd"))
	assert.True(t, m.hasDirtySecrets())
}

func TestGlobalFilterSurvivesConfigSwitch(t *testing.T) {
	m := cachedSidebarModel()
	m.filter = "DEV"
	m.globalFilter = "TOKEN"
	m.stashCurrentSecrets()
	m.putSecretsCache("api", "prd", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("PRD", "9", "masked")},
		filter:  "PRD",
	})
	require.True(t, m.applyCachedSecrets("api", "prd"))
	assert.Equal(t, "PRD", m.filter)
	assert.Equal(t, "TOKEN", m.globalFilter)
}

func TestSidebarMarksCachedConfigs(t *testing.T) {
	m := cachedSidebarModel()
	m.width = 80
	m.height = 24
	out := renderSidebar(t, m)
	assert.Contains(t, out, "dev")
	assert.Contains(t, out, "prd")
	assert.NotContains(t, out, "◉")
	assert.NotContains(t, out, "○")

	prd := m.tree[findTreeIndex(m.tree, treeConfig, "api", "prd")]
	personal := m.tree[findTreeIndex(m.tree, treeConfig, "api", "dev_personal")]
	assert.Equal(t, cachedConfigStyle, m.treeRowStyle(prd, false))
	assert.Equal(t, baseStyle, m.treeRowStyle(personal, false))
}

func TestBackspaceUnloadsCachedConfig(t *testing.T) {
	m := cachedSidebarModel()
	m.width = 80
	m.height = 24
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")
	m.revealHighlightedConfig()
	require.True(t, m.configIsCached("api", "prd"))

	next, _ := m.Update(namedKey(tcell.KeyBackspace2))
	mod := next
	assert.False(t, mod.configIsCached("api", "prd"))
	assert.True(t, mod.configIsCached("api", "dev"))
	assert.Equal(t, "dev", mod.activeConfig)
	assert.Equal(t, "DEV", mod.secrets[0].name)
	assert.Equal(t, findTreeIndex(mod.tree, treeConfig, "api", "prd"), mod.treeIdx)
	assert.Contains(t, mod.statusMsg, "Unloaded api / prd")
	prd := mod.tree[findTreeIndex(mod.tree, treeConfig, "api", "prd")]
	assert.False(t, prd.cached)
}

func TestBackspaceUnloadsLastCachedConfig(t *testing.T) {
	m := cachedSidebarModel()
	m.dropSecretsCache("api", "prd")
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")

	next, _ := m.unloadHighlightedConfig()
	mod := next
	assert.False(t, mod.configIsCached("api", "dev"))
	assert.Empty(t, mod.activeConfig)
	assert.Empty(t, mod.secrets)
	assert.Equal(t, findTreeIndex(mod.tree, treeConfig, "api", "dev"), mod.treeIdx)
}

func TestBackspaceRefusesDirtyCachedConfig(t *testing.T) {
	m := cachedSidebarModel()
	m.secrets[0].value = "changed"
	m.stashCurrentSecrets()

	next, _ := m.unloadHighlightedConfig()
	mod := next
	assert.True(t, mod.configIsCached("api", "dev"))
	assert.Equal(t, "dev", mod.activeConfig)
	assert.Contains(t, mod.errMsg, "unsaved")
}

func TestBackspaceOnUncachedConfigDoesNothing(t *testing.T) {
	m := cachedSidebarModel()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev_personal")

	next, _ := m.unloadHighlightedConfig()
	mod := next
	assert.True(t, mod.configIsCached("api", "dev"))
	assert.Equal(t, "dev", mod.activeConfig)
	assert.Contains(t, mod.errMsg, "not loaded")
}

func TestConfigLoadCommands(t *testing.T) {
	m := cachedSidebarModel()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")

	next, _ := m.executeCommand("config load off")
	mod := next
	assert.False(t, mod.configIsCached("api", "prd"))

	next, _ = mod.executeCommand("config load on")
	mod = next
	assert.True(t, mod.fetching || mod.activeConfig == "prd")

	m = cachedSidebarModel()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")
	next, _ = m.executeCommand("config load toggle")
	mod = next
	assert.False(t, mod.configIsCached("api", "prd"))

	next, _ = mod.executeCommand("config load toggle")
	mod = next
	assert.True(t, mod.fetching || mod.configIsCached("api", "prd") || mod.activeConfig == "prd")
}

func TestNKeyHopsCachedWhenNotSearching(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	next, _ := m.Update(runeKey('n'))
	mod := next
	assert.Equal(t, "prd", mod.activeConfig)
}

func TestNKeyInProjectsPromptsForBranch(t *testing.T) {
	m := cachedSidebarModel()
	require.True(t, m.highlightIs("api", "dev"))

	next, _ := m.Update(runeKey('n'))
	mod := next
	assert.Equal(t, focusCreateConfig, mod.focus)
	assert.Equal(t, configPromptCreate, mod.configPromptMode)
	assert.Equal(t, "+ dev_", mod.createConfigInput.Prompt)
	assert.Equal(t, "", mod.createConfigInput.Value())

	for _, r := range "ci" {
		next, _ = mod.Update(runeKey(r))
		mod = next
	}
	next, cmd := mod.Update(namedKey(tcell.KeyEnter))
	mod = next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.createConfigPrefix)
}

func TestBranchPromptName(t *testing.T) {
	m := cachedSidebarModel()
	next, _ := m.beginBranchConfig()
	mod := next
	assert.Equal(t, "dev_", mod.createConfigPrefix)
	assert.Equal(t, "dev", mod.createConfigEnv)

	mod.createConfigInput.SetValue("ci")
	assert.Equal(t, "dev_ci", mod.createConfigName())
	mod.createConfigInput.SetValue("dev_ci")
	assert.Equal(t, "dev_ci", mod.createConfigName(), "an already-prefixed name is not doubled")

	mod.treeIdx = findTreeIndex(mod.tree, treeProject, "api", "")
	next, _ = mod.beginBranchConfig()
	assert.Equal(t, "Select a config to branch from", next.errMsg)
}

func TestNKeyInProjectsStillStepsSearch(t *testing.T) {
	m := cachedSidebarModel()
	m.searchPane = focusProjects
	require.NoError(t, m.applySearch("prd", false))

	next, _ := m.Update(runeKey('n'))
	assert.Equal(t, focusProjects, next.focus)
	assert.True(t, next.highlightIs("api", "prd"))
}

func sidebarRowClick(m Model, idx int) mouseMsg {
	layout := m.computeLayout()
	visible := max(1, layout.projects.h-m.panelChrome())
	start := clampScrollOffset(m.treeOffset, m.treeIdx, visible, len(m.tree))
	return leftClick(layout.projects.x+2, layout.projects.y+1+(idx-start))
}

func TestSidebarSingleClickSelectsWithoutLoading(t *testing.T) {
	m := cachedSidebarModel()
	m.width = 80
	m.height = 24
	m.projects = []string{"api", "web"}
	m.rebuildTree()
	webIdx := findTreeIndex(m.tree, treeProject, "web", "")
	cfgIdx := findTreeIndex(m.tree, treeConfig, "api", "dev_personal")

	next, cmd := m.handleMouse(sidebarRowClick(m, webIdx))
	mod := next
	assert.Nil(t, cmd)
	assert.False(t, mod.fetching)
	assert.Equal(t, webIdx, mod.treeIdx)
	assert.Equal(t, "api", mod.activeProject)

	next, cmd = mod.handleMouse(sidebarRowClick(mod, cfgIdx))
	mod = next
	assert.Nil(t, cmd)
	assert.False(t, mod.fetching)
	assert.Equal(t, cfgIdx, mod.treeIdx)
	assert.Equal(t, "dev", mod.activeConfig)
}

func TestSidebarDoubleClickLoads(t *testing.T) {
	m := cachedSidebarModel()
	m.width = 80
	m.height = 24
	m.projects = []string{"api", "web"}
	m.rebuildTree()
	webIdx := findTreeIndex(m.tree, treeProject, "web", "")
	click := sidebarRowClick(m, webIdx)

	next, cmd := m.handleMouse(click)
	mod := next
	assert.Nil(t, cmd)
	assert.False(t, mod.fetching)

	next, cmd = mod.handleMouse(click)
	mod = next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
}
