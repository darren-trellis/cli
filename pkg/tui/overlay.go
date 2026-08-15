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
	mh := lipgloss.Height(modal)
	x := max(0, (m.width-mw)/2)
	y := max(0, (m.height-mh)/2)
	return placeOverlay(x, y, modal, base)
}

func (m Model) renderModalButtons(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	idx := m.modalBtnIdx
	if idx < 0 {
		idx = 0
	}
	if idx >= len(labels) {
		idx = len(labels) - 1
	}

	parts := make([]string, len(labels))
	for i, label := range labels {
		text := " " + label + " "
		if i == idx {
			parts[i] = buttonFocusStyle.Render(text)
		} else {
			parts[i] = buttonStyle.Render(text)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
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
