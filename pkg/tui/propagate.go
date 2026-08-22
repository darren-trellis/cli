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
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type propagateTarget struct {
	name   string
	locked bool
	dirty  bool
	on     bool
}

func (t propagateTarget) selectable() bool {
	return !t.locked && !t.dirty
}

func (m Model) siblingRootConfigs() []configRow {
	var out []configRow
	for _, c := range m.projectConfigs[m.activeProject] {
		if c.root && c.name != m.activeConfig {
			out = append(out, c)
		}
	}
	return out
}

func (m Model) shouldAskPropagate() bool {
	if len(m.pendingChanges) == 0 {
		return false
	}
	if !m.configIsRoot(m.activeProject, m.activeConfig) {
		return false
	}
	return len(m.siblingRootConfigs()) > 0
}

func (m Model) siblingPropagateTargets() []propagateTarget {
	var out []propagateTarget
	for _, c := range m.siblingRootConfigs() {
		out = append(out, propagateTarget{
			name:   c.name,
			locked: c.locked,
			dirty:  m.configIsDirty(m.activeProject, c.name),
		})
	}
	return out
}

func (m *Model) clearPropagate() {
	m.propagateTargets = nil
	m.propagateIdx = 0
}

func (m Model) selectedPropagateConfigs() []string {
	var names []string
	for _, t := range m.propagateTargets {
		if t.on && t.selectable() {
			names = append(names, t.name)
		}
	}
	return names
}

func (m *Model) clampPropagateIdx() {
	if len(m.propagateTargets) == 0 {
		m.propagateIdx = 0
		return
	}
	if m.propagateIdx < 0 {
		m.propagateIdx = 0
	}
	if m.propagateIdx >= len(m.propagateTargets) {
		m.propagateIdx = len(m.propagateTargets) - 1
	}
}

func (m *Model) movePropagateIdx(delta int) {
	n := len(m.propagateTargets)
	if n == 0 {
		m.propagateIdx = 0
		return
	}
	m.propagateIdx = (m.propagateIdx + delta) % n
	if m.propagateIdx < 0 {
		m.propagateIdx += n
	}
}

func (m *Model) togglePropagateAt(idx int) {
	if idx < 0 || idx >= len(m.propagateTargets) {
		return
	}
	if !m.propagateTargets[idx].selectable() {
		return
	}
	m.propagateTargets[idx].on = !m.propagateTargets[idx].on
}

func (m *Model) toggleAllPropagate() {
	allOn := true
	any := false
	for _, t := range m.propagateTargets {
		if !t.selectable() {
			continue
		}
		any = true
		if !t.on {
			allOn = false
		}
	}
	if !any {
		return
	}
	on := !allOn
	for i := range m.propagateTargets {
		if m.propagateTargets[i].selectable() {
			m.propagateTargets[i].on = on
		}
	}
}

func (m Model) openPropagate() (tea.Model, tea.Cmd) {
	m.propagateTargets = m.siblingPropagateTargets()
	m.propagateIdx = 0
	m.modalBtnIdx = 0
	m.focus = focusPropagate
	return m, nil
}

func (m Model) requestSaveConfirm() (tea.Model, tea.Cmd) {
	if m.shouldAskPropagate() {
		return m.openPropagate()
	}
	return m.confirmSave()
}

func (m Model) handlePropagateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	buttons := m.propagateModalButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	if isSpaceKey(msg) {
		m.togglePropagateAt(m.propagateIdx)
		return m, nil
	}
	if isLetterKey(msg, 'a') {
		m.toggleAllPropagate()
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "y":
		return m.confirmSave()
	case "n":
		m.clearPropagate()
		return m.confirmSave()
	case "esc", "q":
		m.clearPropagate()
		m.modalBtnIdx = 0
		m.focus = focusSave
		return m, nil
	case "j", "down":
		m.movePropagateIdx(1)
	case "k", "up":
		m.movePropagateIdx(-1)
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func isSpaceKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeySpace {
		return true
	}
	switch msg.String() {
	case " ", "space":
		return true
	}
	if chord, ok := encodeKey(msg); ok && chord == "space" {
		return true
	}
	return len(msg.Runes) == 1 && msg.Runes[0] == ' '
}

func isLetterKey(msg tea.KeyMsg, letter rune) bool {
	letter = unicode.ToLower(letter)
	if len(msg.Runes) == 1 && unicode.ToLower(msg.Runes[0]) == letter {
		return true
	}
	s := msg.String()
	if rs := []rune(s); len(rs) == 1 && unicode.ToLower(rs[0]) == letter {
		return true
	}
	if chord, ok := encodeKey(msg); ok {
		if rs := []rune(chord); len(rs) == 1 && unicode.ToLower(rs[0]) == letter {
			return true
		}
	}
	return false
}

func formatSaveStatus(applied, failed []string) string {
	s := "Saved"
	if len(applied) > 0 {
		s += " · applied to " + strings.Join(applied, ", ")
	}
	if len(failed) > 0 {
		s += " · failed on " + strings.Join(failed, ", ")
	}
	return s
}
