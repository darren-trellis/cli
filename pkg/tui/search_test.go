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
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

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
	mod := next.(Model)
	assert.Equal(t, focusFilter, mod.focus)
	assert.False(t, mod.filterGlobal)

	m = newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	mod = next.(Model)
	assert.Equal(t, focusFilter, mod.focus)
	assert.True(t, mod.filterGlobal)

	m = newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.Equal(t, focusSearch, next.(Model).focus)
}

func TestSecretMatchesSearchUsesFullValue(t *testing.T) {
	long := strings.Repeat("x", 50) + "NEEDLE" + strings.Repeat("y", 10)
	s := newSecretRow("TOKEN", long, "masked")
	re := regexp.MustCompile("NEEDLE")
	assert.True(t, secretMatchesSearch(s, re))
	assert.False(t, secretMatchesSearch(s, regexp.MustCompile("missing")))

	restricted := newSecretRow("SECRET", "hidden", "restricted")
	assert.False(t, secretMatchesSearch(restricted, regexp.MustCompile("hidden")))
	assert.True(t, secretMatchesSearch(restricted, regexp.MustCompile("SECRET")))
}

func TestGlobalSearchFindsCachedConfigs(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("PRD", false))
	assert.True(t, m.searchGlobal)
	require.Len(t, m.globalHits, 1)
	assert.Equal(t, "api", m.globalHits[0].project)
	assert.Equal(t, "prd", m.globalHits[0].config)
	assert.Equal(t, "PRD", m.globalHits[0].name)
	assert.Equal(t, "dev", m.activeConfig)
}

func TestGlobalSearchNextJumpsConfig(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.putSecretsCache("api", "dev", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("SHARED", "a", "masked")},
	})
	m.putSecretsCache("api", "prd", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("SHARED", "b", "masked")},
	})
	m.secrets = []secretRow{newSecretRow("SHARED", "a", "masked")}
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("SHARED", true))
	require.Len(t, m.globalHits, 2)
	assert.Equal(t, "dev", m.activeConfig)

	cmd := m.stepSearchMatch(1)
	assert.Nil(t, cmd)
	assert.Equal(t, "prd", m.activeConfig)
	assert.Equal(t, "SHARED", m.secrets[0].name)
	assert.Equal(t, 1, m.searchMatchIdx)
}

func TestGlobalSearchKeybinding(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets

	cmd, ok := m.keys.Resolve(focusSecrets, "C-f")
	require.True(t, ok)
	assert.Equal(t, "search global", cmd)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	mod := next.(Model)
	assert.Equal(t, focusSearch, mod.focus)
	assert.True(t, mod.searchGlobal)
	assert.Equal(t, "g/ ", mod.searchInput.Prompt)
}

func TestWorkplaceSearchDoesNotDeadlockWithManyProjects(t *testing.T) {
	projects := make([]string, 16)
	known := map[string][]configRow{}
	cached := map[string][]secretRow{}
	for i := range projects {
		p := fmt.Sprintf("p%d", i)
		projects[i] = p
		known[p] = []configRow{{name: "dev"}}
		cached[secretsCacheKey(p, "dev")] = []secretRow{newSecretRow("FOO", "bar", "masked")}
	}

	cmd := workplaceSearchCmd(models.ScopedOptions{}, 1, "FOO", "insensitive", projects, known, cached)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		got, ok := msg.(workplaceSearchMsg)
		require.True(t, ok)
		assert.NoError(t, got.err)
		assert.Len(t, got.hits, 16)
	case <-time.After(2 * time.Second):
		t.Fatal("workplace search deadlocked")
	}
}

func TestWorkplaceSearchIgnoresStaleGen(t *testing.T) {
	m := cachedSidebarModel()
	m.searchGlobal = true
	m.searchGen = 3
	m.fetching = true
	next, _ := m.applyWorkplaceSearch(workplaceSearchMsg{
		gen:  2,
		hits: []globalHit{{project: "api", config: "prd", name: "PRD"}},
	})
	mod := next.(Model)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.globalHits)
}
