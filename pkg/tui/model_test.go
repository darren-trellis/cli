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

	"github.com/DopplerHQ/cli/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSecretName(t *testing.T) {
	assert.Equal(t, "FOO_BAR", normalizeSecretName("foo bar"))
	assert.Equal(t, "A_B_1", normalizeSecretName("a-b.1"))
	assert.Equal(t, "ALREADY_OK", normalizeSecretName("ALREADY_OK"))
}

func TestFilterSecretIndexes(t *testing.T) {
	secrets := []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
		newSecretRow("GAMMA", "3", "masked"),
	}
	secrets[1].value = "changed"
	assert.True(t, secrets[1].isDirty())

	idxs := filterSecretIndexes(secrets, "a")
	assert.Equal(t, []int{0, 1, 2}, idxs)

	idxs = filterSecretIndexes(secrets, "BETA")
	assert.Equal(t, []int{1}, idxs)

	idxs = filterSecretIndexes(secrets, "zzz")
	assert.Equal(t, []int{1}, idxs)
}

func TestCollectChanges(t *testing.T) {
	secrets := []secretRow{
		newSecretRow("KEEP", "same", "masked"),
		newSecretRow("EDIT", "old", "masked"),
	}
	secrets[1].value = "new"
	del := newSecretRow("GONE", "x", "masked")
	del.shouldDelete = true
	secrets = append(secrets, del)

	changes := collectChanges(secrets)
	assert.Len(t, changes, 2)
	names := []string{changes[0].Name, changes[1].Name}
	assert.Contains(t, names, "EDIT")
	assert.Contains(t, names, "GONE")
}

func TestSecretRowUndoAndRestricted(t *testing.T) {
	s := newSecretRow("SECRET", "hidden", "restricted")
	assert.Equal(t, "[RESTRICTED]", s.displayValue())
	assert.False(t, s.isDirty())

	s.isTouched = true
	s.value = "plain"
	assert.True(t, s.isDirty())

	s.undo()
	assert.False(t, s.shouldDelete)
	assert.False(t, s.isTouched)
	assert.Equal(t, "[RESTRICTED]", s.displayValue())
}

func TestNavKeyFocusSwitch(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = false
	m.focus = focusSecrets

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	assert.Equal(t, focusProjects, next.(Model).focus)

	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	assert.Equal(t, focusSecrets, next.(Model).focus)
}

func TestMoveSecretsList(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("A", "1", "masked"),
		newSecretRow("B", "2", "masked"),
		newSecretRow("C", "3", "masked"),
	}
	m.secretIdx = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, next.(Model).secretIdx)

	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, next.(Model).secretIdx)
}

func TestEnterEditorAndEsc(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{newSecretRow("A", "val", "masked")}
	m.loadEditorFromSelection()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod := next.(Model)
	assert.Equal(t, focusEditorName, mod.focus)

	next, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusSecrets, next.(Model).focus)
}

func TestLoadedMsgSetsState(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	next, _ := m.Update(loadedMsg{
		projects:      []string{"p1", "p2"},
		configs:       buildConfigTree([]models.ConfigInfo{{Name: "dev", Environment: "dev", Root: true}}),
		secrets:       []secretRow{newSecretRow("X", "1", "masked")},
		activeProject: "p2",
		activeConfig:  "dev",
	})
	mod := next.(Model)
	assert.False(t, mod.fetching)
	assert.Equal(t, []string{"p1", "p2"}, mod.projects)
	assert.Equal(t, "p2", mod.activeProject)
	assert.True(t, mod.expanded["p2"])
	assert.Equal(t, "X", mod.nameInput.Value())
	require.NotEmpty(t, mod.tree)
	assert.Equal(t, treeConfig, mod.tree[mod.treeIdx].kind)
	assert.Equal(t, "dev", mod.tree[mod.treeIdx].config)
}

func TestProjectSelectedMsgLoadsSecrets(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = true
	m.focus = focusProjects

	next, _ := m.Update(projectSelectedMsg{
		configs: buildConfigTree([]models.ConfigInfo{
			{Name: "dev", Environment: "dev", Root: true},
			{Name: "dev_personal", Environment: "dev", Root: false},
		}),
		secrets: []secretRow{newSecretRow("FROM_OTHER", "1", "masked")},
		project: "backend-ts",
		config:  "dev_personal",
	})
	mod := next.(Model)
	assert.False(t, mod.fetching)
	assert.Equal(t, "backend-ts", mod.activeProject)
	assert.Equal(t, "dev_personal", mod.activeConfig)
	assert.True(t, mod.expanded["backend-ts"])
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Equal(t, "FROM_OTHER", mod.nameInput.Value())
}

