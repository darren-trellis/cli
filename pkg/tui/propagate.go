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
)

type propagateTarget struct {
	name   string
	locked bool
	dirty  bool
	on     bool
}

func (t propagateTarget) selectable() bool {
	return !t.dirty
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

func (m Model) openPropagate() (Model, Cmd) {
	m.propagateTargets = m.siblingPropagateTargets()
	m.propagateIdx = 0
	m.modalBtnIdx = 0
	m.setFocus(focusPropagate)
	return m, nil
}

func (m Model) requestSaveConfirm() (Model, Cmd) {
	if m.shouldAskPropagate() {
		return m.openPropagate()
	}
	return m.confirmSave()
}

func (m Model) handlePropagateKey(msg keyMsg) (Model, Cmd) {
	buttons := m.propagateModalButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}

	switch msg.Chord {
	case "space":
		m.togglePropagateAt(m.propagateIdx)
	case "a", "A":
		m.toggleAllPropagate()
	case "enter":
		return m.activateFocusedModalButton()
	case "y":
		return m.confirmSave()
	case "c", "esc", "q":
		return m.cancelPropagate()
	case "j", "down":
		m.movePropagateIdx(1)
	case "k", "up":
		m.movePropagateIdx(-1)
	case "C-c":
		return m, Quit
	}
	return m, nil
}

func (m Model) cancelPropagate() (Model, Cmd) {
	m.clearPropagate()
	m.pendingChanges = nil
	m.setFocus(focusSecrets)
	return m, nil
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
