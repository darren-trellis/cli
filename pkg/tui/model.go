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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type focusArea int

const (
	focusProjects focusArea = iota
	focusSecrets
	focusEditorName
	focusEditorValue
	focusFilter
	focusIntro
	focusHelp
	focusSave
)

type Model struct {
	opts models.ScopedOptions

	width  int
	height int

	focus focusArea

	projects       []string
	projectConfigs map[string][]configRow
	expanded       map[string]bool
	expandedEnvs   map[string]bool
	tree           []treeRow
	treeIdx        int

	secrets   []secretRow
	secretIdx int // index into filteredIndexes()

	filter      string
	filterInput textinput.Model

	nameInput  textinput.Model
	valueInput textarea.Model

	fetching  bool
	statusMsg string
	errMsg    string
	spinner   spinner.Model

	activeProject string
	activeConfig  string

	pendingChanges []models.ChangeRequest

	helpViewport viewport.Model
}

func newModel(opts models.ScopedOptions) Model {
	fi := textinput.New()
	fi.Placeholder = "Filter secrets…"
	fi.CharLimit = 128
	fi.Prompt = "/ "

	ni := textinput.New()
	ni.Placeholder = "SECRET_NAME"
	ni.CharLimit = 256
	ni.Prompt = "Name: "

	vi := textarea.New()
	vi.Placeholder = "Secret value"
	vi.SetHeight(6)
	vi.ShowLineNumbers = false
	vi.Prompt = ""

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := Model{
		opts:           opts,
		focus:          focusSecrets,
		filterInput:    fi,
		nameInput:      ni,
		valueInput:     vi,
		spinner:        sp,
		fetching:       true,
		projectConfigs: map[string][]configRow{},
		expanded:       map[string]bool{},
		expandedEnvs:   map[string]bool{},
	}

	if configuration.TUIShouldShowIntro() {
		m.focus = focusIntro
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadCmd(m.opts))
}

func (m Model) filteredIndexes() []int {
	return filterSecretIndexes(m.secrets, m.filter)
}

func (m Model) selectedSecretIndex() (int, bool) {
	idxs := m.filteredIndexes()
	if len(idxs) == 0 || m.secretIdx < 0 || m.secretIdx >= len(idxs) {
		return 0, false
	}
	return idxs[m.secretIdx], true
}

func (m *Model) clampSecretIdx() {
	idxs := m.filteredIndexes()
	if len(idxs) == 0 {
		m.secretIdx = 0
		return
	}
	if m.secretIdx >= len(idxs) {
		m.secretIdx = len(idxs) - 1
	}
	if m.secretIdx < 0 {
		m.secretIdx = 0
	}
}

func (m *Model) loadEditorFromSelection() {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		m.nameInput.SetValue("")
		m.valueInput.SetValue("")
		return
	}
	s := m.secrets[idx]
	m.nameInput.SetValue(s.name)
	m.valueInput.SetValue(s.displayValue())
}

func (m *Model) applyEditorToSelection() {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return
	}
	s := &m.secrets[idx]
	s.name = normalizeSecretName(m.nameInput.Value())
	val := m.valueInput.Value()
	if s.originalVisibility == "restricted" && !s.isTouched && val == "[RESTRICTED]" {
		return
	}
	if s.originalVisibility == "restricted" && (s.isTouched || val != "[RESTRICTED]") {
		s.isTouched = true
		if val == "[RESTRICTED]" {
			val = ""
		}
	}
	s.value = val
}

func (m Model) currentTreeRow() (treeRow, bool) {
	if len(m.tree) == 0 || m.treeIdx < 0 || m.treeIdx >= len(m.tree) {
		return treeRow{}, false
	}
	return m.tree[m.treeIdx], true
}

func (m *Model) rebuildTree() {
	var kind treeKind
	var project, config string
	if row, ok := m.currentTreeRow(); ok {
		kind = row.kind
		project = row.project
		config = row.config
	} else if m.activeProject != "" {
		kind = treeConfig
		project = m.activeProject
		config = m.activeConfig
	}

	if m.projectConfigs == nil {
		m.projectConfigs = map[string][]configRow{}
	}
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	if m.expandedEnvs == nil {
		m.expandedEnvs = map[string]bool{}
	}

	m.tree = buildProjectTree(m.projects, m.projectConfigs, m.expanded, m.expandedEnvs, m.activeProject, m.activeConfig)
	m.treeIdx = findTreeIndex(m.tree, kind, project, config)
	if m.treeIdx >= len(m.tree) {
		m.treeIdx = max(0, len(m.tree)-1)
	}
}

func (m Model) inModal() bool {
	return m.focus == focusIntro || m.focus == focusHelp || m.focus == focusSave
}

func (m Model) inEditor() bool {
	return m.focus == focusEditorName || m.focus == focusEditorValue
}

var paneOrder = []focusArea{
	focusProjects,
	focusSecrets,
	focusEditorName,
	focusEditorValue,
	focusFilter,
}

func (m *Model) setFocus(f focusArea) {
	m.nameInput.Blur()
	m.valueInput.Blur()
	m.filterInput.Blur()

	m.focus = f
	switch f {
	case focusEditorName:
		if _, ok := m.selectedSecretIndex(); !ok {
			m.focus = focusSecrets
			return
		}
		m.loadEditorFromSelection()
		m.nameInput.Focus()
	case focusEditorValue:
		if _, ok := m.selectedSecretIndex(); !ok {
			m.focus = focusSecrets
			return
		}
		m.loadEditorFromSelection()
		m.focusValueEditor()
	case focusFilter:
		m.filterInput.SetValue(m.filter)
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
	}
}

func (m *Model) cyclePane(delta int) {
	if m.inModal() {
		return
	}

	cur := -1
	for i, p := range paneOrder {
		if p == m.focus {
			cur = i
			break
		}
	}
	if cur == -1 {
		m.setFocus(focusSecrets)
		return
	}

	n := len(paneOrder)
	for i := 0; i < n; i++ {
		cur = (cur + delta%n + n) % n
		next := paneOrder[cur]
		if (next == focusEditorName || next == focusEditorValue) && len(m.filteredIndexes()) == 0 {
			continue
		}
		m.setFocus(next)
		return
	}
}
