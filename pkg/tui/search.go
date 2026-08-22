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
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func searchableTreeText(row treeRow) string {
	if row.kind == treeProject {
		return row.project
	}
	return row.config
}

type globalHit struct {
	project string
	config  string
	name    string
}

func (m *Model) clearSearch() {
	m.searchGen++
	m.searchQuery = ""
	m.searchRe = nil
	m.searchMatches = nil
	m.searchMatchIdx = 0
	m.searchGlobal = false
	m.globalHits = nil
	m.pendingSearchName = ""
	m.searchInput.SetValue("")
	m.setSearchPrompt()
	if m.statusMsg == "No matches" || m.statusMsg == "Invalid regex" {
		m.statusMsg = ""
	}
}

func (m *Model) setSearchPrompt() {
	if m.searchGlobal {
		m.searchInput.Prompt = "g/ "
		m.searchInput.Placeholder = "Keys and values…"
		return
	}
	m.searchInput.Prompt = "/ "
	m.searchInput.Placeholder = "Search regex…"
}

func (m *Model) beginSearch() {
	m.searchGlobal = false
	m.globalHits = nil
	m.setSearchPrompt()
	m.searchPane = m.focus
	if m.searchPane != focusProjects && m.searchPane != focusSecrets {
		m.searchPane = focusSecrets
	}
	m.searchHistory.Reset()
	m.setFocus(focusSearch)
}

func (m *Model) beginGlobalSearch() {
	m.searchGlobal = true
	m.searchPane = focusSecrets
	m.setSearchPrompt()
	m.searchHistory.Reset()
	if m.searchQuery != "" {
		_ = m.applySearch(m.searchQuery, false)
	}
	m.setFocus(focusSearch)
}

func (m *Model) compileSearch(query string) error {
	return m.applySearch(query, true)
}

func (m *Model) applySearch(query string, jump bool) error {
	if query == "" {
		wasGlobal := m.searchGlobal
		m.clearSearch()
		m.searchGlobal = wasGlobal
		m.setSearchPrompt()
		return nil
	}
	pattern := query
	if searchShouldIgnoreCase(query, m.cfg.CaseMode) {
		pattern = "(?i)" + query
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		m.searchQuery = query
		m.searchRe = nil
		m.searchMatches = nil
		m.searchMatchIdx = 0
		return err
	}
	m.searchQuery = query
	m.searchRe = re
	if m.searchGlobal {
		m.applyGlobalHits(jump)
		return nil
	}
	m.refreshSearchMatches()
	if len(m.searchMatches) == 0 {
		m.searchMatchIdx = 0
		if jump {
			m.statusMsg = "No matches"
		} else {
			m.statusMsg = ""
		}
		return nil
	}
	m.statusMsg = ""
	m.searchMatchIdx = 0
	if jump {
		m.jumpToSearchMatch(0)
	} else {
		m.selectSearchMatch(0)
	}
	return nil
}

func searchShouldIgnoreCase(query, caseMode string) bool {
	switch caseMode {
	case "sensitive":
		return false
	case "insensitive":
		return true
	default: // smart
		return !hasUpper(query)
	}
}

