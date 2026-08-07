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
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type rect struct {
	x, y, w, h int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type layoutRegions struct {
	projects rect
	secrets  rect
	status   rect
}

func (m Model) computeLayout() layoutRegions {
	w := max(40, m.width)
	h := max(12, m.height)

	leftW := max(18, w/5)
	rightW := w - leftW
	statusH := 1
	topH := h - statusH

	return layoutRegions{
		projects: rect{0, 0, leftW, topH},
		secrets:  rect{leftW, 0, rightW, topH},
		status:   rect{0, topH, w, statusH},
	}
}

const secretColSep = "│"

func secretColumnWidths(panelW int) (nameW, valueW int) {
	inner := max(10, panelW-2)
	avail := max(9, inner-lipgloss.Width(secretColSep))
	nameW = max(12, avail*2/5)
	valueW = max(8, avail-nameW)
	return nameW, valueW
}

func secretColSeparator() string {
	style := lipgloss.NewStyle().Foreground(dim)
	if background != "" {
		style = style.Background(background)
	}
	return style.Render(secretColSep)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	layout := m.computeLayout()

	left := m.renderProjectTree(layout.projects.w, layout.projects.h)
	right := m.renderSecretsTable(layout.secrets.w, layout.secrets.h)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	status := m.renderStatus(layout.status.w)

	base := lipgloss.JoinVertical(lipgloss.Left, main, status)

	switch m.focus {
	case focusIntro:
		base = m.renderOverlay(base, m.renderIntroModal())
	case focusHelp:
		base = m.renderOverlay(base, m.renderHelpModal())
	case focusSave:
		base = m.renderOverlay(base, m.renderSaveModal())
	}

	return appStyle.Width(m.width).Height(m.height).Render(base)
}

func (m Model) renderProjectTree(width, height int) string {
	lines := make([]string, len(m.tree))
	for i, row := range m.tree {
		lines[i] = formatTreeRow(row, m.activeProject, m.activeConfig)
	}
	title := fmt.Sprintf("Projects (%d)", len(m.projects))
	hits := map[int]struct{}(nil)
	if m.searchPane == focusProjects {
		hits = m.searchMatchSet()
	}
	return m.renderLinesPanel(title, lines, m.treeIdx, m.focus == focusProjects, width, height, hits)
}

func (m Model) renderSecretsTable(width, height int) string {
	nameW, valueW := secretColumnWidths(width)
	innerH := max(1, height-2)
	sep := secretColSeparator()
	header := dimStyle.Render(padRight("  NAME", nameW)) + sep + dimStyle.Render(padRight("VALUE", valueW))

	idxs := m.filteredIndexes()
	bodyH := max(1, innerH-1)
	start := 0
	if m.secretIdx >= bodyH {
		start = m.secretIdx - bodyH + 1
	}
	end := start + bodyH
	if end > len(idxs) {
		end = len(idxs)
	}

	hits := map[int]struct{}(nil)
	if m.searchPane == focusSecrets {
		hits = m.searchMatchSet()
	}
	secretsActive := m.focus == focusSecrets || m.focus == focusSecretInsert

	var rows []string
	rows = append(rows, header)
	for i := start; i < end; i++ {
		s := m.secrets[idxs[i]]
		rows = append(rows, m.renderSecretRow(s, i, nameW, valueW, secretsActive, hits))
	}
	for len(rows) < innerH {
		rows = append(rows, "")
	}

	title := fmt.Sprintf("Secrets (%d)", len(idxs))
	if m.activeProject != "" && m.activeConfig != "" {
		title = fmt.Sprintf("Secrets (%d) [%s / %s]", len(idxs), m.activeProject, m.activeConfig)
	}
	return renderTitledListPanel(title, strings.Join(rows[:innerH], "\n"), width, height, secretsActive)
}

func (m Model) renderSecretRow(s secretRow, listIdx, nameW, valueW int, paneActive bool, hits map[int]struct{}) string {
	marker := "  "
	if s.shouldDelete {
		marker = "D "
	} else if s.isDirty() {
		marker = "* "
	}

	nameText := s.name
	valueText := flattenForCell(s.previewValueUnlimited())
	if len([]rune(valueText)) > 200 {
		valueText = string([]rune(valueText)[:200]) + "…"
	}

	rowSelected := paneActive && listIdx == m.secretIdx
	nameActive := rowSelected && m.secretCol == colName
	valueActive := rowSelected && m.secretCol == colValue
	editing := m.focus == focusSecretInsert && listIdx == m.secretIdx
	hit := hits != nil && containsHit(hits, listIdx) && !rowSelected

	var nameCell, valueCell string
	if editing && m.secretCol == colName {
		m.cellInput.Width = max(1, nameW-lipgloss.Width(marker))
		m.cellInput.TextStyle = selectedStyle
		m.cellInput.Cursor.Style = selectedStyle
		m.cellInput.Cursor.TextStyle = selectedStyle
		nameCell = padStyledCell(marker+m.cellInput.View(), nameW, true)
	} else {
		nameCell = styleSecretCell(marker+nameText, nameW, nameActive, s.shouldDelete, s.isDirty(), hit)
	}

	if editing && m.secretCol == colValue {
		m.cellInput.Width = max(1, valueW)
		m.cellInput.TextStyle = selectedStyle
		m.cellInput.Cursor.Style = selectedStyle
		m.cellInput.Cursor.TextStyle = selectedStyle
		valueCell = padStyledCell(m.cellInput.View(), valueW, true)
	} else {
		valueCell = styleSecretCell(valueText, valueW, valueActive, s.shouldDelete, s.isDirty(), hit)
	}

	return nameCell + secretColSeparator() + valueCell
}

func padStyledCell(styled string, width int, selected bool) string {
	w := lipgloss.Width(styled)
	if w > width {
		return lipgloss.NewStyle().MaxWidth(width).Render(styled)
	}
	pad := strings.Repeat(" ", width-w)
	if selected {
		pad = selectedStyle.Render(pad)
	} else if background != "" {
		pad = lipgloss.NewStyle().Background(background).Render(pad)
	}
	return styled + pad
}

func (s secretRow) previewValueUnlimited() string {
	v := s.displayValue()
	return strings.ReplaceAll(v, "\n", " ")
}

func containsHit(hits map[int]struct{}, i int) bool {
	_, ok := hits[i]
	return ok
}

func styleSecretCell(text string, width int, active, del, dirty, hit bool) string {
	plain := truncate(ansi.Strip(text), width)
	plain = padRight(plain, width)
	switch {
	case active:
		return selectedStyle.Width(width).MaxWidth(width).Render(plain)
	case del:
		return deleteStyle.Width(width).MaxWidth(width).Render(plain)
	case dirty:
		return dirtyStyle.Width(width).MaxWidth(width).Render(plain)
	case hit:
		return searchHitStyle.Width(width).MaxWidth(width).Render(plain)
	default:
		style := lipgloss.NewStyle().Width(width).MaxWidth(width)
		if background != "" {
			style = style.Background(background)
		}
		if textColor != "" {
			style = style.Foreground(textColor)
		}
		return style.Render(plain)
	}
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m Model) renderLinesPanel(title string, lines []string, selected int, active bool, width, height int, hits map[int]struct{}) string {
	innerW := max(1, width-2)
	innerH := max(1, height-2)

	visible := innerH
	start := 0
	if selected >= visible {
		start = selected - visible + 1
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	var body []string
	for i := start; i < end; i++ {
		line := truncatePreserve(lines[i], innerW)
		if active && i == selected {
			plain := truncate(ansi.Strip(lines[i]), innerW)
			line = selectedStyle.Width(innerW).MaxWidth(innerW).Render(plain)
		} else if _, ok := hits[i]; ok {
			plain := truncate(ansi.Strip(lines[i]), innerW)
			line = searchHitStyle.Width(innerW).MaxWidth(innerW).Render(plain)
		}
		body = append(body, line)
	}
	for len(body) < innerH {
		body = append(body, "")
	}

	return renderTitledListPanel(title, strings.Join(body[:innerH], "\n"), width, height, active)
}

func (m Model) renderStatus(width int) string {
	var left string
	if m.fetching {
		left = m.spinner.View() + " Loading…"
	} else if m.focus == focusSearch {
		left = helpStyle.Render("search · enter apply · esc clear")
	} else if m.focus == focusFilter {
		left = helpStyle.Render("filter · enter/esc apply")
	} else if m.focus == focusSecretInsert {
		left = helpStyle.Render("insert · esc/enter normal · tab next cell")
	} else if m.errMsg != "" {
		left = errorStyle.Render(m.errMsg)
	} else if m.statusMsg != "" {
		left = statusStyle.Render(m.statusMsg)
	} else {
		left = helpStyle.Render("tab cycle · hjkl move · i edit · / search · ? help")
	}

	var right string
	rightW := max(12, min(40, width/2))
	inputTextStyle := lipgloss.NewStyle().Foreground(textColor)
	if background != "" {
		inputTextStyle = inputTextStyle.Background(background)
	}
	switch {
	case m.focus == focusSearch:
		m.searchInput.Width = max(8, rightW-2)
		m.searchInput.TextStyle = inputTextStyle
		m.searchInput.PromptStyle = helpStyle
		right = m.searchInput.View()
	case m.focus == focusFilter:
		m.filterInput.Width = max(8, rightW-2)
		m.filterInput.TextStyle = inputTextStyle
		m.filterInput.PromptStyle = helpStyle
		right = m.filterInput.View()
	case m.searchQuery != "":
		right = searchHitStyle.Render(m.searchStatusLabel())
		if m.filter != "" {
			right = helpStyle.Render("f "+m.filter) + "  " + right
		}
	case m.filter != "":
		right = activeEnvStyle.Render("f " + m.filter)
	}

	var line string
	if right == "" {
		line = left
	} else {
		leftW := lipgloss.Width(left)
		rw := lipgloss.Width(right)
		gap := width - leftW - rw
		if gap < 1 {
			maxLeft := max(0, width-rw-1)
			left = lipgloss.NewStyle().MaxWidth(maxLeft).Render(left)
			leftW = lipgloss.Width(left)
			gap = max(1, width-leftW-rw)
		}
		gapStyle := lipgloss.NewStyle()
		if background != "" {
			gapStyle = gapStyle.Background(background)
		}
		line = left + gapStyle.Render(strings.Repeat(" ", gap)) + right
	}

	return statusBarStyle.Width(width).MaxWidth(width).Render(line)
}

func renderTitledPanel(title, content string, width, height int, active bool) string {
	style := panelStyle
	if active {
		style = activePanelStyle
	}
	return renderTitledPanelStyle(title, content, width, height, active, style)
}

func renderTitledListPanel(title, content string, width, height int, active bool) string {
	style := listPanelStyle
	if active {
		style = activeListPanelStyle
	}
	return renderTitledPanelStyle(title, content, width, height, active, style)
}

func renderTitledPanelStyle(title, content string, width, height int, active bool, style lipgloss.Style) string {
	rendered := style.Width(width - 2).Height(height - 2).Render(content)
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}
	lines[0] = titledTopBorder(title, width, active)
	return strings.Join(lines, "\n")
}

func titledTopBorder(title string, width int, active bool) string {
	fg := dim
	tStyle := titleStyle
	if active {
		fg = accent
		tStyle = activeTitleStyle
	}
	borderStyle := lipgloss.NewStyle().Foreground(fg)
	if background != "" {
		borderStyle = borderStyle.Background(background)
		tStyle = tStyle.Background(background)
	}

	label := " " + strings.TrimSpace(title) + " "
	inner := max(0, width-2)
	if lipgloss.Width(label) > inner {
		label = " " + truncate(strings.TrimSpace(title), max(1, inner-2)) + " "
	}

	gap := inner - lipgloss.Width(label)
	left := 1
	if gap < left {
		left = gap
	}
	right := gap - left

	return borderStyle.Render(roundedBorder.TopLeft) +
		borderStyle.Render(strings.Repeat(roundedBorder.Top, left)) +
		tStyle.Render(label) +
		borderStyle.Render(strings.Repeat(roundedBorder.Top, right)) +
		borderStyle.Render(roundedBorder.TopRight)
}

func (m Model) renderIntroModal() string {
	body := `Welcome to the Doppler TUI!

Close this window with Enter/Esc, then press ?
for keybindings.

Secrets are a two-column grid. Use hjkl to move,
i/Enter to edit a cell, Esc for normal mode.

https://github.com/DopplerHQ/cli`
	return renderTitledPanel("Welcome", body, min(64, max(40, m.width-8)), 12, true)
}

func (m Model) renderHelpModal() string {
	body := m.helpViewport.View() + "\n\n" + helpStyle.Render("Enter/Esc close")
	return renderTitledPanel("Help", body, min(64, m.width-4), min(28, m.height-4), true)
}

func (m Model) renderSaveModal() string {
	var body string
	if len(m.pendingChanges) == 0 {
		body = "There are no changes to save"
	} else {
		body = "The following secrets will be updated:\n\n"
		for _, c := range m.pendingChanges {
			body += "● " + fmt.Sprint(c.Name) + "\n"
		}
		body += "\nEnter confirm · Esc/q cancel"
	}
	return renderTitledPanel("Confirm Changes", body, min(60, m.width-4), min(20, m.height-4), true)
}

func (m Model) renderOverlay(base, modal string) string {
	opts := []lipgloss.WhitespaceOption{
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(dim),
	}
	if background != "" {
		opts = append(opts, lipgloss.WithWhitespaceBackground(background))
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, opts...)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func truncatePreserve(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
