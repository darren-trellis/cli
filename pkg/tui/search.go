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

	"github.com/DopplerHQ/cli/pkg/configuration"
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
	m.rebuildTree()
}

func (m *Model) setSearchPrompt() {
	if m.searchGlobal {
		m.searchInput.Prompt = "g/ "
		m.searchInput.Placeholder = "Secret names…"
		return
	}
	m.searchInput.Prompt = "/ "
	m.searchInput.Placeholder = "Search regex…"
}

func (m *Model) beginSearch() {
	m.clearSearch()
	m.searchPane = m.focus
	if m.searchPane != focusProjects && m.searchPane != focusSecrets {
		m.searchPane = focusSecrets
	}
	m.setSearchPrompt()
	m.searchHistory.Reset()
	m.setFocus(focusSearch)
}

func (m *Model) beginGlobalSearch() Cmd {
	m.clearSearch()
	m.searchGlobal = true
	m.searchPane = focusSecrets
	m.setSearchPrompt()
	m.searchHistory.Reset()
	m.setFocus(focusSearch)
	return m.startNamesIndex()
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

func collectNameHits(re *regexp.Regexp, project, config string, names []string) []globalHit {
	if re == nil || project == "" || config == "" {
		return nil
	}
	var hits []globalHit
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if re.MatchString(name) {
			hits = append(hits, globalHit{project: project, config: config, name: name})
		}
	}
	return hits
}

func namesFromSecrets(secrets []secretRow) []string {
	names := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s.shouldDelete || strings.TrimSpace(s.name) == "" {
			continue
		}
		names = append(names, s.name)
	}
	return names
}

func (m Model) namesFor(project, config string) []string {
	if secrets := m.configSecrets(project, config); secrets != nil {
		return namesFromSecrets(secrets)
	}
	if names, ok := m.secretNames[secretsCacheKey(project, config)]; ok {
		return names
	}
	return nil
}

