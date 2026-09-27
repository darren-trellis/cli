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
	"github.com/gdamore/tcell/v2"
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

	next, _ := m.Update(namedKey(tcell.KeyEsc))
	mod := next
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
	next, _ := m.Update(runeKey('A'))
	m = next
	assert.Equal(t, focusSearch, m.focus)
	require.NotNil(t, m.searchRe)
	assert.Equal(t, []int{0, 1, 2}, m.searchMatches) // ALPHA, BETA, ALPACA
	assert.Equal(t, 0, m.secretIdx)

	next, _ = m.Update(runeKey('L'))
	m = next
	assert.Equal(t, focusSearch, m.focus)
	assert.Equal(t, "AL", m.searchInput.Value())
	assert.Equal(t, []int{0, 2}, m.searchMatches)

	next, _ = m.Update(runeKey('P'))
	m = next
	assert.Equal(t, "ALP", m.searchInput.Value())
	assert.Equal(t, []int{0, 2}, m.searchMatches)

	next, _ = m.Update(runeKey('Z'))
	m = next
	assert.Equal(t, focusSearch, m.focus)
	assert.Empty(t, m.searchMatches)
	assert.NotNil(t, m.searchRe)
}

func TestSearchSpansCoverOnlyMatchedText(t *testing.T) {
	re := regexp.MustCompile(`(?i)alp`)

	spans := matchRuneSpans("ALPHA_KEY", re)
	require.Len(t, spans, 1)
	assert.Equal(t, runeSpan{0, 3}, spans[0])

	// Only the matched runes are inside a span.
	for i := 0; i < 3; i++ {
		assert.True(t, inSpans(spans, i), "rune %d should be highlighted", i)
	}
	for i := 3; i < len([]rune("ALPHA_KEY")); i++ {
		assert.False(t, inSpans(spans, i), "rune %d should not be highlighted", i)
	}

	assert.Empty(t, matchRuneSpans("ALPHA_KEY", regexp.MustCompile(`zzz`)))
}

func TestSpansForNeedleOffsetsPastTreeDecoration(t *testing.T) {
	re := regexp.MustCompile(`dev`)
	display := "├──◇ dev"

	spans := spansForNeedle(display, "dev", re)
	require.Len(t, spans, 1)

	// The span must land on "dev" at the end, not on the tree glyphs.
	runes := []rune(display)
	assert.Equal(t, "dev", string(runes[spans[0].start:spans[0].end]))
	assert.False(t, inSpans(spans, 0))

	// A needle that is not present highlights nothing.
	assert.Nil(t, spansForNeedle(display, "nope", re))
	assert.Nil(t, spansForNeedle(display, "dev", nil))
}

func TestFilterKeybinding(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets

	next, _ := m.Update(runeKey('f'))
	mod := next
	assert.Equal(t, focusFilter, mod.focus)
	assert.False(t, mod.filterGlobal)

	m = newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	next, _ = m.Update(runeKey('F'))
	mod = next
	assert.Equal(t, focusFilter, mod.focus)
	assert.True(t, mod.filterGlobal)

	m = newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true, ListScrollbarVertical: true, SidebarScrollbarVertical: true})
	m.fetching = false
	m.focus = focusSecrets
	next, _ = m.Update(runeKey('/'))
	assert.Equal(t, focusSearch, next.focus)
	assert.True(t, next.searchGlobal)
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

	cmd, ok = m.keys.Resolve(focusSecrets, "/")
	require.True(t, ok)
	assert.Equal(t, "search global", cmd)

	next, _ := m.Update(namedKey(tcell.KeyCtrlF))
	mod := next
	assert.Equal(t, focusSearch, mod.focus)
	assert.True(t, mod.searchGlobal)
	assert.Equal(t, "g/ ", mod.searchInput.Prompt)
	assert.Equal(t, "", mod.searchQuery)
	assert.Equal(t, "", mod.searchInput.Value())
}

func projectFilterModel() Model {
	m := cachedSidebarModel()
	m.width = 90
	m.height = 24
	m.projects = []string{"api", "billing", "web-api"}
	m.rebuildTree()
	return m
}

