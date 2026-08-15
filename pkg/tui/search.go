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

	"github.com/charmbracelet/lipgloss"
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
	m.searchHistory.Reset()
	m.setFocus(focusSearch)
}

func (m *Model) compileSearch(query string) error {
	return m.applySearch(query, true)
}

func (m *Model) applySearch(query string, jump bool) error {
	if query == "" {
		m.clearSearch()
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
			s := m.secrets[idx]
			if m.searchRe.MatchString(s.name) || m.searchRe.MatchString(s.previewValue()) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	}
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

func (m Model) searchStatusLabel() string {
	if m.searchQuery == "" {
		return ""
	}
	if len(m.searchMatches) == 0 {
		return "/ " + m.searchQuery + " (0)"
	}
	return fmt.Sprintf("%d/%d / %s", m.searchMatchIdx+1, len(m.searchMatches), m.searchQuery)
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