func (m *Model) refreshSearchMatches() {
	m.searchMatches = nil
	if m.searchRe == nil {
		return
	}
	switch m.searchPane {
	case focusProjects:
		for i, row := range m.tree {
			if m.searchRe.MatchString(searchableTreeText(row)) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	case focusSecrets:
		for i, idx := range m.filteredIndexes() {
			if secretMatchesSearch(m.secrets[idx], m.searchRe) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	}
}

func secretMatchesSearch(s secretRow, re *regexp.Regexp) bool {
	if re == nil {
		return false
	}
	if re.MatchString(s.name) {
		return true
	}
	if s.originalVisibility == "restricted" && !s.isTouched {
		return false
	}
	return re.MatchString(s.value)
}

func collectSecretHits(re *regexp.Regexp, project, config string, secrets []secretRow) []globalHit {
	if re == nil || project == "" || config == "" {
		return nil
	}
	var hits []globalHit
	for _, s := range secrets {
		if s.shouldDelete || strings.TrimSpace(s.name) == "" {
			continue
		}
		if secretMatchesSearch(s, re) {
			hits = append(hits, globalHit{project: project, config: config, name: s.name})
		}
	}
	return hits
}

func (m *Model) applyGlobalHits(jump bool) {
	m.refreshGlobalMatches()
	m.syncLocalMatchesFromGlobal()
	if len(m.globalHits) == 0 {
		m.searchMatchIdx = 0
		if jump {
			m.statusMsg = "No matches"
		} else {
			m.statusMsg = ""
		}
		return
	}
	m.statusMsg = ""
	if jump {
		m.searchMatchIdx = 0
		m.jumpToGlobalHit(0)
		return
	}
	if i := m.firstGlobalHitIn(m.activeProject, m.activeConfig); i >= 0 {
		m.searchMatchIdx = i
		m.selectSecretByName(m.globalHits[i].name)
		return
	}
	m.searchMatchIdx = 0
}

func (m *Model) refreshGlobalMatches() {
	m.globalHits = nil
	if m.searchRe == nil {
		return
	}
	m.stashCurrentSecrets()
	seen := map[string]bool{}
	for _, project := range m.projects {
		for _, cfg := range m.projectConfigs[project] {
			key := secretsCacheKey(project, cfg.name)
			secrets := m.configSecrets(project, cfg.name)
			if secrets == nil {
				continue
			}
			seen[key] = true
			m.globalHits = append(m.globalHits, collectSecretHits(m.searchRe, project, cfg.name, secrets)...)
		}
	}
	for key, e := range m.secretsCache {
		if seen[key] {
			continue
		}
		project, config, ok := splitSecretsCacheKey(key)
		if !ok {
			continue
		}
		m.globalHits = append(m.globalHits, collectSecretHits(m.searchRe, project, config, e.secrets)...)
	}
}

func (m *Model) syncLocalMatchesFromGlobal() {
	m.searchMatches = nil
	m.searchPane = focusSecrets
	if !m.searchGlobal {
		return
	}
	nameIdx := map[string]int{}
	for i, idx := range m.filteredIndexes() {
		nameIdx[m.secrets[idx].name] = i
	}
	for _, h := range m.globalHits {
		if h.project != m.activeProject || h.config != m.activeConfig {
			continue
		}
		if i, ok := nameIdx[h.name]; ok {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
}

func (m Model) firstGlobalHitIn(project, config string) int {
	for i, h := range m.globalHits {
		if h.project == project && h.config == config {
			return i
		}
	}
	return -1
}

func (m *Model) selectSecretByName(name string) bool {
	if name == "" {
		return false
	}
	for i, idx := range m.filteredIndexes() {
		if m.secrets[idx].name == name {
			m.secretIdx = i
			return true
		}
	}
	for _, s := range m.secrets {
		if s.name != name {
			continue
		}
		m.filter = ""
		m.filterInput.SetValue("")
		for i, idx := range m.filteredIndexes() {
			if m.secrets[idx].name == name {
				m.secretIdx = i
				return true
			}
		}
	}
	return false
}

func (m *Model) jumpToGlobalHit(matchPos int) tea.Cmd {
	if matchPos < 0 || matchPos >= len(m.globalHits) {
		return nil
	}
	m.searchMatchIdx = matchPos
	hit := m.globalHits[matchPos]
	if hit.project == m.activeProject && hit.config == m.activeConfig {
		m.selectSecretByName(hit.name)
		m.syncLocalMatchesFromGlobal()
		m.setFocus(focusSecrets)
		return nil
	}
	if m.configIsCached(hit.project, hit.config) {
		m.stashCurrentSecrets()
		if m.applyCachedSecrets(hit.project, hit.config) {
			m.selectSecretByName(hit.name)
			m.syncLocalMatchesFromGlobal()
			m.setFocus(focusSecrets)
			m.expanded[hit.project] = true
			m.rebuildTree()
			m.treeIdx = findTreeIndex(m.tree, treeConfig, hit.project, hit.config)
		}
		return nil
	}
	m.pendingSearchName = hit.name
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	return tea.Batch(m.spinner.Tick, selectConfigCmd(m.opts, hit.project, hit.config))
}

func (m *Model) selectSearchMatch(matchPos int) {
	if matchPos < 0 || matchPos >= len(m.searchMatches) {
		return
	}
	idx := m.searchMatches[matchPos]
	switch m.searchPane {
	case focusProjects:
		m.treeIdx = idx
	case focusSecrets:
		m.secretIdx = idx
	}
}

func (m *Model) jumpToSearchMatch(matchPos int) {
	m.selectSearchMatch(matchPos)
	switch m.searchPane {
	case focusProjects:
		m.focus = focusProjects
		m.revealHighlightedConfig()
	case focusSecrets:
		m.focus = focusSecrets
	}
}

func (m *Model) stepSearchMatch(delta int) tea.Cmd {
	if m.searchRe == nil {
		return nil
	}
	if m.searchGlobal {
		if len(m.globalHits) == 0 {
			m.refreshGlobalMatches()
		}
		if len(m.globalHits) == 0 {
			m.statusMsg = "No matches"
			return nil
		}
		n := len(m.globalHits)
		m.searchMatchIdx = (m.searchMatchIdx + delta%n + n) % n
		cmd := m.jumpToGlobalHit(m.searchMatchIdx)
		m.statusMsg = fmt.Sprintf("Match %d/%d", m.searchMatchIdx+1, n)
		return cmd
	}
	m.refreshSearchMatches()
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No matches"
		return nil
	}
	n := len(m.searchMatches)
	m.searchMatchIdx = (m.searchMatchIdx + delta%n + n) % n
	m.jumpToSearchMatch(m.searchMatchIdx)
	m.statusMsg = fmt.Sprintf("Match %d/%d", m.searchMatchIdx+1, n)
	return nil
}

func (m Model) searchStatusLabel() string {
	if m.searchQuery == "" {
		return ""
	}
	prefix := "/ "
	if m.searchGlobal {
		prefix = "g/ "
	}
	n := len(m.searchMatches)
	if m.searchGlobal {
		n = len(m.globalHits)
	}
	if n == 0 {
		return prefix + m.searchQuery + " (0)"
	}
	return fmt.Sprintf("%d/%d %s%s", m.searchMatchIdx+1, n, prefix, m.searchQuery)
}

func baseTextStyle() lipgloss.Style {
	style := lipgloss.NewStyle()
	if background != "" {
		style = style.Background(background)
	}
	if textColor != "" {
		style = style.Foreground(textColor)
	}
	return style
}

func highlightMatches(text string, re *regexp.Regexp, base lipgloss.Style) string {
	if re == nil || text == "" {
		return base.Render(text)
	}
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return base.Render(text)
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		if m[0] < last || m[0] >= m[1] {
			continue
		}
		if m[0] > last {
			b.WriteString(base.Render(text[last:m[0]]))
		}
		b.WriteString(searchHitStyle.Render(text[m[0]:m[1]]))
		last = m[1]
	}
	if last < len(text) {
		b.WriteString(base.Render(text[last:]))
	}
	return b.String()
}

func highlightNeedleInDisplay(display, needle string, re *regexp.Regexp, base lipgloss.Style) string {
	if re == nil || needle == "" {
		return base.Render(display)
	}
	offset := strings.LastIndex(display, needle)
	if offset < 0 {
		return base.Render(display)
	}
	matches := re.FindAllStringIndex(needle, -1)
	if len(matches) == 0 {
		return base.Render(display)
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		start := offset + m[0]
		end := offset + m[1]
		if start < last || start >= end {
			continue
		}
		if start > last {
			b.WriteString(base.Render(display[last:start]))
		}
		b.WriteString(searchHitStyle.Render(display[start:end]))
		last = end
	}
	if last < len(display) {
		b.WriteString(base.Render(display[last:]))
	}
	return b.String()
}

func (m *Model) startWorkplaceSearch() tea.Cmd {
	if m.searchQuery == "" || m.searchRe == nil {
		return nil
	}
	m.stashCurrentSecrets()
	m.searchGen++
	gen := m.searchGen
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""

	projects := append([]string(nil), m.projects...)
	known := make(map[string][]configRow, len(m.projectConfigs))
	for p, cfgs := range m.projectConfigs {
		known[p] = append([]configRow(nil), cfgs...)
	}
	cached := make(map[string][]secretRow, len(m.secretsCache))
	for k, e := range m.secretsCache {
		cached[k] = append([]secretRow(nil), e.secrets...)
	}
	return tea.Batch(m.spinner.Tick, workplaceSearchCmd(m.opts, gen, m.searchQuery, m.cfg.CaseMode, projects, known, cached))
}

func (m Model) applyWorkplaceSearch(msg workplaceSearchMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.searchGen || !m.searchGlobal {
		return m, nil
	}
	m.fetching = false
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	for project, cfgs := range msg.configs {
		if _, ok := m.projectConfigs[project]; !ok {
			m.projectConfigs[project] = cfgs
		}
	}
	for key, secrets := range msg.fetched {
		project, config, ok := splitSecretsCacheKey(key)
		if !ok {
			continue
		}
		if m.configIsDirty(project, config) {
			continue
		}
		if _, ok := m.secretsCache[key]; ok {
			continue
		}
		m.putSecretsCache(project, config, secretsCacheEntry{secrets: secrets})
	}

	keep := globalHit{}
	if m.searchMatchIdx >= 0 && m.searchMatchIdx < len(m.globalHits) {
		keep = m.globalHits[m.searchMatchIdx]
	}
	m.globalHits = msg.hits
	sortGlobalHits(m.globalHits, m.projects, m.projectConfigs)
	m.syncLocalMatchesFromGlobal()
	if len(m.globalHits) == 0 {
		m.searchMatchIdx = 0
		m.statusMsg = "No matches"
		return m, nil
	}
	idx := 0
	for i, h := range m.globalHits {
		if h == keep {
			idx = i
			break
		}
	}
	m.statusMsg = fmt.Sprintf("Match %d/%d", idx+1, len(m.globalHits))
	return m, m.jumpToGlobalHit(idx)
}

func sortGlobalHits(hits []globalHit, projects []string, configs map[string][]configRow) {
	projIdx := map[string]int{}
	for i, p := range projects {
		projIdx[p] = i
	}
	cfgIdx := map[string]int{}
	for p, cfgs := range configs {
		for i, c := range cfgs {
			cfgIdx[secretsCacheKey(p, c.name)] = i
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if projIdx[a.project] != projIdx[b.project] {
			return projIdx[a.project] < projIdx[b.project]
		}
		if a.project != b.project {
			return a.project < b.project
		}
		ka := secretsCacheKey(a.project, a.config)
		kb := secretsCacheKey(b.project, b.config)
		if cfgIdx[ka] != cfgIdx[kb] {
			return cfgIdx[ka] < cfgIdx[kb]
		}
		if a.config != b.config {
			return a.config < b.config
		}
		return a.name < b.name
	})
}
