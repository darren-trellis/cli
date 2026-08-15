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
	"regexp"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type focusArea int

const (
	focusProjects focusArea = iota
	focusSecrets
	focusSecretInsert
	focusFilter
	focusSearch
	focusCreateConfig
	focusCommand
	focusHelp
	focusSave
	focusSwitchConfirm
)

type secretCol int

const (
	colName secretCol = iota
	colValue
)

type configPromptKind int

const (
	configPromptCreate configPromptKind = iota
	configPromptRename
)

type Model struct {
	opts models.ScopedOptions
	cfg  configuration.TUISettings

	width  int
	height int

	focus focusArea

	projects       []string
	projectConfigs map[string][]configRow
	expanded       map[string]bool
	expandedEnvs   map[string]bool
	tree           []treeRow
	treeIdx        int
	treeOffset     int

	secrets      []secretRow
	secretIdx    int // index into filteredIndexes()
	secretOffset int
	secretCol    secretCol
	undoStack    []int // secret store indexes, most recent last

	filter      string
	filterInput textinput.Model

	searchQuery    string
	searchInput    textinput.Model
	searchRe       *regexp.Regexp
	searchPane     focusArea
	searchMatches  []int
	searchMatchIdx int

	createConfigInput   textinput.Model
	createConfigProject string
	createConfigEnv     string
	configPromptMode    configPromptKind // create or rename
	renameFromConfig    string

	commandInput textinput.Model
	completions  CompletionState
	keys         KeysConfig
	pendingYank  bool

	cellInput textinput.Model

	fetching  bool
	statusMsg string
	errMsg    string
	spinner   spinner.Model

	activeProject string
	activeConfig  string

	pendingChanges []models.ChangeRequest

	pendingSwitchProject string
	pendingSwitchConfig  string

	helpViewport  viewport.Model
	configModTime time.Time
}

func newModel(opts models.ScopedOptions, cfg configuration.TUISettings) Model {
	cfg = configuration.NormalizeTUISettings(cfg)

	fi := textinput.New()
	fi.Placeholder = "Filter secrets…"
	fi.CharLimit = 128
	fi.Prompt = "f "

	si := textinput.New()
	si.Placeholder = "Search regex…"
	si.CharLimit = 256
	si.Prompt = "/ "

	cci := textinput.New()
	cci.Placeholder = "New config (e.g. dev_personal)…"
	cci.CharLimit = 128
	cci.Prompt = "+ "

	cmi := textinput.New()
	cmi.Placeholder = "command…"
	cmi.CharLimit = 256
	cmi.Prompt = ": "

	ci := textinput.New()
	ci.CharLimit = 4096
	ci.Prompt = ""
	ci.Placeholder = ""
	_ = ci.Cursor.SetMode(cursor.CursorStatic)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := Model{
		opts:              opts,
		cfg:               cfg,
		focus:             focusSecrets,
		filterInput:       fi,
		searchInput:       si,
		searchPane:        focusSecrets,
		createConfigInput: cci,
		commandInput:      cmi,
		keys:              MergeKeys(cfg.Keys),
		cellInput:         ci,
		secretCol:         colName,
		spinner:           sp,
		fetching:          true,
		projectConfigs:    map[string][]configRow{},
		expanded:          map[string]bool{},
		expandedEnvs:      map[string]bool{},
	}

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, loadCmd(m.opts)}
	if m.cfg.Autoreload {
		cmds = append(cmds, watchConfigCmd())
	}
	return tea.Batch(cmds...)
}

func (m Model) filteredIndexes() []int {
	return filterSecretIndexes(m.secrets, m.filter, m.cfg.CaseMode)
}

func (m Model) panelChrome() int {
	if m.cfg.Border {
		return 2
	}
	return 1 // title row without pane border
}

func (m Model) projectsVisibleRows() int {
	if m.width == 0 || m.height == 0 || !m.cfg.Sidebar {
		return 1
	}
	return max(1, m.computeLayout().projects.h-m.panelChrome())
}

func (m Model) secretsVisibleRows() int {
	if m.width == 0 || m.height == 0 {
		return 1
	}
	return max(1, m.computeLayout().secrets.h-m.panelChrome()-1)
}

