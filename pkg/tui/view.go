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
	editor   rect
	filter   rect
	status   rect
}

func (m Model) computeLayout() layoutRegions {
	w := max(40, m.width)
	h := max(12, m.height)

	leftW := max(18, w/5)
	rightW := w - leftW
	statusH := 1
	filterH := 3
	editorH := max(8, h/3)
	secretsH := h - editorH - statusH - filterH
	if secretsH < 4 {
		secretsH = 4
		editorH = max(6, h-secretsH-statusH-filterH)
	}
	topH := secretsH + editorH

	return layoutRegions{
		projects: rect{0, 0, leftW, topH},
		secrets:  rect{leftW, 0, rightW, secretsH},
		editor:   rect{leftW, secretsH, rightW, editorH},
		status:   rect{0, topH, w, statusH},
		filter:   rect{0, topH + statusH, w, filterH},
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	layout := m.computeLayout()

	left := m.renderProjectTree(layout.projects.w, layout.projects.h)

	secretsTitle := fmt.Sprintf("Secrets (%d)", len(m.filteredIndexes()))
	if m.activeProject != "" && m.activeConfig != "" {
		secretsTitle = fmt.Sprintf("Secrets (%d) [%s / %s]", len(m.filteredIndexes()), m.activeProject, m.activeConfig)
	}

	idxs := m.filteredIndexes()
	secretLines := make([]string, len(idxs))
	for i, idx := range idxs {
		s := m.secrets[idx]
		marker := "  "
		if s.shouldDelete {
			marker = "D "
		} else if s.isDirty() {
			marker = "* "
		}
		line := fmt.Sprintf("%s%s  %s", marker, s.name, dimStyle.Render(s.previewValue()))
		if s.shouldDelete {
			line = deleteStyle.Render(fmt.Sprintf("%s%s  %s", marker, s.name, s.previewValue()))
		} else if s.isDirty() {
			line = dirtyStyle.Render(fmt.Sprintf("%s%s  %s", marker, s.name, s.previewValue()))
		}
		secretLines[i] = line
	}

	secretsPanel := m.renderLinesPanel(secretsTitle, secretLines, m.secretIdx, m.focus == focusSecrets, layout.secrets.w, layout.secrets.h)
	editorPanel := m.renderEditor(layout.editor.w, layout.editor.h)
	right := lipgloss.JoinVertical(lipgloss.Left, secretsPanel, editorPanel)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	status := m.renderStatus(layout.status.w)
	filter := m.renderFilter(layout.filter.w, layout.filter.h)

	base := lipgloss.JoinVertical(lipgloss.Left, main, status, filter)

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
	return m.renderLinesPanel(title, lines, m.treeIdx, m.focus == focusProjects, width, height)
}

func (m Model) renderLinesPanel(title string, lines []string, selected int, active bool, width, height int) string {
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
		}
		body = append(body, line)
	}
	for len(body) < innerH {
		body = append(body, "")
	}

	return renderTitledListPanel(title, strings.Join(body[:innerH], "\n"), width, height, active)
}

func (m Model) renderEditor(width, height int) string {
	active := m.inEditor()

	innerW := max(10, width-4)
	m.nameInput.Width = max(8, innerW-7)
	m.valueInput.SetWidth(innerW)
	valueH := max(3, height-5)
	m.valueInput.SetHeight(valueH)

	nameLine := m.nameInput.View()
	if m.focus == focusEditorName {
		nameLine = selectedStyle.Render("▸ ") + m.nameInput.View()
	} else {
		nameLine = "  " + m.nameInput.View()
	}

	valueLabel := "Value:"
	if m.focus == focusEditorValue {
		valueLabel = selectedStyle.Render("▸ Value:")
	} else {
		valueLabel = "  Value:"
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		nameLine,
		valueLabel,
		m.valueInput.View(),
		helpStyle.Render("tab next pane · esc back"),
	)
	return renderTitledPanel("Editor", content, width, height, active)
}

func (m Model) renderStatus(width int) string {
	var text string
	if m.fetching {
		text = m.spinner.View() + " Loading…"
	} else if m.errMsg != "" {
		text = errorStyle.Render(m.errMsg)
	} else if m.statusMsg != "" {
		text = statusStyle.Render(m.statusMsg)
	} else {
		text = helpStyle.Render("tab cycle · / filter · ? help · q quit")
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func (m Model) renderFilter(width, height int) string {
	active := m.focus == focusFilter
	content := m.filterInput.View()
	if !active && m.filter != "" {
		content = "/ " + m.filter
	} else if !active {
		content = dimStyle.Render("/ filter")
	}
	return renderTitledPanel("Filter", content, width, height, active)
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

Secrets use a list + editor: select a secret,
press Enter/e to edit name and value.

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