func TestSlashInProjectsFiltersProjects(t *testing.T) {
	m := projectFilterModel()
	require.Equal(t, focusProjects, m.focus)

	next, _ := m.Update(runeKey('/'))
	mod := next
	assert.Equal(t, focusFilter, mod.focus)
	assert.True(t, mod.filterProjects)
	assert.False(t, mod.searchGlobal, "does not start the workplace secret search")
	assert.Equal(t, "/ ", mod.filterInput.Prompt)

	for _, r := range "api" {
		next, _ = mod.Update(runeKey(r))
		mod = next
	}
	assert.Equal(t, 2, mod.visibleProjectCount())
	assert.NotContains(t, treeLabels(mod.tree), "billing")
	assert.Equal(t, []string{"DEV"}, filteredSecretNames(mod), "secrets are not filtered")
	assert.Contains(t, renderSidebar(t, mod), "/api (2)")

	next, _ = mod.Update(namedKey(tcell.KeyEnter))
	mod = next
	assert.Equal(t, focusProjects, mod.focus)
	assert.Equal(t, "api", mod.projectFilter)
	assert.Contains(t, renderSidebar(t, mod), "/api (2)")
	assert.Equal(t, "Projects (2) /api", mod.projectsTitle(40), "a wide sidebar keeps the full title")
}

func TestProjectFilterEscClears(t *testing.T) {
	m := projectFilterModel()
	next, _ := m.Update(runeKey('/'))
	mod := next
	next, _ = mod.Update(runeKey('w'))
	mod = next
	require.Equal(t, 1, mod.visibleProjectCount())

	next, _ = mod.Update(namedKey(tcell.KeyEsc))
	mod = next
	assert.Equal(t, focusProjects, mod.focus)
	assert.Empty(t, mod.projectFilter)
	assert.Equal(t, 3, mod.visibleProjectCount())
	assert.Contains(t, renderSidebar(t, mod), "Projects (3)")
	assert.NotContains(t, renderSidebar(t, mod), "/")
}

func TestProjectFilterStartsEmpty(t *testing.T) {
	m := projectFilterModel()
	m.projectFilter = "bill"
	m.rebuildTree()
	require.Equal(t, 1, m.visibleProjectCount())

	next, _ := m.Update(runeKey('/'))
	assert.Equal(t, "", next.filterInput.Value())
	assert.Equal(t, 3, next.visibleProjectCount())
}

func TestGlobalSearchStartsEmpty(t *testing.T) {
	m := cachedSidebarModel()
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("PRD", false))
	require.NotEmpty(t, m.searchQuery)
	require.NotEmpty(t, m.globalHits)

	m.setFocus(focusSecrets)
	next, _ := m.Update(runeKey('/'))
	assert.Equal(t, focusSearch, next.focus)
	assert.True(t, next.searchGlobal)
	assert.Equal(t, "", next.searchQuery)
	assert.Equal(t, "", next.searchInput.Value())
	assert.Nil(t, next.searchRe)
	assert.Empty(t, next.globalHits)
	assert.Equal(t, []string{"DEV"}, filteredSecretNames(next))
	assert.Greater(t, next.visibleProjectCount(), 0)
}

func TestWorkplaceIndexSkipsCachedNames(t *testing.T) {
	projects := make([]string, 16)
	known := map[string][]configRow{}
	haveNames := map[string]bool{}
	for i := range projects {
		p := fmt.Sprintf("p%d", i)
		projects[i] = p
		known[p] = []configRow{{name: "dev"}}
		haveNames[secretsCacheKey(p, "dev")] = true
	}

	cmd := workplaceIndexCmd(models.ScopedOptions{}, projects, known, haveNames, false)
	done := make(chan Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		got, ok := msg.(workplaceSearchMsg)
		require.True(t, ok)
		assert.NoError(t, got.err)
		assert.Empty(t, got.names)
	case <-time.After(2 * time.Second):
		t.Fatal("workplace index deadlocked")
	}
}

