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

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
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
	suggest  rect
	status   rect
}

func (m Model) computeLayout() layoutRegions {
	w := max(40, m.width)
	h := max(12, m.height)

	statusH := 1
	suggestH := 0
	if m.focus == focusCommand {
		suggestH = m.completions.DesiredHeight(max(0, h-statusH-4))
	}
	topH := h - statusH - suggestH
	layout := layoutRegions{
		status: rect{0, topH + suggestH, w, statusH},
	}
	if suggestH > 0 {
		layout.suggest = rect{0, topH, w, suggestH}
	}
	if !m.cfg.Sidebar {
		layout.secrets = rect{0, 0, w, topH}
		return layout
	}

	sideW := m.sidebarWidth(w)
	mainW := w - sideW
	if m.cfg.SidebarPosition == "right" {
		layout.secrets = rect{0, 0, mainW, topH}
		layout.projects = rect{mainW, 0, sideW, topH}
	} else {
		layout.projects = rect{0, 0, sideW, topH}
		layout.secrets = rect{sideW, 0, mainW, topH}
	}
	return layout
}

func (m Model) sidebarWidth(totalW int) int {
	if m.cfg.SidebarWidth > 0 {
		w := clamp(m.cfg.SidebarWidth, 12, 60)
		maxW := max(12, totalW/2)
		return min(w, maxW)
	}
	return max(18, totalW/5)
}

const secretColSep = "│"

func (m Model) secretColumnWidths(panelW int) (nameW, valueW int) {
	chrome := m.panelChrome()
	inner := max(10, panelW-chrome)
	avail := max(9, inner-lipgloss.Width(secretColSep))
	pct := m.cfg.NameColumnPercent
	if pct < 1 || pct > 99 {
		pct = 40
	}
	nameW = max(12, avail*pct/100)
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

	secrets := m.renderSecretsTable(layout.secrets.w, layout.secrets.h)
	var main string
	if !m.cfg.Sidebar {
		main = secrets
	} else {
		projects := m.renderProjectTree(layout.projects.w, layout.projects.h)
		if m.cfg.SidebarPosition == "right" {
			main = lipgloss.JoinHorizontal(lipgloss.Top, secrets, projects)
		} else {
			main = lipgloss.JoinHorizontal(lipgloss.Top, projects, secrets)
		}
	}
	status := m.renderStatus(layout.status.w)

	var base string
	if layout.suggest.h > 0 {
		suggest := m.renderCompletions(layout.suggest.w, layout.suggest.h)
		base = lipgloss.JoinVertical(lipgloss.Left, main, suggest, status)
	} else {
		base = lipgloss.JoinVertical(lipgloss.Left, main, status)
	}

	switch m.focus {
	case focusHelp:
		base = m.renderOverlay(base, m.renderHelpModal())
	case focusSave:
		base = m.renderOverlay(base, m.renderSaveModal())
	case focusSwitchConfirm:
		base = m.renderOverlay(base, m.renderSwitchConfirmModal())
	}

	return appStyle.Width(m.width).Height(m.height).Render(base)
}

func (m Model) renderProjectTree(width, height int) string {
	var searchRe *regexp.Regexp
	if m.searchPane == focusProjects {
		searchRe = m.searchRe
	}
	lines := make([]string, len(m.tree))
	needles := make([]string, len(m.tree))
	for i, row := range m.tree {
		needles[i] = searchableTreeText(row)
		line := formatTreeRow(row, m.activeProject, m.activeConfig, m.hasDirtySecrets())
		if searchRe != nil {
			line = highlightNeedleInDisplay(ansi.Strip(line), needles[i], searchRe, baseTextStyle())
		}
		lines[i] = line
	}
	title := fmt.Sprintf("Projects (%d)", len(m.projects))
	selStyle := selectedStyle
	if row, ok := m.currentTreeRow(); ok && row.kind == treeConfig && row.project == m.activeProject && row.config == m.activeConfig {
		selStyle = selectedActiveStyle
	}
	return m.renderLinesPanel(title, lines, needles, m.treeIdx, m.focus == focusProjects, width, height, searchRe, selStyle)
}

