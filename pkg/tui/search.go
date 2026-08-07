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
)

func searchableTreeText(row treeRow) string {
	if row.kind == treeProject {
		return row.project
	}
	return row.config
}

func (m *Model) clearSearch() {
	m.searchQuery = ""
	m.searchRe = nil
	m.searchMatches = nil
	m.searchMatchIdx = 0
	m.searchInput.SetValue("")
	if m.statusMsg == "No matches" || m.statusMsg == "Invalid regex" {
		m.statusMsg = ""
	}
}

func (m *Model) beginSearch() {
	m.searchPane = m.focus
	if m.searchPane != focusProjects && m.searchPane != focusSecrets {
		m.searchPane = focusSecrets
	}
	m.setFocus(focusSearch)
}

func (m *Model) compileSearch(query string) error {
	if query == "" {
		m.clearSearch()
		return nil
	}
	re, err := regexp.Compile(query)
	if err != nil {
		return err
	}
	m.searchQuery = query
	m.searchRe = re
	m.refreshSearchMatches()
	if len(m.searchMatches) == 0 {
		m.searchMatchIdx = 0
		m.statusMsg = "No matches"
		return nil
	}
	m.statusMsg = ""
	m.searchMatchIdx = 0
	m.jumpToSearchMatch(0)
	return nil
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
			s := m.secrets[idx]
			if m.searchRe.MatchString(s.name) || m.searchRe.MatchString(s.previewValue()) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	}
}

func (m *Model) jumpToSearchMatch(matchPos int) {
	if matchPos < 0 || matchPos >= len(m.searchMatches) {
		return
	}
	idx := m.searchMatches[matchPos]
	switch m.searchPane {
	case focusProjects:
		m.treeIdx = idx
		m.focus = focusProjects
	case focusSecrets:
		m.secretIdx = idx
		m.focus = focusSecrets
	}
}

func (m *Model) stepSearchMatch(delta int) {
	if m.searchRe == nil {
		return
	}
	m.refreshSearchMatches()
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No matches"
		return
	}
	n := len(m.searchMatches)
	m.searchMatchIdx = (m.searchMatchIdx + delta%n + n) % n
	m.jumpToSearchMatch(m.searchMatchIdx)
	m.statusMsg = fmt.Sprintf("Match %d/%d", m.searchMatchIdx+1, n)
}

func (m Model) searchMatchSet() map[int]struct{} {
	if m.searchRe == nil || len(m.searchMatches) == 0 {
		return nil
	}
	set := make(map[int]struct{}, len(m.searchMatches))
	for _, idx := range m.searchMatches {
		set[idx] = struct{}{}
	}
	return set
}

func (m Model) searchStatusLabel() string {
	if m.searchQuery == "" {
		return ""
	}
	if len(m.searchMatches) == 0 {
		return "/ " + m.searchQuery + " (0)"
	}
	return fmt.Sprintf("%d/%d / %s", m.searchMatchIdx+1, len(m.searchMatches), m.searchQuery)
}
