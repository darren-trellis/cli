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

	"github.com/gdamore/tcell/v2"
)

// Pane drawing. Every pane is laid out by computeLayout and painted straight
// into tcell cells, so what is drawn and what the mouse handler hit-tests are
// derived from the same geometry.

// paneBox paints a pane's frame and returns the inner drawing area. With
// borders off the pane keeps a plain title row instead, which is why
// panelChrome() differs between the two.
func (m Model) paneBox(sc tcell.Screen, r rect, title string, active bool) rect {
	fillRect(sc, r.x, r.y, r.w, r.h, baseStyle)

	titleSt := titleStyle
	borderSt := borderStyle
	if active {
		titleSt = activeTitleStyle
		borderSt = activeBorderStyle
	}

	if m.cfg.Border {
		drawBox(sc, r.x, r.y, r.w, r.h, title, borderSt, titleSt)
		return rect{x: r.x + 1, y: r.y + 1, w: max(1, r.w-2), h: max(1, r.h-2)}
	}
	drawCell(sc, r.x, r.y, r.w, title, titleSt)
	return rect{x: r.x, y: r.y + 1, w: max(1, r.w), h: max(1, r.h-1)}
}

func (m Model) drawProjectsPane(sc tcell.Screen, r rect) {
	active := m.focus == focusProjects
	inner := m.paneBox(sc, r, m.projectsTitle(r.w), active)

	var searchRe *regexp.Regexp
	if m.searchPane == focusProjects && !m.searchGlobal {
		searchRe = m.searchRe
	}

	visible := inner.h
	showSB := m.cfg.SidebarScrollbarVertical && len(m.tree) > visible
	bodyW := inner.w
	if showSB {
		bodyW = max(1, bodyW-1)
	}

	start := clampScrollOffset(m.treeOffset, m.treeIdx, visible, len(m.tree))
	for i := 0; i < visible; i++ {
		idx := start + i
		y := inner.y + i
		if idx >= len(m.tree) {
			fillRow(sc, inner.x, y, inner.w, baseStyle)
			continue
		}
		row := m.tree[idx]
		lead, label, trail := treeRowParts(row)
		text := lead + label + trail
		selected := idx == m.treeIdx
		style := treeRowStyle(selected)

		if selected {
			text = padRight(text, bodyW)
		}
		spans := spansForNeedle(text, searchableTreeText(row), searchRe)
		var decor []runeSpan
		if !selected {
			leadN, labelN := len([]rune(lead)), len([]rune(label))
			decor = []runeSpan{{0, leadN}, {leadN + labelN, leadN + labelN + len([]rune(trail))}}
		}
		drawTreeText(sc, inner.x, y, bodyW, truncate(text, bodyW), style, borderStyle, decor, spans)
	}

	if showSB {
		drawVerticalScrollbar(sc, inner.x+inner.w-1, inner.y, visible, len(m.tree), start, visible)
	}
}

func treeRowStyle(selected bool) tcell.Style {
	if selected {
		return selectedStyle
	}
	return baseStyle
}

// drawSecretsPane paints the secrets table and returns the cursor position when
// a cell is being edited, or (-1, -1).
func (m Model) drawSecretsPane(sc tcell.Screen, r rect) (int, int) {
	active := m.focus == focusSecrets || m.focus == focusSecretInsert
	idxs := m.filteredIndexes()

	title := fmt.Sprintf("Secrets (%d)", len(idxs))
	if m.activeProject != "" && m.activeConfig != "" {
		title = fmt.Sprintf("Secrets (%d) [%s / %s]", len(idxs), m.activeProject, m.activeConfig)
	}
	inner := m.paneBox(sc, r, title, active)

	bodyH := max(1, inner.h-1)
	showSB := m.cfg.ListScrollbarVertical && len(idxs) > bodyH
	contentW := r.w
	if showSB {
		contentW = max(12, r.w-1)
	}
	nameW, valueW := m.secretColumnWidths(contentW)
	sepW := textWidth(secretColSep)

	// Header
	drawCell(sc, inner.x, inner.y, nameW, padRight("  NAME", nameW), dimStyle)
	drawCell(sc, inner.x+nameW, inner.y, sepW, secretColSep, dimStyle)
	drawCell(sc, inner.x+nameW+sepW, inner.y, valueW, padRight("VALUE", valueW), dimStyle)

	var searchRe *regexp.Regexp
	if m.searchPane == focusSecrets && !m.searchGlobal {
		searchRe = m.searchRe
	}

	start := clampScrollOffset(m.secretOffset, m.secretIdx, bodyH, len(idxs))
	cursorX, cursorY := -1, -1

	for i := 0; i < bodyH; i++ {
		listIdx := start + i
		y := inner.y + 1 + i
		if listIdx >= len(idxs) {
			fillRow(sc, inner.x, y, inner.w, baseStyle)
			continue
		}
		cx, cy := m.drawSecretRow(sc, m.secrets[idxs[listIdx]], listIdx, inner.x, y, nameW, valueW, active, searchRe)
		if cx >= 0 {
			cursorX, cursorY = cx, cy
		}
	}

	if showSB {
		drawVerticalScrollbar(sc, inner.x+inner.w-1, inner.y+1, bodyH, len(idxs), start, bodyH)
	}
	return cursorX, cursorY
}