func clampScrollOffset(offset, selected, visible, total int) int {
	if visible <= 0 || total <= visible {
		return 0
	}
	maxOffset := total - visible
	if selected < offset {
		offset = selected
	} else if selected >= offset+visible {
		offset = selected - visible + 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	return offset
}

func (m *Model) syncScrollOffsets() {
	m.treeOffset = clampScrollOffset(m.treeOffset, m.treeIdx, m.projectsVisibleRows(), len(m.tree))
	m.secretOffset = clampScrollOffset(m.secretOffset, m.secretIdx, m.secretsVisibleRows(), len(m.filteredIndexes()))
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

func flattenForCell(s string) string {
	return strings.ReplaceAll(s, "\n", "⏎")
}

func unflattenFromCell(s string) string {
	return strings.ReplaceAll(s, "⏎", "\n")
}

func (m *Model) loadCellFromSelection() {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		m.cellInput.SetValue("")
		return
	}
	s := m.secrets[idx]
	if m.secretCol == colName {
		m.cellInput.SetValue(s.name)
	} else {
		m.cellInput.SetValue(flattenForCell(s.displayValue()))
	}
	m.cellInput.CursorEnd()
}

func (m *Model) applyCellToSelection() {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return
	}
	s := &m.secrets[idx]
	if m.secretCol == colName {
		s.name = normalizeSecretName(m.cellInput.Value())
		m.noteSecretEdit(idx)
		return
	}

	val := unflattenFromCell(m.cellInput.Value())
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
	m.noteSecretEdit(idx)
}

func (m *Model) noteSecretEdit(idx int) {
	if idx < 0 || idx >= len(m.secrets) || !m.secrets[idx].isDirty() {
		return
	}
	for i, v := range m.undoStack {
		if v == idx {
			m.undoStack = append(m.undoStack[:i], m.undoStack[i+1:]...)
			break
		}
	}
	m.undoStack = append(m.undoStack, idx)
}

func (m *Model) adjustUndoStackForRemoval(removed int) {
	next := m.undoStack[:0]
	for _, i := range m.undoStack {
		if i == removed {
			continue
		}
		if i > removed {
			next = append(next, i-1)
		} else {
			next = append(next, i)
		}
	}
	m.undoStack = next
}

func (m *Model) selectSecretByStoreIndex(storeIdx int) {
	idxs := m.filteredIndexes()
	for i, idx := range idxs {
		if idx == storeIdx {
			m.secretIdx = i
			return
		}
	}
	m.clampSecretIdx()
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
	if m.searchPane == focusProjects && m.searchRe != nil {
		m.refreshSearchMatches()
	}
}

func (m Model) inModal() bool {
	return m.focus == focusHelp || m.focus == focusSave || m.focus == focusSwitchConfirm
}

func (m Model) hasDirtySecrets() bool {
	return len(collectChanges(m.secrets)) > 0
}

func (m Model) inSecretInsert() bool {
	return m.focus == focusSecretInsert
}

var paneOrder = []focusArea{
	focusProjects,
	focusSecrets,
}

func (m *Model) setFocus(f focusArea) {
	m.cellInput.Blur()
	m.filterInput.Blur()
	m.searchInput.Blur()
	m.createConfigInput.Blur()
	m.commandInput.Blur()

	m.focus = f
	switch f {
	case focusSecretInsert:
		if _, ok := m.selectedSecretIndex(); !ok {
			m.focus = focusSecrets
			return
		}
		m.loadCellFromSelection()
		m.cellInput.Focus()
	case focusFilter:
		m.filterInput.SetValue(m.filter)
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
	case focusSearch:
		m.searchInput.SetValue(m.searchQuery)
		m.searchInput.CursorEnd()
		m.searchInput.Focus()
	case focusCreateConfig:
		m.createConfigInput.Focus()
		m.createConfigInput.CursorEnd()
	case focusCommand:
		m.commandInput.Focus()
		m.commandInput.CursorEnd()
	}
}

func (m *Model) enterInsert() {
	if _, ok := m.selectedSecretIndex(); !ok {
		return
	}
	m.setFocus(focusSecretInsert)
}

func (m *Model) cyclePane(delta int) {
	if m.inModal() || m.inSecretInsert() {
		return
	}

	order := paneOrder
	if !m.cfg.Sidebar {
		order = []focusArea{focusSecrets}
	}

	cur := -1
	for i, p := range order {
		if p == m.focus {
			cur = i
			break
		}
	}
	if cur == -1 {
		m.setFocus(focusSecrets)
		return
	}

	n := len(order)
	cur = (cur + delta%n + n) % n
	m.setFocus(order[cur])
}

func (m *Model) toggleSidebar() {
	m.cfg.Sidebar = !m.cfg.Sidebar
	if !m.cfg.Sidebar && m.focus == focusProjects {
		m.focus = focusSecrets
	}
	if m.cfg.Autosave {
		configuration.TUISaveSettings(m.cfg)
	}
}