func (m Model) renderSecretsTable(width, height int) string {
	chrome := m.panelChrome()
	innerH := max(1, height-chrome)
	idxs := m.filteredIndexes()
	bodyH := max(1, innerH-1)
	showSB := m.cfg.ListScrollbarVertical && len(idxs) > bodyH
	contentW := width
	if showSB {
		contentW = max(12, width-1)
	}
	nameW, valueW := m.secretColumnWidths(contentW)
	sep := secretColSeparator()
	header := dimStyle.Render(padRight("  NAME", nameW)) + sep + dimStyle.Render(padRight("VALUE", valueW))

	start := clampScrollOffset(m.secretOffset, m.secretIdx, bodyH, len(idxs))
	end := start + bodyH
	if end > len(idxs) {
		end = len(idxs)
	}

	var searchRe *regexp.Regexp
	if m.searchPane == focusSecrets {
		searchRe = m.searchRe
	}
	secretsActive := m.focus == focusSecrets || m.focus == focusSecretInsert

	var rows []string
	rows = append(rows, header)
	for i := start; i < end; i++ {
		s := m.secrets[idxs[i]]
		rows = append(rows, m.renderSecretRow(s, i, nameW, valueW, secretsActive, searchRe))
	}
	for len(rows) < innerH {
		rows = append(rows, "")
	}

	body := strings.Join(rows[:innerH], "\n")
	body = joinWithScrollbar(body, innerH, len(idxs), start, bodyH, showSB)

	title := fmt.Sprintf("Secrets (%d)", len(idxs))
	if m.activeProject != "" && m.activeConfig != "" {
		title = fmt.Sprintf("Secrets (%d) [%s / %s]", len(idxs), m.activeProject, m.activeConfig)
	}
	return m.renderTitledListPanel(title, body, width, height, secretsActive)
}

func (m Model) renderSecretRow(s secretRow, listIdx, nameW, valueW int, paneActive bool, searchRe *regexp.Regexp) string {
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

	var nameCell, valueCell string
	if editing && m.secretCol == colName {
		m.cellInput.Width = max(1, nameW-lipgloss.Width(marker))
		m.cellInput.TextStyle = selectedStyle
		m.cellInput.Cursor.Style = selectedStyle
		m.cellInput.Cursor.TextStyle = selectedStyle
		nameCell = padStyledCell(marker+m.cellInput.View(), nameW, true)
	} else {
		nameCell = styleSecretCell(marker+nameText, nameW, nameActive, s.shouldDelete, s.isDirty(), searchRe)
	}

	if editing && m.secretCol == colValue {
		m.cellInput.Width = max(1, valueW)
		m.cellInput.TextStyle = selectedStyle
		m.cellInput.Cursor.Style = selectedStyle
		m.cellInput.Cursor.TextStyle = selectedStyle
		valueCell = padStyledCell(m.cellInput.View(), valueW, true)
	} else {
		valueCell = styleSecretCell(valueText, valueW, valueActive, s.shouldDelete, s.isDirty(), searchRe)
	}

	return nameCell + secretColSeparator() + valueCell
}

func padStyledCell(styled string, width int, selected bool) string {
	if selected {
		return padWithStyle(styled, width, selectedStyle)
	}
	fill := lipgloss.NewStyle()
	if background != "" {
		fill = fill.Background(background)
	}
	return padWithStyle(styled, width, fill)
}

func padWithStyle(styled string, width int, fill lipgloss.Style) string {
	w := lipgloss.Width(styled)
	if w > width {
		return lipgloss.NewStyle().MaxWidth(width).Render(styled)
	}
	return styled + fill.Render(strings.Repeat(" ", width-w))
}

func (s secretRow) previewValueUnlimited() string {
	v := s.displayValue()
	return strings.ReplaceAll(v, "\n", " ")
}

func styleSecretCell(text string, width int, active, del, dirty bool, re *regexp.Regexp) string {
	plain := truncate(ansi.Strip(text), width)
	var styled string
	switch {
	case active:
		styled = highlightMatches(plain, re, selectedStyle)
		return padStyledCell(styled, width, true)
	case del:
		styled = highlightMatches(plain, re, deleteStyle)
	case dirty:
		styled = highlightMatches(plain, re, dirtyStyle)
	default:
		styled = highlightMatches(plain, re, baseTextStyle())
	}
	return padStyledCell(styled, width, false)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m Model) renderLinesPanel(title string, lines, needles []string, selected int, active bool, width, height int, searchRe *regexp.Regexp, selStyle lipgloss.Style) string {
	chrome := m.panelChrome()
	innerH := max(1, height-chrome)
	visible := innerH
	showSB := m.cfg.SidebarScrollbarVertical && len(lines) > visible
	innerW := max(1, width-chrome)
	if showSB {
		innerW = max(1, innerW-1)
	}

	start := clampScrollOffset(m.treeOffset, selected, visible, len(lines))
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	var body []string
	for i := start; i < end; i++ {
		var line string
		if active && i == selected {
			plain := truncate(ansi.Strip(lines[i]), innerW)
			needle := ""
			if i < len(needles) {
				needle = needles[i]
			}
			if searchRe != nil {
				line = padWithStyle(highlightNeedleInDisplay(plain, needle, searchRe, selStyle), innerW, selStyle)
			} else {
				line = selStyle.Width(innerW).MaxWidth(innerW).Render(plain)
			}
		} else {
			line = padStyledCell(truncatePreserve(lines[i], innerW), innerW, false)
		}
		body = append(body, line)
	}
	for len(body) < innerH {
		body = append(body, padStyledCell("", innerW, false))
	}

	content := strings.Join(body[:innerH], "\n")
	content = joinWithScrollbar(content, innerH, len(lines), start, visible, showSB)
	return m.renderTitledListPanel(title, content, width, height, active)
}