func (m Model) drawSecretRow(sc tcell.Screen, s secretRow, listIdx, x, y, nameW, valueW int, paneActive bool, searchRe *regexp.Regexp) (int, int) {
	marker := "  "
	switch {
	case s.shouldDelete:
		marker = "D "
	case s.isDirty():
		marker = "* "
	}

	rowSelected := paneActive && listIdx == m.secretIdx
	editing := m.focus == focusSecretInsert && listIdx == m.secretIdx
	sepW := textWidth(secretColSep)
	valueX := x + nameW + sepW

	cursorX, cursorY := -1, -1

	// Name cell
	if editing && m.secretCol == colName {
		markerW := textWidth(marker)
		drawCell(sc, x, y, markerW, marker, selectedStyle)
		field := m.cellInput
		if cx := field.draw(sc, x+markerW, y, max(1, nameW-markerW), selectedStyle, selectedStyle, selectedStyle); cx >= 0 {
			cursorX, cursorY = cx, y
		}
	} else {
		m.drawSecretCell(sc, x, y, nameW, marker+s.name, s, rowSelected && m.secretCol == colName, searchRe)
	}

	// Separator
	sepStyle := dimStyle
	drawCell(sc, x+nameW, y, sepW, secretColSep, sepStyle)

	// Value cell
	if editing && m.secretCol == colValue {
		field := m.cellInput
		if cx := field.draw(sc, valueX, y, valueW, selectedStyle, selectedStyle, selectedStyle); cx >= 0 {
			cursorX, cursorY = cx, y
		}
	} else {
		valueText := flattenForCell(s.previewValueUnlimited())
		if len([]rune(valueText)) > 200 {
			valueText = string([]rune(valueText)[:200]) + "…"
		}
		m.drawSecretCell(sc, valueX, y, valueW, valueText, s, rowSelected && m.secretCol == colValue, searchRe)
	}

	return cursorX, cursorY
}

func (m Model) drawSecretCell(sc tcell.Screen, x, y, w int, text string, s secretRow, cellActive bool, re *regexp.Regexp) {
	var style tcell.Style
	switch {
	case cellActive:
		style = selectedStyle
	case s.shouldDelete:
		style = deleteStyle
	case s.isDirty():
		style = dirtyStyle
	default:
		style = baseStyle
	}
	text = truncate(text, w)
	if cellActive {
		text = padRight(text, w)
	}
	drawHighlighted(sc, x, y, w, text, style, searchHitStyle, re)
}

// drawStatus paints the bottom row, which doubles as the prompt for the search,
// command, filter and create-config inputs. Returns the cursor position when an
// input is focused.
func (m Model) drawStatus(sc tcell.Screen, r rect) (int, int) {
	fillRow(sc, r.x, r.y, r.w, baseStyle)

	var field *textField
	switch m.focus {
	case focusSearch:
		f := m.searchInput
		field = &f
	case focusCommand:
		f := m.commandInput
		field = &f
	case focusCreateConfig:
		f := m.createConfigInput
		field = &f
	case focusFilter:
		f := m.filterInput
		field = &f
	}
	if field != nil {
		field.Focus()
		if cx := field.draw(sc, r.x, r.y, r.w, baseStyle, helpStyle, helpStyle); cx >= 0 {
			return cx, r.y
		}
		return -1, -1
	}

	left, leftStyle := "", statusStyle
	switch {
	case m.fetching:
		left = m.spinner.View() + " Loading…"
	case m.highlightLoading != "":
		project, config, _ := splitSecretsCacheKey(m.highlightLoading)
		left = m.spinner.View() + " Loading " + project + " / " + config + "…"
	case m.workplaceIndexing && (m.searchGlobal || m.focus == focusSearch):
		left = m.spinner.View() + " Indexing secret names…"
	case m.errMsg != "":
		left, leftStyle = m.errMsg, errorStyle
	case m.statusMsg != "":
		left = m.statusMsg
	}

	right, rightStyle := "", activeEnvStyle
	filters := m.filterStatusLabel()
	switch {
	case m.searchQuery != "":
		right, rightStyle = m.searchStatusLabel(), searchHitStyle
	case filters != "":
		right = filters
	}

	// Filters keep their own dim prefix when a search label is also showing.
	prefix := ""
	if m.searchQuery != "" && filters != "" {
		prefix = filters + "  "
	}

	rightW := textWidth(prefix) + textWidth(right)
	leftW := min(textWidth(left), max(0, r.w-rightW-1))
	drawText(sc, r.x, r.y, leftW, truncate(left, leftW), leftStyle)

	if rightW > 0 && rightW <= r.w {
		rx := r.x + r.w - rightW
		if prefix != "" {
			rx += drawText(sc, rx, r.y, textWidth(prefix), prefix, helpStyle)
		}
		drawText(sc, rx, r.y, textWidth(right), right, rightStyle)
	}
	return -1, -1
}

