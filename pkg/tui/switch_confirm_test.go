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
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivateSelectionPromptsWhenDirty(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "prd", Environment: "prd", Root: true},
	})
	m.expanded["api"] = true
	m.secrets = []secretRow{newSecretRow("FOO", "old", "masked")}
	m.secrets[0].value = "new"
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "prd")

	next, cmd := m.activateSelection()
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSwitchConfirm, mod.focus)
	assert.Equal(t, "api", mod.pendingSwitchProject)
	assert.Equal(t, "prd", mod.pendingSwitchConfig)
	assert.NotEmpty(t, mod.pendingChanges)
}

func TestSwitchConfirmEscCancels(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSwitchConfirm
	m.pendingSwitchProject = "api"
	m.pendingSwitchConfig = "prd"
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}

	next, _ := m.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyEsc})
	mod := next.(Model)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingSwitchProject)
	assert.Empty(t, mod.pendingSwitchConfig)
}

func TestSwitchConfirmDiscardStartsNavigation(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSwitchConfirm
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.pendingSwitchProject = "api"
	m.pendingSwitchConfig = "prd"

	next, cmd := m.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mod := next.(Model)
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingSwitchConfig)
}

func TestSwitchConfirmTabCyclesButtons(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSwitchConfirm
	m.modalBtnIdx = 0

	next, _ := m.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	mod := next.(Model)
	assert.Equal(t, 1, mod.modalBtnIdx)

	next, _ = mod.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	mod = next.(Model)
	assert.Equal(t, 2, mod.modalBtnIdx)

	next, _ = mod.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyTab})
	mod = next.(Model)
	assert.Equal(t, 0, mod.modalBtnIdx)

	next, _ = mod.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	mod = next.(Model)
	assert.Equal(t, 2, mod.modalBtnIdx)

	next, _ = mod.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyRight})
	mod = next.(Model)
	assert.Equal(t, 0, mod.modalBtnIdx)

	next, _ = mod.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyLeft})
	mod = next.(Model)
	assert.Equal(t, 2, mod.modalBtnIdx)
}

func TestSwitchConfirmMouseClicksDiscard(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.fetching = false
	m.focus = focusSwitchConfirm
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.pendingSwitchProject = "api"
	m.pendingSwitchConfig = "prd"
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}

	modal := m.renderSwitchConfirmModal()
	ox := (m.width - lipgloss.Width(modal)) / 2
	oy := (m.height - lipgloss.Height(modal)) / 2
	hits := buttonHitRects(modal, m.switchConfirmButtons(), ox, oy)
	require.Len(t, hits, 3)

	next, cmd := m.handleMouse(tea.MouseMsg{
		X:      hits[1].x,
		Y:      hits[1].y,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})
	mod := next.(Model)
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingSwitchConfig)
}

func TestSwitchConfirmCancelButton(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSwitchConfirm
	m.pendingSwitchProject = "api"
	m.pendingSwitchConfig = "prd"
	m.modalBtnIdx = 2

	next, cmd := m.handleSwitchConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingSwitchProject)
}

func TestSwitchConfirmModalCopyAndSize(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.focus = focusSwitchConfirm
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}, {Name: "BAR"}}

	view := m.renderSwitchConfirmModal()
	assert.NotContains(t, view, "You have unsaved")
	assert.Contains(t, view, "Modified:")
	assert.Contains(t, view, "FOO")
	assert.Contains(t, view, "Save (enter)")
	assert.Contains(t, view, "Discard (d)")
	assert.Contains(t, view, "Cancel (esc)")
	assert.Less(t, strings.Count(view, "\n")+1, 12)
}

func TestActivateSameConfigSkipsPrompt(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusProjects
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.projects = []string{"api"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.secrets = []secretRow{newSecretRow("FOO", "old", "masked")}
	m.secrets[0].value = "new"
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")

	next, cmd := m.activateSelection()
	mod := next.(Model)
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.NotEqual(t, focusSwitchConfirm, mod.focus)
}
