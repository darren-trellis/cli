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
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Modals are described as data and drawn by modalPane. Keeping the description
// separate from the drawing is what lets the tests assert on modal contents
// without a screen, and lets mouse hit-testing use real row coordinates instead
// of searching rendered text.

type modalButton struct {
	Label string
	Key   string
}

func (b modalButton) text() string {
	if b.Key == "" {
		return b.Label
	}
	return b.Label + " (" + b.Key + ")"
}

type modalLineKind int

const (
	modalLinePlain modalLineKind = iota
	modalLineDim
	modalLineSelected
)

type modalLine struct {
	text string
	kind modalLineKind
	// row is the propagate-target index this line represents, or -1.
	row int
}

func plainLine(s string) modalLine { return modalLine{text: s, row: -1} }
func dimLine(s string) modalLine   { return modalLine{text: s, kind: modalLineDim, row: -1} }

type modalSpec struct {
	title   string
	lines   []modalLine
	buttons []modalButton
	width   int
	height  int
	// scroll, when set, draws a scrollable body (the help modal) instead of lines.
	scroll *textScroll
}

func (m Model) helpModalButtons() []modalButton {
	return []modalButton{{Label: "Close", Key: "esc"}}
}

func (m Model) saveModalButtons() []modalButton {
	if len(m.pendingChanges) == 0 {
		return []modalButton{{Label: "Close", Key: "esc"}}
	}
	return []modalButton{
		{Label: "Save", Key: "y"},
		{Label: "Cancel", Key: "n"},
	}
}

func (m Model) propagateModalButtons() []modalButton {
	return []modalButton{
		{Label: "Apply", Key: "y"},
		{Label: "Cancel", Key: "c"},
	}
}

func (m Model) deleteConfirmButtons() []modalButton {
	return []modalButton{
		{Label: "Delete", Key: "d"},
		{Label: "Cancel", Key: "c"},
	}
}

func (m Model) switchConfirmButtons() []modalButton {
	return []modalButton{
		{Label: "Save", Key: "s"},
		{Label: "Discard", Key: "d"},
		{Label: "Cancel", Key: "c"},
	}
}

func (m Model) currentModalButtons() []modalButton {
	switch m.focus {
	case focusHelp:
		return m.helpModalButtons()
	case focusSave:
		return m.saveModalButtons()
	case focusSwitchConfirm:
		return m.switchConfirmButtons()
	case focusDeleteConfirm:
		return m.deleteConfirmButtons()
	case focusPropagate:
		return m.propagateModalButtons()
	default:
		return nil
	}
}

// modalChrome is the number of rows/columns the modal border and padding take.
func (m Model) modalChrome() int {
	if m.cfg.Border {
		return 2
	}
	return 1
}

// currentModalSpec describes the modal for the current focus, if any.
func (m Model) currentModalSpec() (modalSpec, bool) {
	switch m.focus {
	case focusHelp:
		return m.helpModalSpec(), true
	case focusSave:
		return m.saveModalSpec(), true
	case focusSwitchConfirm:
		return m.switchConfirmModalSpec(), true
	case focusDeleteConfirm:
		return m.deleteConfirmModalSpec(), true
	case focusPropagate:
		return m.propagateModalSpec(), true
	default:
		return modalSpec{}, false
	}
}

func (m Model) modalWidth(preferred, floor int) int {
	return min(preferred, max(floor, m.width-4))
}

// bodyHeight sizes a modal to its content: body lines, a blank row, the button
// row, and the border, capped to the terminal.
func (m Model) modalHeight(bodyLines int) int {
	return min(bodyLines+2+m.modalChrome(), max(4, m.height-2))
}

func (m Model) helpModalSpec() modalSpec {
	width := m.modalWidth(64, 24)
	vp := m.helpViewport
	return modalSpec{
		title:   "Help",
		buttons: m.helpModalButtons(),
		width:   width,
		height:  min(28, max(6, m.height-4)),
		scroll:  &vp,
	}
}

func (m Model) saveModalSpec() modalSpec {
	var lines []modalLine
	if len(m.pendingChanges) > 0 {
		lines = append(lines, plainLine("Modified:"))
		for _, c := range m.pendingChanges {
			lines = append(lines, plainLine("● "+c.Name))
		}
	} else {
		lines = append(lines, plainLine("No modifications"))
	}
	return modalSpec{
		title:   "Confirm Changes",
		lines:   lines,
		buttons: m.saveModalButtons(),
		width:   m.modalWidth(60, 24),
		height:  m.modalHeight(len(lines)),
	}
}

func (m Model) switchConfirmModalSpec() modalSpec {
	var lines []modalLine
	switch {
	case len(m.quitDirty) > 0:
		lines = append(lines, plainLine("Modified:"))
		for _, g := range m.quitDirty {
			lines = append(lines, plainLine(g.project+" / "+g.config))
			for _, c := range g.changes {
				lines = append(lines, plainLine("● "+c.Name))
			}
		}
	case len(m.pendingChanges) > 0:
		lines = append(lines, plainLine("Modified:"))
		for _, c := range m.pendingChanges {
			lines = append(lines, plainLine("● "+c.Name))
		}
	default:
		lines = append(lines, plainLine("No modifications"))
	}
	return modalSpec{
		title:   "Unsaved Changes",
		lines:   lines,
		buttons: m.switchConfirmButtons(),
		width:   m.modalWidth(60, 24),
		height:  m.modalHeight(len(lines)),
	}
}

