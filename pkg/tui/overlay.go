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
	cellW := 0
	for i, b := range buttons {
		labels[i] = b.text()
		if w := lipgloss.Width(labels[i]); w > cellW {
			cellW = w
		}
	}

	gapStyle := lipgloss.NewStyle()
	if background != "" {
		gapStyle = gapStyle.Background(background)
	}
	parts := make([]string, 0, len(buttons)*2-1)
	for i, label := range labels {
		if i > 0 {
			parts = append(parts, gapStyle.Render(" "))
		}
		text := " " + padRight(label, cellW) + " "
		if i == idx {
			parts = append(parts, buttonFocusStyle.Render(text))
		} else {
			parts = append(parts, buttonStyle.Render(text))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
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

func modalButtonCellWidth(buttons []modalButton) int {
	cellW := 0
	for _, b := range buttons {
		if w := lipgloss.Width(b.text()); w > cellW {
			cellW = w
		}
	}
	return cellW
}

func buttonHitRects(modal string, buttons []modalButton, originX, originY int) []rect {
	if len(buttons) == 0 || modal == "" {
		return nil
	}
	cellW := modalButtonCellWidth(buttons)
	first := " " + padRight(buttons[0].text(), cellW) + " "
	for i, line := range strings.Split(modal, "\n") {
		plain := ansi.Strip(line)
		col := strings.Index(plain, first)
		if col < 0 {
			continue
		}
		btnW := cellW + 2
		rects := make([]rect, len(buttons))
		for j := range buttons {
			rects[j] = rect{x: originX + col + j*(btnW+1), y: originY + i, w: btnW, h: 1}
		}
		return rects
	}
	return nil
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