func (m Model) renderStatus(width int) string {
	switch {
	case m.focus == focusSearch:
		return renderStatusInput(&m.searchInput, width)
	case m.focus == focusCommand:
		return renderStatusInput(&m.commandInput, width)
	case m.focus == focusCreateConfig:
		return renderStatusInput(&m.createConfigInput, width)
	case m.focus == focusFilter:
		return renderStatusInput(&m.filterInput, width)
	}

	var left string
	if m.fetching {
		left = m.spinner.View() + " Loading…"
	} else if m.errMsg != "" {
		left = errorStyle.Render(m.errMsg)
	} else if m.statusMsg != "" {
		left = statusStyle.Render(m.statusMsg)
	}

	var right string
	switch {
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

func renderStatusInput(ti *textinput.Model, width int) string {
	text := lipgloss.NewStyle().Foreground(textColor)
	cursorStyle := lipgloss.NewStyle().Foreground(textColor)
	if background != "" {
		text = text.Background(background)
		cursorStyle = cursorStyle.Background(background)
	}
	ti.TextStyle = text
	ti.PromptStyle = helpStyle
	ti.PlaceholderStyle = helpStyle
	ti.Cursor.Style = cursorStyle
	ti.Cursor.TextStyle = text
	_ = ti.Cursor.SetMode(cursor.CursorStatic)

	// textinput's placeholder path pads with unstyled spaces; keep Width=0 when
	// empty so statusBarStyle can fill the full row with the theme background.
	promptW := lipgloss.Width(ti.PromptStyle.Render(ti.Prompt))
	avail := max(1, width-promptW)
	if ti.Value() == "" {
		ti.Width = 0
	} else {
		ti.Width = avail
	}

	view := ti.View()
	if lipgloss.Width(view) > width {
		view = lipgloss.NewStyle().MaxWidth(width).Render(view)
	}
	return statusBarStyle.Width(width).MaxWidth(width).Render(view)
}

func (m Model) renderCompletions(width, height int) string {
	if height < 2 || len(m.completions.Items) == 0 {
		return ""
	}
	innerH := max(1, height-2)
	m.completions.ViewportH = innerH
	m.completions.EnsureVisible()

	title := "suggestions"
	innerW := max(1, width-2)
	boxStyle := lipgloss.NewStyle().
		Border(roundedBorder).
		BorderForeground(accent).
		Width(innerW).
		Height(innerH)
	if background != "" {
		boxStyle = boxStyle.Background(background).BorderBackground(background)
	}

	start := m.completions.Scroll
	end := min(start+innerH, len(m.completions.Items))
	var lines []string
	for i := start; i < end; i++ {
		item := m.completions.Items[i]
		selected := m.completions.Selected != nil && *m.completions.Selected == i
		lines = append(lines, renderCompletionRow(item, selected, innerW))
	}
	body := strings.Join(lines, "\n")
	rendered := boxStyle.Render(body)
	outLines := strings.Split(rendered, "\n")
	if len(outLines) > 0 {
		outLines[0] = titledTopBorder(title, width, true)
	}
	return strings.Join(outLines, "\n")
}

func renderCompletionRow(item Suggestion, selected bool, width int) string {
	inner := max(1, width)
	marker := "  "
	if selected {
		marker = "▸ "
	}
	label := marker + padRight(item.Label, 18)
	helpBudget := max(0, inner-lipgloss.Width(label)-1)
	help := truncate(item.Help, helpBudget)
	if selected {
		return selectedStyle.Width(inner).MaxWidth(inner).Render(padRight(label+" "+help, inner))
	}
	bg := lipgloss.NewStyle()
	if background != "" {
		bg = bg.Background(background)
	}
	pad := max(0, inner-lipgloss.Width(label)-1-lipgloss.Width(help))
	return statusStyle.Bold(true).Render(label) +
		bg.Render(" ") +
		helpStyle.Render(help) +
		bg.Render(strings.Repeat(" ", pad))
}

func (m Model) renderTitledPanel(title, content string, width, height int, active bool) string {
	style := panelStyle
	if active {
		style = activePanelStyle
	}
	return m.renderTitledPanelStyle(title, content, width, height, active, style)
}

func (m Model) renderTitledListPanel(title, content string, width, height int, active bool) string {
	style := listPanelStyle
	if active {
		style = activeListPanelStyle
	}
	return m.renderTitledPanelStyle(title, content, width, height, active, style)
}

func (m Model) renderTitledPanelStyle(title, content string, width, height int, active bool, style lipgloss.Style) string {
	if !m.cfg.Border {
		style = style.UnsetBorderStyle().Border(lipgloss.HiddenBorder(), false)
		titleLine := titledTopPlain(title, width, active)
		bodyH := max(1, height-1)
		body := style.Width(max(1, width)).Height(bodyH).Render(content)
		return titleLine + "\n" + body
	}
	rendered := style.Width(max(1, width-2)).Height(max(1, height-2)).Render(content)
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}
	lines[0] = titledTopBorder(title, width, active)
	return strings.Join(lines, "\n")
}

func titledTopPlain(title string, width int, active bool) string {
	tStyle := titleStyle
	if active {
		tStyle = activeTitleStyle
	}
	if background != "" {
		tStyle = tStyle.Background(background)
	}
	label := strings.TrimSpace(title)
	if lipgloss.Width(label) > width {
		label = truncate(label, width)
	}
	pad := max(0, width-lipgloss.Width(label))
	line := tStyle.Render(label)
	if pad > 0 {
		padStyle := lipgloss.NewStyle()
		if background != "" {
			padStyle = padStyle.Background(background)
		}
		line += padStyle.Render(strings.Repeat(" ", pad))
	}
	return line
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

func (m Model) helpModalButtons() []modalButton {
	return []modalButton{{Label: "Close", Key: "esc"}}
}

func (m Model) saveModalButtons() []modalButton {
	if len(m.pendingChanges) == 0 {
		return []modalButton{{Label: "Close", Key: "esc"}}
	}
	return []modalButton{
		{Label: "Confirm", Key: "enter"},
		{Label: "Cancel", Key: "esc"},
	}
}

func (m Model) switchConfirmButtons() []modalButton {
	return []modalButton{
		{Label: "Save", Key: "s"},
		{Label: "Discard", Key: "d"},
		{Label: "Cancel", Key: "c"},
	}
}

func (m Model) modalInnerWidth(width int) int {
	if m.cfg.Border {
		return max(1, width-4)
	}
	return max(1, width-2)
}

func (m Model) renderHelpModal() string {
	width := min(64, m.width-4)
	body := m.helpViewport.View() + "\n\n" + m.renderModalButtons(m.helpModalButtons(), m.modalInnerWidth(width))
	return m.renderTitledPanel("Help", body, width, min(28, m.height-4), true)
}

func (m Model) renderSaveModal() string {
	width := min(60, m.width-4)
	var body string
	if len(m.pendingChanges) == 0 {
		body = "There are no changes to save\n\n"
	} else {
		body = "The following secrets will be updated:\n\n"
		for _, c := range m.pendingChanges {
			body += "● " + fmt.Sprint(c.Name) + "\n"
		}
		body += "\n"
	}
	body += m.renderModalButtons(m.saveModalButtons(), m.modalInnerWidth(width))
	return m.renderTitledPanel("Confirm Changes", body, width, min(20, m.height-4), true)
}

func (m Model) renderSwitchConfirmModal() string {
	width := min(60, max(24, m.width-4))
	inner := m.modalInnerWidth(width)
	var lines []string
	if len(m.pendingChanges) > 0 {
		lines = append(lines, "Modified:")
		for _, c := range m.pendingChanges {
			lines = append(lines, "● "+fmt.Sprint(c.Name))
		}
	} else {
		lines = append(lines, "No modifications")
	}
	body := strings.Join(lines, "\n") + "\n\n" + m.renderModalButtons(m.switchConfirmButtons(), inner)
	height := min(lipgloss.Height(body)+m.panelChrome(), max(4, m.height-2))
	return m.renderTitledPanel("Unsaved Changes", body, width, height, true)
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
