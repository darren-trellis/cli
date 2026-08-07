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

func TestSearchSecretsNextPrevAndClear(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
		newSecretRow("GAMMA", "3", "masked"),
		newSecretRow("ALPACA", "4", "masked"),
	}

	m.beginSearch()
	assert.Equal(t, focusSearch, m.focus)
	m.searchInput.SetValue("ALP")
	require.NoError(t, m.compileSearch("ALP"))
	m.setFocus(focusSecrets)
	assert.Equal(t, []int{0, 3}, m.searchMatches)
	assert.Equal(t, 0, m.secretIdx)

	m.stepSearchMatch(1)
	assert.Equal(t, 3, m.secretIdx)
	assert.Equal(t, 1, m.searchMatchIdx)

	m.stepSearchMatch(1)
	assert.Equal(t, 0, m.secretIdx)

	m.stepSearchMatch(-1)
	assert.Equal(t, 3, m.secretIdx)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod := next.(Model)
	assert.Nil(t, mod.searchRe)
	assert.Equal(t, "", mod.searchQuery)
}

func TestSearchProjectsRegex(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusProjects
	m.projects = []string{"api", "web", "admin"}
	m.projectConfigs["api"] = buildConfigTree([]models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
	})
	m.expanded["api"] = true
	m.rebuildTree()

	m.beginSearch()
	require.NoError(t, m.compileSearch("a"))
	assert.Equal(t, focusProjects, m.searchPane)
	assert.Contains(t, m.searchMatches, 0) // api project
	assert.NotEmpty(t, m.searchMatches)

	err := m.compileSearch("[")
	require.Error(t, err)
}

func TestFilterKeybinding(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	assert.Equal(t, focusFilter, next.(Model).focus)

	m = newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.Equal(t, focusSearch, next.(Model).focus)
}
