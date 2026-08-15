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
	"regexp"
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enableTestColors(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

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

func TestLiveSearchHighlightsWhileTyping(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
		newSecretRow("ALPACA", "4", "masked"),
	}

	m.beginSearch()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	m = next.(Model)
	assert.Equal(t, focusSearch, m.focus)
	require.NotNil(t, m.searchRe)
	assert.Equal(t, []int{0, 1, 2}, m.searchMatches) // ALPHA, BETA, ALPACA
	assert.Equal(t, 0, m.secretIdx)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = next.(Model)
	assert.Equal(t, focusSearch, m.focus)
	assert.Equal(t, "AL", m.searchInput.Value())
	assert.Equal(t, []int{0, 2}, m.searchMatches)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	m = next.(Model)
	assert.Equal(t, "ALP", m.searchInput.Value())
	assert.Equal(t, []int{0, 2}, m.searchMatches)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = next.(Model)
	assert.Equal(t, focusSearch, m.focus)
	assert.Empty(t, m.searchMatches)
	assert.NotNil(t, m.searchRe)
}

func TestHighlightMatchesOnlyMatchedText(t *testing.T) {
	enableTestColors(t)
	re := regexp.MustCompile(`(?i)alp`)
	base := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	hit := lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15"))
	old := searchHitStyle
	searchHitStyle = hit
	t.Cleanup(func() { searchHitStyle = old })

	out := highlightMatches("ALPHA_KEY", re, base)
	assert.Equal(t, "ALPHA_KEY", ansi.Strip(out))
	assert.Contains(t, out, hit.Render("ALP"))
	assert.NotContains(t, out, hit.Render("ALPHA_KEY"))

	selected := lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("5"))
	selectedOut := highlightMatches("ALPHA_KEY", re, selected)
	assert.Equal(t, "ALPHA_KEY", ansi.Strip(selectedOut))
	assert.Contains(t, selectedOut, hit.Render("ALP"))
	assert.Contains(t, selectedOut, selected.Render("HA_KEY"))
}

func TestHighlightNeedleInDisplay(t *testing.T) {
	enableTestColors(t)
	re := regexp.MustCompile(`dev`)
	hit := lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15"))
	old := searchHitStyle
	searchHitStyle = hit
	t.Cleanup(func() { searchHitStyle = old })

	out := highlightNeedleInDisplay("  ├─ ▾ *dev", "dev", re, baseTextStyle())
	assert.Equal(t, "  ├─ ▾ *dev", ansi.Strip(out))
	assert.Contains(t, out, hit.Render("dev"))
	assert.NotContains(t, out, hit.Render("  ├─ ▾ *dev"))
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