func TestWorkplaceIndexMergesNamesAndRefilters(t *testing.T) {
	m := cachedSidebarModel()
	m.searchGlobal = true
	m.workplaceIndexing = true
	m.searchQuery = "PRD"
	require.NoError(t, m.applySearch("PRD", false))
	assert.Equal(t, 1, len(m.globalHits))

	next, _ := m.applyWorkplaceSearch(workplaceSearchMsg{
		names: map[string][]string{
			secretsCacheKey("web", "dev"): {"PRD_TOKEN", "OTHER"},
		},
		configs: map[string][]configRow{
			"web": {{name: "dev"}},
		},
	})
	mod := next
	assert.False(t, mod.workplaceIndexing)
	assert.True(t, mod.namesIndexDone)
	assert.False(t, mod.fetching)
	assert.GreaterOrEqual(t, len(mod.globalHits), 2)
	assert.Contains(t, treeProjectNames(mod), "api")
	assert.Contains(t, treeProjectNames(mod), "web")
}

func TestGlobalSearchMatchesNamesNotValues(t *testing.T) {
	m := cachedSidebarModel()
	m.secrets = []secretRow{newSecretRow("TOKEN", "hidden-value", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("hidden-value", false))
	assert.Empty(t, m.globalHits)

	require.NoError(t, m.applySearch("TOKEN", false))
	require.NotEmpty(t, m.globalHits)
	assert.Equal(t, "TOKEN", m.globalHits[0].name)
}

func TestGlobalSearchFiltersSecretsAndHidesProjects(t *testing.T) {
	m := cachedSidebarModel()
	m.projects = []string{"api", "web"}
	m.projectConfigs["web"] = []configRow{{name: "dev"}}
	m.putSecretsCache("web", "dev", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("OTHER", "x", "masked")},
	})
	m.secrets = []secretRow{
		newSecretRow("DEV", "1", "masked"),
		newSecretRow("TOKEN", "2", "masked"),
		newSecretRow("SHARED", "3", "masked"),
	}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.rebuildTree()

	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("TOKEN", false))

	assert.Equal(t, []string{"TOKEN"}, filteredSecretNames(m))
	assert.Equal(t, []string{"api"}, treeProjectNames(m))
	assert.Equal(t, []string{"dev"}, treeConfigNames(m))
	assert.Equal(t, 1, m.visibleProjectCount())

	m.clearSearch()
	assert.Equal(t, []string{"DEV", "TOKEN", "SHARED"}, filteredSecretNames(m))
	assert.Equal(t, []string{"api", "web"}, treeProjectNames(m))
}

func TestLocalSearchKeepsNonMatchingSecrets(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secrets = []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
		newSecretRow("ALPACA", "4", "masked"),
	}
	m.beginSearch()
	require.NoError(t, m.compileSearch("ALP"))
	assert.Equal(t, []string{"ALPHA", "BETA", "ALPACA"}, filteredSecretNames(m))
	assert.Equal(t, []int{0, 2}, m.searchMatches)
}

func TestGlobalSearchHidesNonMatchingConfigs(t *testing.T) {
	m := cachedSidebarModel()
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("PRD", false))
	assert.Equal(t, []string{"api"}, treeProjectNames(m))
	assert.Equal(t, []string{"prd"}, treeConfigNames(m))
	assert.Empty(t, filteredSecretNames(m))
}

func TestGlobalSearchKeepsProjectsFolded(t *testing.T) {
	m := cachedSidebarModel()
	m.projects = []string{"api", "web"}
	m.projectConfigs["web"] = []configRow{{name: "dev"}}
	m.putSecretsCache("web", "dev", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("TOKEN_WEB", "x", "masked")},
	})
	m.expanded["api"] = true
	m.expanded["web"] = false
	m.secrets = []secretRow{newSecretRow("TOKEN", "2", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.rebuildTree()

	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("TOKEN", false))

	assert.True(t, m.expanded["api"])
	assert.False(t, m.expanded["web"])
	assert.Equal(t, []string{"api", "web"}, treeProjectNames(m))
	assert.Equal(t, []string{"dev"}, treeConfigNames(m))
	assert.True(t, m.tree[findTreeIndex(m.tree, treeProject, "web", "")].folded)
}

