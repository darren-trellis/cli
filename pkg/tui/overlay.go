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

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// placeOverlay draws fg on top of bg at (x, y), preserving ANSI in both.
func placeOverlay(x, y int, fg, bg string) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	for len(bgLines) < y+len(fgLines) {
		bgLines = append(bgLines, "")
	}

	for i, fgLine := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bgLine := bgLines[row]
		bgWidth := ansi.StringWidth(bgLine)
		if bgWidth < x {
			bgLine += strings.Repeat(" ", x-bgWidth)
		}

		fgWidth := ansi.StringWidth(fgLine)
		left := ansi.Truncate(bgLine, x, "")
		right := ""
		rightStart := x + fgWidth
		if rightStart < ansi.StringWidth(bgLine) {
			right = ansi.TruncateLeft(bgLine, rightStart, "")
		}
		bgLines[row] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}

func (m Model) renderOverlay(base, modal string) string {
	base = lipgloss.NewStyle().
		Width(max(1, m.width)).
		Height(max(1, m.height)).
		MaxHeight(max(1, m.height)).
		Render(base)

	mw := lipgloss.Width(modal)
	modal = fillLineBackground(modal, mw)
	mh := lipgloss.Height(modal)
	x := max(0, (m.width-mw)/2)
	y := max(0, (m.height-mh)/2)
	return placeOverlay(x, y, modal, base)
}

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

func (m Model) renderModalButtons(buttons []modalButton, width int) string {
	if len(buttons) == 0 {
		return ""
	}
	idx := m.modalBtnIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(buttons) {
		idx = len(buttons) - 1
	}

	labels := make([]string, len(buttons))
	for i, b := range buttons {
		labels[i] = b.text()
	}
	widths, gap := modalButtonLayout(labelWidths(labels), width)

	gapStyle := lipgloss.NewStyle()
	if background != "" {
		gapStyle = gapStyle.Background(background)
	}
	parts := make([]string, 0, len(buttons)*2-1)
	for i, label := range labels {
		if i > 0 && gap > 0 {
			parts = append(parts, gapStyle.Render(strings.Repeat(" ", gap)))
		}
		parts = append(parts, renderModalButton(label, widths[i], i == idx))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	if width <= 0 {
		return row
	}
	pad := max(0, width-lipgloss.Width(row))
	if pad == 0 {
		return row
	}
	left := pad / 2
	return gapStyle.Render(strings.Repeat(" ", left)) + row + gapStyle.Render(strings.Repeat(" ", pad-left))
}

func renderModalButton(label string, width int, focused bool) string {
	text := padButtonLabel(label, width)
	st := buttonStyle
	if focused {
		st = buttonFocusStyle
	}
	return st.Padding(0, 0).Width(width).MaxWidth(width).Inline(true).Render(text)
}

func labelWidths(labels []string) []int {
	out := make([]int, len(labels))
	for i, label := range labels {
		out[i] = max(1, lipgloss.Width(label))
	}
	return out
}

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

func padButtonLabel(label string, width int) string {
	w := lipgloss.Width(label)
	if width <= 0 {
		return label
	}
	if w >= width {
		return truncate(label, width)
	}
	pad := width - w
	left := pad / 2
	return strings.Repeat(" ", left) + label + strings.Repeat(" ", pad-left)
}

func inferButtonRowWidth(plain string) int {
	w := ansi.StringWidth(plain)
	if w == 0 {
		return 0
	}
	runes := []rune(plain)
	if runes[0] != '│' && runes[0] != '|' {
		return w
	}
	inner := w - 2
	if len(runes) > 2 && runes[1] == ' ' {
		inner--
	}
	if len(runes) > 2 && runes[len(runes)-2] == ' ' {
		inner--
	}
	return max(1, inner)
}

func fillLineBackground(s string, width int) string {
	if width <= 0 {
		return s
	}
	padStyle := lipgloss.NewStyle()
	if background != "" {
		padStyle = padStyle.Background(background)
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := ansi.StringWidth(line)
		if w < width {
			lines[i] = line + padStyle.Render(strings.Repeat(" ", width-w))
		}
	}
	return strings.Join(lines, "\n")
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
	case "shift+tab", "left":
		return -1, true
	default:
		return 0, false
	}
}

func buttonHitRects(modal string, buttons []modalButton, originX, originY int) []rect {
	if len(buttons) == 0 || modal == "" {
		return nil
	}
	first := buttons[0].text()
	for i, line := range strings.Split(modal, "\n") {
		plain := ansi.Strip(line)
		col := strings.Index(plain, first)
		if col < 0 {
			continue
		}
		cursor := col + len(first)
		ok := true
		for _, b := range buttons[1:] {
			idx := strings.Index(plain[cursor:], b.text())
			if idx < 0 {
				ok = false
				break
			}
			cursor += idx + len(b.text())
		}
		if !ok {
			continue
		}
		labels := make([]string, len(buttons))
		for j, b := range buttons {
			labels[j] = b.text()
		}
		widths, gap := modalButtonLayout(labelWidths(labels), inferButtonRowWidth(plain))
		start := originX + col - (widths[0]-lipgloss.Width(first))/2
		rects := make([]rect, len(buttons))
		x := start
		for j, w := range widths {
			rects[j] = rect{x: x, y: originY + i, w: w, h: 1}
			x += w + gap
		}
		return rects
	}
	return nil
}

func propagateRowHitRects(modal string, targets []propagateTarget, originX, originY int) []rect {
	if len(targets) == 0 || modal == "" {
		return nil
	}
	rects := make([]rect, 0, len(targets))
	used := make([]bool, len(targets))
	for i, line := range strings.Split(modal, "\n") {
		plain := ansi.Strip(line)
		if !strings.Contains(plain, "[ ]") && !strings.Contains(plain, "[x]") {
			continue
		}
		for j, t := range targets {
			if used[j] || !strings.Contains(plain, t.name) {
				continue
			}
			used[j] = true
			col := strings.Index(plain, "[")
			if col < 0 {
				col = strings.Index(plain, t.name)
			}
			if col < 0 {
				col = 0
			}
			rects = append(rects, rect{x: originX + col, y: originY + i, w: max(1, ansi.StringWidth(plain)-col), h: 1})
			break
		}
	}
	if len(rects) != len(targets) {
		return nil
	}
	return rects
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

func (m Model) currentModalView() string {
	switch m.focus {
	case focusHelp:
		return m.renderHelpModal()
	case focusSave:
		return m.renderSaveModal()
	case focusSwitchConfirm:
		return m.renderSwitchConfirmModal()
	case focusDeleteConfirm:
		return m.renderDeleteConfirmModal()
	case focusPropagate:
		return m.renderPropagateModal()
	default:
		return ""
	}
}