func (m *Model) applyGlobalHits(jump bool) {
	m.refreshGlobalMatches()
	m.rebuildTree()
	m.syncLocalMatchesFromGlobal()
	m.clampSecretIdx()
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
			if cfg.name == "" {
				continue
			}
			key := secretsCacheKey(project, cfg.name)
			names := m.namesFor(project, cfg.name)
			if names == nil {
				continue
			}
			seen[key] = true
			m.globalHits = append(m.globalHits, collectNameHits(m.searchRe, project, cfg.name, names)...)
		}
	}
	for key, names := range m.secretNames {
		if seen[key] {
			continue
		}
		project, config, ok := splitSecretsCacheKey(key)
		if !ok {
			continue
		}
		seen[key] = true
		m.globalHits = append(m.globalHits, collectNameHits(m.searchRe, project, config, names)...)
	}
	for key, e := range m.secretsCache {
		if seen[key] {
			continue
		}
		project, config, ok := splitSecretsCacheKey(key)
		if !ok {
			continue
		}
		m.globalHits = append(m.globalHits, collectNameHits(m.searchRe, project, config, namesFromSecrets(e.secrets))...)
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

func (m *Model) jumpToGlobalHit(matchPos int) Cmd {
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
	return Batch(m.spinner.Tick, selectConfigCmd(m.opts, hit.project, hit.config))
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

func (m *Model) stepSearchMatch(delta int) Cmd {
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

func (m Model) filteredSearchTree() (projects []string, configs map[string][]configRow) {
	keep := map[string]bool{}
	for _, h := range m.globalHits {
		if h.project == "" || h.config == "" {
			continue
		}
		keep[secretsCacheKey(h.project, h.config)] = true
	}

	order := append([]string(nil), m.projects...)
	seenProj := map[string]bool{}
	for _, p := range m.projects {
		seenProj[p] = true
	}
	var extra []string
	for key := range keep {
		proj, _, ok := splitSecretsCacheKey(key)
		if !ok || seenProj[proj] {
			continue
		}
		extra = append(extra, proj)
		seenProj[proj] = true
	}
	sort.Strings(extra)
	order = append(order, extra...)

	configs = map[string][]configRow{}
	for _, p := range order {
		var rows []configRow
		seen := map[string]bool{}
		for _, c := range m.projectConfigs[p] {
			if !keep[secretsCacheKey(p, c.name)] {
				continue
			}
			rows = append(rows, c)
			seen[c.name] = true
		}
		for key := range keep {
			proj, cfg, ok := splitSecretsCacheKey(key)
			if !ok || proj != p || seen[cfg] {
				continue
			}
			rows = append(rows, configRow{name: cfg})
			seen[cfg] = true
		}
		if len(rows) == 0 {
			continue
		}
		configs[p] = rows
		projects = append(projects, p)
	}
	return projects, configs
}

func (m *Model) startNamesIndex() Cmd {
	return m.startWorkplaceIndex(false)
}

func (m *Model) startWorkplaceIndex(refresh bool) Cmd {
	if m.workplaceIndexing {
		return nil
	}
	if m.namesIndexDone {
		return nil
	}
	if len(m.projects) == 0 {
		return nil
	}
	m.workplaceIndexing = true
	m.errMsg = ""

	projects := append([]string(nil), m.projects...)
	known := make(map[string][]configRow, len(m.projectConfigs))
	for p, cfgs := range m.projectConfigs {
		known[p] = append([]configRow(nil), cfgs...)
	}
	haveNames := map[string]bool{}
	if !refresh {
		for k := range m.secretNames {
			haveNames[k] = true
		}
		for k := range m.secretsCache {
			haveNames[k] = true
		}
		if m.activeProject != "" && m.activeConfig != "" {
			haveNames[secretsCacheKey(m.activeProject, m.activeConfig)] = true
		}
	}
	return Batch(m.spinner.Tick, workplaceIndexCmd(m.opts, projects, known, haveNames, refresh))
}

func (m Model) applyWorkplaceSearch(msg workplaceSearchMsg) (Model, Cmd) {
	if msg.err != nil {
		m.workplaceIndexing = false
		if m.searchGlobal || m.focus == focusSearch {
			m.errMsg = msg.err.Error()
		}
		return m, nil
	}
	if m.secretNames == nil {
		m.secretNames = map[string][]string{}
	}
	listed := map[string]bool{}
	for project, cfgs := range msg.configs {
		if msg.refresh || m.projectConfigs[project] == nil {
			m.projectConfigs[project] = cfgs
		}
		for _, c := range cfgs {
			if c.name == "" {
				continue
			}
			listed[secretsCacheKey(project, c.name)] = true
		}
	}
	if msg.refresh {
		for key := range m.secretNames {
			project, _, ok := splitSecretsCacheKey(key)
			if !ok {
				continue
			}
			if _, ok := msg.configs[project]; ok && !listed[key] {
				delete(m.secretNames, key)
			}
		}
		for key, names := range msg.names {
			m.secretNames[key] = names
		}
	} else {
		for key, names := range msg.names {
			if _, ok := m.secretNames[key]; !ok {
				m.secretNames[key] = names
			}
		}
	}
	m.overlayLiveSecretNames()
	m.workplaceIndexing = false
	m.namesIndexDone = true
	m.persistNamesIndex()
	if m.searchGlobal && m.searchQuery != "" {
		wasFocus := m.focus
		m.applyGlobalHits(false)
		if wasFocus == focusSearch {
			m.setFocus(focusSearch)
		}
	} else if m.searchGlobal {
		m.rebuildTree()
	}
	if m.focus == focusSecretInsert {
		m.refreshSecretRefCompletions()
	}
	return m, nil
}

func (m *Model) overlayLiveSecretNames() {
	if m.secretNames == nil {
		m.secretNames = map[string][]string{}
	}
	for key, e := range m.secretsCache {
		m.secretNames[key] = namesFromSecrets(e.secrets)
	}
	if m.activeProject != "" && m.activeConfig != "" {
		m.secretNames[secretsCacheKey(m.activeProject, m.activeConfig)] = namesFromSecrets(m.secrets)
	}
}

func (m *Model) loadNamesIndex() {
	names, ok := configuration.LoadTUINamesIndex(m.opts.Token.Value, m.opts.APIHost.Value)
	if !ok {
		return
	}
	m.secretNames = names
	m.overlayLiveSecretNames()
}

func (m Model) persistNamesIndex() {
	if !m.sessionEnabled {
		return
	}
	configuration.SaveTUINamesIndex(m.opts.Token.Value, m.opts.APIHost.Value, m.secretNames)
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