// projectsTitle shows the project count and any active project filter. A
// narrow sidebar drops the "Projects" word before it would cut off the filter.
func (m Model) projectsTitle(paneW int) string {
	n := m.visibleProjectCount()
	if m.projectFilter == "" {
		return fmt.Sprintf("Projects (%d)", n)
	}
	full := fmt.Sprintf("Projects (%d) /%s", n, m.projectFilter)
	avail := paneW
	if m.cfg.Border {
		avail -= 4
	}
	if textWidth(full) <= avail {
		return full
	}
	return fmt.Sprintf("/%s (%d)", m.projectFilter, n)
}

func (m Model) visibleProjectCount() int {
	n := 0
	for _, row := range m.tree {
		if row.kind == treeProject {
			n++
		}
	}
	return n
}

func (m Model) filterStatusLabel() string {
	var parts []string
	if m.globalFilter != "" {
		parts = append(parts, "F "+m.globalFilter)
	}
	if m.filter != "" {
		parts = append(parts, "f "+m.filter)
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + "  " + parts[1]
}

// drawCompletions paints the command suggestion box above the status row.
func (m Model) drawCompletions(sc tcell.Screen, r rect) {
	if r.h < 2 || len(m.completions.Items) == 0 {
		return
	}
	fillRect(sc, r.x, r.y, r.w, r.h, baseStyle)
	drawBox(sc, r.x, r.y, r.w, r.h, "suggestions", activeBorderStyle, activeTitleStyle)

	innerX, innerY := r.x+1, r.y+1
	innerW, innerH := max(1, r.w-2), max(1, r.h-2)

	state := m.completions
	state.ViewportH = innerH
	state.EnsureVisible()

	for i := 0; i < innerH; i++ {
		idx := state.Scroll + i
		y := innerY + i
		if idx >= len(state.Items) {
			fillRow(sc, innerX, y, innerW, baseStyle)
			continue
		}
		item := state.Items[idx]
		selected := state.Selected != nil && *state.Selected == idx

		marker := "  "
		if selected {
			marker = "▸ "
		}
		label := marker + padRight(item.Label, 18)
		if selected {
			line := padRight(label+" "+item.Help, innerW)
			drawCell(sc, innerX, y, innerW, line, selectedStyle)
			continue
		}
		fillRow(sc, innerX, y, innerW, baseStyle)
		w := drawText(sc, innerX, y, innerW, label, statusStyle.Bold(true))
		w += drawText(sc, innerX+w, y, max(0, innerW-w), " ", baseStyle)
		drawText(sc, innerX+w, y, max(0, innerW-w), item.Help, helpStyle)
	}
}

// draw paints the whole frame and returns the cursor position, or (-1, -1) when
// the cursor should be hidden.
func (m Model) draw(sc tcell.Screen) (int, int) {
	if m.width == 0 || m.height == 0 {
		return -1, -1
	}
	fillRect(sc, 0, 0, m.width, m.height, baseStyle)

	layout := m.computeLayout()
	cursorX, cursorY := m.drawSecretsPane(sc, layout.secrets)
	if m.cfg.Sidebar {
		m.drawProjectsPane(sc, layout.projects)
	}
	if layout.suggest.h > 0 {
		m.drawCompletions(sc, layout.suggest)
	}
	if cx, cy := m.drawStatus(sc, layout.status); cx >= 0 {
		cursorX, cursorY = cx, cy
	}
	if r, ok := m.secretRefPopupRect(cursorX, cursorY); ok {
		m.drawCompletions(sc, r)
	}

	if spec, ok := m.currentModalSpec(); ok {
		spec.draw(sc, m.width, m.height, m.modalBtnIdx, m.cfg.Border)
		// A modal owns the screen; no cell cursor behind it.
		cursorX, cursorY = -1, -1
	}
	return cursorX, cursorY
}