func (m Model) deleteConfirmModalSpec() modalSpec {
	var lines []modalLine
	title := "Delete Config"
	if m.deletingProject() {
		title = "Delete Project"
		lines = append(lines, plainLine("Delete project and all of its configs?"))
		if m.pendingDeleteProject != "" {
			lines = append(lines, plainLine("● "+m.pendingDeleteProject))
		}
		lines = append(lines, plainLine("This cannot be undone."))
		if m.projectHasUnsavedSecrets(m.pendingDeleteProject) {
			lines = append(lines, plainLine("Unsaved secret changes will be discarded."))
		}
	} else {
		target := m.pendingDeleteConfig
		if m.pendingDeleteProject != "" && target != "" {
			target = m.pendingDeleteProject + " / " + target
		}
		lines = append(lines, plainLine("Delete config?"))
		if target != "" {
			lines = append(lines, plainLine("● "+target))
		}
	}
	return modalSpec{
		title:   title,
		lines:   lines,
		buttons: m.deleteConfirmButtons(),
		width:   m.modalWidth(60, 24),
		height:  m.modalHeight(len(lines)),
	}
}

func (m Model) propagateModalSpec() modalSpec {
	lines := []modalLine{
		plainLine("Also apply these changes to other root configs?"),
		plainLine(""),
	}
	for i, t := range m.propagateTargets {
		box := "[ ]"
		if t.on {
			box = "[x]"
		}
		label := box + " " + t.name
		if t.dirty {
			label += " (unsaved)"
		}
		line := modalLine{text: label, row: i}
		switch {
		case i == m.propagateIdx:
			line.kind = modalLineSelected
		case !t.selectable():
			line.kind = modalLineDim
		}
		lines = append(lines, line)
	}
	rewriteIdx := len(m.propagateTargets)
	rewriteBox := "[ ]"
	if m.propagateRewriteRefs {
		rewriteBox = "[x]"
	}
	rewrite := modalLine{text: rewriteBox + " Rewrite references for each environment", row: rewriteIdx}
	if m.propagateIdx == rewriteIdx {
		rewrite.kind = modalLineSelected
	}
	lines = append(lines, plainLine(""), rewrite)
	lines = append(lines, plainLine(""), dimLine("Space toggles · a all except unsaved · r rewrite refs"))
	return modalSpec{
		title:   "Apply to other environments",
		lines:   lines,
		buttons: m.propagateModalButtons(),
		width:   m.modalWidth(60, 28),
		height:  m.modalHeight(len(lines)),
	}
}

func (m Model) projectHasUnsavedSecrets(project string) bool {
	if project == "" {
		return false
	}
	for _, g := range m.dirtyGroups() {
		if g.project == project {
			return true
		}
	}
	return false
}

// --- button layout -------------------------------------------------------

func labelWidths(labels []string) []int {
	out := make([]int, len(labels))
	for i, label := range labels {
		out[i] = max(1, textWidth(label))
	}
	return out
}

// modalButtonLayout distributes total columns across the buttons, returning
// each button's width and the gap between them.
func modalButtonLayout(mins []int, total int) ([]int, int) {
	n := len(mins)
	out := make([]int, n)
	if n == 0 {
		return out, 0
	}
	if total <= 0 {
		for i := range out {
			out[i] = 1
		}
		return out, 0
	}

	gap := 1
	usable := total - (n-1)*gap
	if usable < n {
		gap = 0
		usable = total
	}
	if usable < n {
		for i := range out {
			out[i] = 1
		}
		return out, 0
	}

	sumMin := 0
	for i, w := range mins {
		if w < 1 {
			w = 1
			mins[i] = 1
		}
		sumMin += w
	}
	if sumMin <= usable {
		extra := usable - sumMin
		base := extra / n
		rem := extra % n
		for i := range out {
			out[i] = mins[i] + base
			if i < rem {
				out[i]++
			}
		}
		return out, gap
	}

	base := usable / n
	rem := usable % n
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out, gap
}

// buttonRects lays the buttons out across width columns starting at (x, y).
func buttonRects(buttons []modalButton, x, y, width int) []rect {
	if len(buttons) == 0 {
		return nil
	}
	labels := make([]string, len(buttons))
	for i, b := range buttons {
		labels[i] = b.text()
	}
	widths, gap := modalButtonLayout(labelWidths(labels), width)

	rowW := 0
	for i, w := range widths {
		if i > 0 {
			rowW += gap
		}
		rowW += w
	}
	cx := x + max(0, (width-rowW)/2)

	rects := make([]rect, len(buttons))
	for i, w := range widths {
		rects[i] = rect{x: cx, y: y, w: w, h: 1}
		cx += w + gap
	}
	return rects
}