func TestGlobalSearchAllowsProjectFold(t *testing.T) {
	m := cachedSidebarModel()
	m.secrets = []secretRow{newSecretRow("TOKEN", "2", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("TOKEN", false))
	assert.Contains(t, treeConfigNames(m), "dev")

	m.setFocus(focusProjects)
	m.treeIdx = findTreeIndex(m.tree, treeProject, "api", "")
	next, cmd := m.Update(runeKey(' '))
	assert.Nil(t, cmd)
	assert.False(t, next.expanded["api"])
	assert.True(t, next.tree[0].folded)

	next, cmd = next.Update(runeKey(' '))
	assert.Nil(t, cmd)
	assert.True(t, next.expanded["api"])
	assert.False(t, next.tree[0].folded)
	assert.Contains(t, treeConfigNames(next), "dev")
}

func TestGlobalSearchAllowsEnvFold(t *testing.T) {
	m := cachedSidebarModel()
	m.putSecretsCache("api", "dev_personal", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("TOKEN_BRANCH", "x", "masked")},
	})
	m.secrets = []secretRow{newSecretRow("TOKEN", "2", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("TOKEN", false))
	assert.Contains(t, treeConfigNames(m), "dev_personal")

	m.setFocus(focusProjects)
	m.treeIdx = findTreeIndex(m.tree, treeConfig, "api", "dev")
	next, cmd := m.Update(runeKey(' '))
	assert.Nil(t, cmd)
	assert.False(t, isEnvExpanded(next.expandedEnvs, "api", "dev"))
	assert.NotContains(t, treeConfigNames(next), "dev_personal")

	next, cmd = next.Update(runeKey(' '))
	assert.Nil(t, cmd)
	assert.True(t, isEnvExpanded(next.expandedEnvs, "api", "dev"))
	assert.Contains(t, treeConfigNames(next), "dev_personal")
}

func filteredSecretNames(m Model) []string {
	idxs := m.filteredIndexes()
	names := make([]string, 0, len(idxs))
	for _, i := range idxs {
		names = append(names, m.secrets[i].name)
	}
	return names
}

func treeProjectNames(m Model) []string {
	var names []string
	for _, row := range m.tree {
		if row.kind == treeProject {
			names = append(names, row.project)
		}
	}
	return names
}

func treeConfigNames(m Model) []string {
	var names []string
	for _, row := range m.tree {
		if row.kind == treeConfig {
			names = append(names, row.config)
		}
	}
	return names
}

func TestNamesIndexDoesNotBlockInput(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.projects = []string{"api"}
	cmd := m.startNamesIndex()
	assert.NotNil(t, cmd)
	assert.True(t, m.workplaceIndexing)
	assert.False(t, m.fetching)
}

func TestNamesIndexPersistsAndReloads(t *testing.T) {
	orig := configuration.UserConfigDir
	t.Cleanup(func() { configuration.SetConfigDir(orig) })
	configuration.SetConfigDir(t.TempDir())

	opts := models.ScopedOptions{
		Token:   models.ScopedOption{Value: "tok"},
		APIHost: models.ScopedOption{Value: "https://api.doppler.com"},
	}
	m := newModel(opts, configuration.TUISettings{})
	m.sessionEnabled = true
	m.secretNames = map[string][]string{
		secretsCacheKey("api", "dev"): {"TOKEN", "OTHER"},
	}
	m.persistNamesIndex()

	m2 := newModel(opts, configuration.TUISettings{})
	m2.loadNamesIndex()
	assert.Equal(t, []string{"TOKEN", "OTHER"}, m2.secretNames[secretsCacheKey("api", "dev")])
}

func TestWorkplaceIndexRefreshOverwritesAndPrunes(t *testing.T) {
	m := cachedSidebarModel()
	m.secretNames = map[string][]string{
		secretsCacheKey("api", "dev"):   {"STALE"},
		secretsCacheKey("api", "gone"):  {"OLD"},
		secretsCacheKey("other", "dev"): {"KEEP"},
	}

	next, _ := m.applyWorkplaceSearch(workplaceSearchMsg{
		refresh: true,
		names: map[string][]string{
			secretsCacheKey("api", "dev"): {"TOKEN"},
		},
		configs: map[string][]configRow{
			"api": {{name: "dev"}},
		},
	})
	mod := next
	assert.Equal(t, []string{"DEV"}, mod.secretNames[secretsCacheKey("api", "dev")], "live secrets overlay the fetched names")
	assert.NotContains(t, mod.secretNames, secretsCacheKey("api", "gone"))
	assert.Equal(t, []string{"KEEP"}, mod.secretNames[secretsCacheKey("other", "dev")])
	assert.True(t, mod.namesIndexDone)
	assert.False(t, mod.workplaceIndexing)
}