func TestBuildConfigTree(t *testing.T) {
	rows := buildConfigTree([]models.ConfigInfo{
		{Name: "dev_feature", Environment: "dev", Root: false},
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "prd", Environment: "prd", Root: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
		{Name: "stg_only", Environment: "stg", Root: false},
	})

	assert.Equal(t, []string{"dev", "dev_feature", "dev_personal", "prd", "stg_only"}, configNames(rows))
	assert.Equal(t, 0, rows[0].depth)
	assert.True(t, rows[0].root)
	assert.Equal(t, 1, rows[1].depth)
	assert.False(t, rows[1].lastSibling)
	assert.True(t, rows[2].lastSibling)
	assert.Equal(t, 0, rows[4].depth) // no root for stg
}

func TestBuildProjectTreeFoldAndPinnedActive(t *testing.T) {
	configs := buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
		{Name: "prd", Environment: "prd", Root: true},
	})
	projectConfigs := map[string][]configRow{
		"api": configs,
	}
	expanded := map[string]bool{"api": true}

	tree := buildProjectTree([]string{"api", "web"}, projectConfigs, expanded, "api", "dev")
	assert.Equal(t, []treeKind{treeProject, treeConfig, treeConfig, treeConfig, treeProject}, treeKinds(tree))
	assert.Equal(t, []string{"api", "dev", "dev_personal", "prd", "web"}, treeLabels(tree))
	assert.Equal(t, 1, tree[1].depth)
	assert.False(t, tree[1].lastSibling)
	assert.Equal(t, 2, tree[2].depth)
	assert.True(t, tree[2].parentContinues)
	assert.True(t, tree[3].lastSibling)
	assert.Equal(t, "* ▾ api", ansi.Strip(formatTreeRow(tree[0], "api", "dev")))
	assert.Equal(t, "  ├─ *dev", ansi.Strip(formatTreeRow(tree[1], "api", "dev")))
	assert.Equal(t, "  │  └─ dev_personal", ansi.Strip(formatTreeRow(tree[2], "api", "dev")))
	assert.Equal(t, "  └─ prd", ansi.Strip(formatTreeRow(tree[3], "api", "dev")))
	assert.Contains(t, formatTreeRow(tree[1], "api", "dev"), "dev")

	expanded["api"] = false
	tree = buildProjectTree([]string{"api", "web"}, projectConfigs, expanded, "api", "dev")
	require.Len(t, tree, 3)
	assert.True(t, tree[0].folded)
	assert.Equal(t, treeConfig, tree[1].kind)
	assert.Equal(t, "dev", tree[1].config)
	assert.True(t, tree[1].pinned)
	assert.Equal(t, "  └─ *dev", ansi.Strip(formatTreeRow(tree[1], "api", "dev")))
	assert.Equal(t, "web", tree[2].project)
}

func TestToggleProjectFold(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api", "web"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.rebuildTree()
	m.treeIdx = 0

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.False(t, mod.expanded["api"])
	require.GreaterOrEqual(t, len(mod.tree), 2)
	assert.True(t, mod.tree[0].folded)
	assert.True(t, mod.tree[1].pinned)
	assert.Equal(t, "dev", mod.tree[1].config)

	next, cmd = mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	mod = next.(Model)
	assert.Nil(t, cmd)
	assert.True(t, mod.expanded["api"])
	assert.False(t, mod.tree[0].folded)
}

func TestCyclePane(t *testing.T) {
	m := newModel(models.ScopedOptions{})
	m.fetching = false
	m.focus = focusProjects
	m.secrets = []secretRow{newSecretRow("A", "1", "masked")}

	m.cyclePane(1)
	assert.Equal(t, focusSecrets, m.focus)
	m.cyclePane(1)
	assert.Equal(t, focusEditorName, m.focus)
	m.cyclePane(1)
	assert.Equal(t, focusEditorValue, m.focus)
	m.cyclePane(1)
	assert.Equal(t, focusFilter, m.focus)
	m.cyclePane(1)
	assert.Equal(t, focusProjects, m.focus)

	m.cyclePane(-1)
	assert.Equal(t, focusFilter, m.focus)
}

func configNames(rows []configRow) []string {
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = r.name
	}
	return names
}

func treeKinds(rows []treeRow) []treeKind {
	out := make([]treeKind, len(rows))
	for i, r := range rows {
		out[i] = r.kind
	}
	return out
}

func treeLabels(rows []treeRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		if r.kind == treeProject {
			out[i] = r.project
		} else {
			out[i] = r.config
		}
	}
	return out
}