func (m *Model) cycleModalButton(n int, delta int) {
	if n <= 0 {
		m.modalBtnIdx = 0
		return
	}
	m.modalBtnIdx = (m.modalBtnIdx + delta) % n
	if m.modalBtnIdx < 0 {
		m.modalBtnIdx += n
	}
}

func modalCycleDelta(key string) (int, bool) {
	switch key {
	case "tab", "right":
		return 1, true
	case "backtab", "left":
		return -1, true
	default:
		return 0, false
	}
}

// --- drawing -------------------------------------------------------------

// modalGeometry is where a modal's parts land on screen. Drawing and mouse
// hit-testing both derive from it, so a click can never disagree with what was
// painted.
type modalGeometry struct {
	outer   rect
	innerX  int
	innerY  int
	innerW  int
	innerH  int
	bodyH   int
	btnY    int
	buttons []rect
	rows    map[int]rect
}

func (spec modalSpec) geometry(screenW, screenH int, bordered bool) modalGeometry {
	w := clamp(spec.width, 4, max(4, screenW))
	h := clamp(spec.height, 3, max(3, screenH))
	x := max(0, (screenW-w)/2)
	y := max(0, (screenH-h)/2)

	g := modalGeometry{outer: rect{x: x, y: y, w: w, h: h}, rows: map[int]rect{}}
	if bordered {
		g.innerX, g.innerY = x+2, y+1
		g.innerW, g.innerH = max(1, w-4), max(1, h-2)
	} else {
		g.innerX, g.innerY = x+1, y+1
		g.innerW, g.innerH = max(1, w-2), max(1, h-1)
	}

	// The button row sits on the last inner line, with a blank line above it.
	g.btnY = g.innerY + g.innerH - 1
	g.bodyH = max(0, g.innerH-2)
	g.buttons = buttonRects(spec.buttons, g.innerX, g.btnY, g.innerW)

	for i := 0; i < g.bodyH && i < len(spec.lines); i++ {
		if row := spec.lines[i].row; row >= 0 {
			g.rows[row] = rect{x: g.innerX, y: g.innerY + i, w: g.innerW, h: 1}
		}
	}
	return g
}

// draw paints the modal centred on the screen.
func (spec modalSpec) draw(sc tcell.Screen, screenW, screenH, focusedBtn int, bordered bool) {
	g := spec.geometry(screenW, screenH, bordered)
	fillRect(sc, g.outer.x, g.outer.y, g.outer.w, g.outer.h, baseStyle)

	if bordered {
		drawBox(sc, g.outer.x, g.outer.y, g.outer.w, g.outer.h, spec.title, activeBorderStyle, activeTitleStyle)
	} else {
		drawCell(sc, g.outer.x, g.outer.y, g.outer.w, spec.title, activeTitleStyle)
	}

	if spec.scroll != nil {
		spec.scroll.Width = g.innerW
		spec.scroll.Height = g.bodyH
		for i, line := range spec.scroll.visible() {
			drawCell(sc, g.innerX, g.innerY+i, g.innerW, line, baseStyle)
		}
	} else {
		for i := 0; i < g.bodyH && i < len(spec.lines); i++ {
			line := spec.lines[i]
			ly := g.innerY + i
			switch line.kind {
			case modalLineSelected:
				drawCell(sc, g.innerX, ly, g.innerW, padRight(line.text, g.innerW), selectedStyle)
			case modalLineDim:
				drawCell(sc, g.innerX, ly, g.innerW, line.text, dimStyle)
			default:
				drawCell(sc, g.innerX, ly, g.innerW, line.text, baseStyle)
			}
		}
	}

	for i, b := range spec.buttons {
		if i >= len(g.buttons) {
			break
		}
		style := buttonStyle
		if i == focusedBtn {
			style = buttonFocusStyle
		}
		r := g.buttons[i]
		drawCell(sc, r.x, r.y, r.w, padCenter(b.text(), r.w), style)
	}
}

// drawBox paints a rounded border with a title in the top edge.
func drawBox(sc tcell.Screen, x, y, w, h int, title string, border, titleSt tcell.Style) {
	if w < 2 || h < 2 {
		return
	}
	for i := 1; i < w-1; i++ {
		sc.SetContent(x+i, y, '─', nil, border)
		sc.SetContent(x+i, y+h-1, '─', nil, border)
	}
	for i := 1; i < h-1; i++ {
		sc.SetContent(x, y+i, '│', nil, border)
		sc.SetContent(x+w-1, y+i, '│', nil, border)
	}
	sc.SetContent(x, y, '╭', nil, border)
	sc.SetContent(x+w-1, y, '╮', nil, border)
	sc.SetContent(x, y+h-1, '╰', nil, border)
	sc.SetContent(x+w-1, y+h-1, '╯', nil, border)

	if title = strings.TrimSpace(title); title != "" {
		inner := max(0, w-2)
		label := " " + truncate(title, max(1, inner-2)) + " "
		drawText(sc, x+1, y, inner, label, titleSt)
	}
}
