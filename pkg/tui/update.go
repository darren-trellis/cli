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
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.valueInput.SetWidth(max(20, m.editorInnerWidth()))
		m.nameInput.Width = max(10, m.editorInnerWidth()-7)
		m.filterInput.Width = max(10, m.width-6)
		m.helpViewport.Width = min(60, m.width-8)
		m.helpViewport.Height = min(24, m.height-8)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.fetching {
			return m, cmd
		}
		return m, nil

	case errMsg:
		m.fetching = false
		m.errMsg = msg.Error()
		m.statusMsg = ""
		return m, nil

	case loadedMsg:
		m.fetching = false
		m.errMsg = ""
		m.projects = msg.projects
		m.projectConfigs[msg.activeProject] = msg.configs
		m.expanded[msg.activeProject] = true
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.activeProject = msg.activeProject
		m.activeConfig = msg.activeConfig
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.loadEditorFromSelection()
		return m, nil

	case projectSelectedMsg:
		m.fetching = false
		m.errMsg = ""
		m.projectConfigs[msg.project] = msg.configs
		m.expanded[msg.project] = true
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.activeProject = msg.project
		m.activeConfig = msg.config
		m.pendingChanges = nil
		m.rebuildTree()
		if msg.config != "" {
			m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.project, msg.config)
			m.setFocus(focusSecrets)
		} else {
			m.treeIdx = findTreeIndex(m.tree, treeProject, msg.project, "")
			m.setFocus(focusProjects)
		}
		m.loadEditorFromSelection()
		return m, nil

	case configsLoadedMsg:
		m.fetching = false
		m.errMsg = ""
		m.projectConfigs[msg.project] = msg.configs
		if msg.expand {
			m.expanded[msg.project] = true
		}
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeProject, msg.project, "")
		return m, nil

	case secretsLoadedMsg:
		m.fetching = false
		m.errMsg = ""
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.activeProject = msg.activeProject
		m.activeConfig = msg.activeConfig
		if _, ok := m.projectConfigs[msg.activeProject]; ok {
			m.expanded[msg.activeProject] = true
		}
		m.pendingChanges = nil
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.loadEditorFromSelection()
		m.setFocus(focusSecrets)
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.fetching {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	switch m.focus {
	case focusIntro:
		return m.handleIntroKey(msg)
	case focusHelp:
		return m.handleHelpKey(msg)
	case focusSave:
		return m.handleSaveKey(msg)
	case focusFilter:
		return m.handleFilterKey(msg)
	case focusEditorName, focusEditorValue:
		return m.handleEditorKey(msg)
	default:
		return m.handleNavKey(msg)
	}
}

func (m Model) handleIntroKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		configuration.TUIMarkIntroSeen()
		m.setFocus(focusSecrets)
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.setFocus(focusSecrets)
	case "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.helpViewport.LineDown(1)
	case "k", "up":
		m.helpViewport.LineUp(1)
	}
	return m, nil
}

func (m Model) handleSaveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if len(m.pendingChanges) == 0 {
			m.setFocus(focusSecrets)
			return m, nil
		}
		m.fetching = true
		m.statusMsg = ""
		m.errMsg = ""
		m.setFocus(focusSecrets)
		return m, tea.Batch(m.spinner.Tick, saveSecretsCmd(m.opts, m.activeProject, m.activeConfig, m.pendingChanges))
	case "esc", "q":
		m.setFocus(focusSecrets)
		m.pendingChanges = nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "enter", "esc":
		m.filter = m.filterInput.Value()
		m.secretIdx = 0
		m.loadEditorFromSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m.clampSecretIdx()
	m.loadEditorFromSelection()
	return m, cmd
}

func (m Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.applyEditorToSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "tab":
		m.applyEditorToSelection()
		m.cyclePane(1)
		return m, nil
	case "shift+tab":
		m.applyEditorToSelection()
		m.cyclePane(-1)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	if m.focus == focusEditorName {
		m.nameInput, cmd = m.nameInput.Update(msg)
		normalized := normalizeSecretName(m.nameInput.Value())
		if normalized != m.nameInput.Value() {
			m.nameInput.SetValue(normalized)
			m.nameInput.SetCursor(len(normalized))
		}
	} else {
		idx, ok := m.selectedSecretIndex()
		if ok && m.secrets[idx].originalVisibility == "restricted" && !m.secrets[idx].isTouched {
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
				m.secrets[idx].isTouched = true
				m.valueInput.SetValue("")
			}
		}
		m.valueInput, cmd = m.valueInput.Update(msg)
	}
	m.applyEditorToSelection()
	return m, cmd
}

func (m *Model) focusValueEditor() {
	m.focus = focusEditorValue
	m.nameInput.Blur()
	m.valueInput.Focus()
}

func (m Model) handleNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.cyclePane(1)
		return m, nil
	case "shift+tab":
		m.cyclePane(-1)
		return m, nil
	case "1":
		m.setFocus(focusProjects)
		return m, nil
	case "2":
		m.setFocus(focusSecrets)
		return m, nil
	case "/":
		m.setFocus(focusFilter)
		return m, nil
	case "?":
		m.focus = focusHelp
		m.helpViewport.SetContent(helpText)
		m.helpViewport.GotoTop()
		return m, nil
	case "j", "down":
		m.moveList(1)
		return m, nil
	case "k", "up":
		m.moveList(-1)
		return m, nil
	case " ":
		if m.focus == focusProjects {
			return m.toggleFold()
		}
		return m, nil
	case "enter":
		return m.activateSelection()
	case "e":
		if m.focus == focusSecrets || m.focus == focusProjects {
			m.setFocus(focusEditorName)
		}
		return m, nil
	case "a":
		return m.addSecret()
	case "d":
		return m.deleteSecret()
	case "u":
		return m.undoSecret()
	case "y":
		return m.yankSecret()
	case "s":
		return m.openSave()
	}
	return m, nil
}

func (m *Model) moveList(delta int) {
	switch m.focus {
	case focusProjects:
		if len(m.tree) == 0 {
			return
		}
		m.treeIdx = clamp(m.treeIdx+delta, 0, len(m.tree)-1)
	case focusSecrets:
		idxs := m.filteredIndexes()
		if len(idxs) == 0 {
			return
		}
		m.secretIdx = clamp(m.secretIdx+delta, 0, len(idxs)-1)
		m.loadEditorFromSelection()
	}
}

func (m Model) toggleFold() (tea.Model, tea.Cmd) {
	row, ok := m.currentTreeRow()
	if !ok {
		return m, nil
	}

	switch {
	case row.kind == treeProject:
		return m.toggleProjectFold(row.project)
	case row.kind == treeConfig && row.hasChildren:
		return m.toggleEnvFold(row.project, row.foldRoot)
	default:
		return m, nil
	}
}

func (m Model) toggleEnvFold(project, rootConfig string) (tea.Model, tea.Cmd) {
	if m.expandedEnvs == nil {
		m.expandedEnvs = map[string]bool{}
	}
	key := envKey(project, rootConfig)
	m.expandedEnvs[key] = !isEnvExpanded(m.expandedEnvs, project, rootConfig)
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, project, rootConfig)
	return m, nil
}

func (m Model) toggleProjectFold(project string) (tea.Model, tea.Cmd) {
	if project == "" {
		return m, nil
	}

	if m.expanded[project] {
		m.expanded[project] = false
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeProject, project, "")
		return m, nil
	}

	if _, cached := m.projectConfigs[project]; cached {
		m.expanded[project] = true
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeProject, project, "")
		return m, nil
	}

	m.fetching = true
	m.errMsg = ""
	m.treeIdx = findTreeIndex(m.tree, treeProject, project, "")
	return m, tea.Batch(m.spinner.Tick, fetchProjectConfigsCmd(m.opts, project))
}

func (m Model) activateSelection() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusProjects:
		row, ok := m.currentTreeRow()
		if !ok {
			return m, nil
		}
		if row.kind == treeConfig {
			if row.project == "" || row.config == "" {
				return m, nil
			}
			m.fetching = true
			m.errMsg = ""
			return m, tea.Batch(m.spinner.Tick, selectConfigCmd(m.opts, row.project, row.config))
		}
		if row.project == "" {
			return m, nil
		}
		m.fetching = true
		m.errMsg = ""
		return m, tea.Batch(m.spinner.Tick, selectProjectCmd(m.opts, row.project, m.activeConfig))
	case focusSecrets:
		m.enterEditor(false)
		return m, nil
	}
	return m, nil
}

func (m *Model) enterEditor(focusValue bool) {
	if focusValue {
		m.setFocus(focusEditorValue)
		return
	}
	m.setFocus(focusEditorName)
}

func (m Model) addSecret() (tea.Model, tea.Cmd) {
	m.applyEditorToSelection()
	m.secrets = append(m.secrets, newEmptySecretRow())
	m.filter = ""
	m.filterInput.SetValue("")
	idxs := m.filteredIndexes()
	m.secretIdx = len(idxs) - 1
	m.loadEditorFromSelection()
	m.setFocus(focusEditorName)
	return m, nil
}

func (m Model) deleteSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	if m.secrets[idx].originalName == nil {
		m.secrets = append(m.secrets[:idx], m.secrets[idx+1:]...)
		m.clampSecretIdx()
		m.loadEditorFromSelection()
		return m, nil
	}
	m.secrets[idx].shouldDelete = true
	return m, nil
}

func (m Model) undoSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	m.secrets[idx].undo()
	m.loadEditorFromSelection()
	return m, nil
}

func (m Model) yankSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	text := m.secrets[idx].displayValue()
	if m.focus == focusEditorName {
		text = m.secrets[idx].name
	}
	_ = utils.CopyToClipboard(text)
	m.statusMsg = "Copied to clipboard"
	return m, nil
}

func (m Model) openSave() (tea.Model, tea.Cmd) {
	m.applyEditorToSelection()
	m.pendingChanges = collectChanges(m.secrets)
	m.focus = focusSave
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.fetching || m.inModal() {
		return m, nil
	}

	layout := m.computeLayout()

	if msg.Button == tea.MouseButtonWheelUp {
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(-1)
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(1)
		return m, nil
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	switch {
	case layout.projects.contains(msg.X, msg.Y):
		m.setFocus(focusProjects)
		rel := msg.Y - layout.projects.y - 1
		if rel >= 0 {
			visible := max(1, layout.projects.h-2)
			start := 0
			if m.treeIdx >= visible {
				start = m.treeIdx - visible + 1
			}
			idx := start + rel
			if idx >= 0 && idx < len(m.tree) {
				m.treeIdx = idx
				return m.activateSelection()
			}
		}
	case layout.secrets.contains(msg.X, msg.Y):
		m.setFocus(focusSecrets)
		rel := msg.Y - layout.secrets.y - 1
		idxs := m.filteredIndexes()
		visible := max(1, layout.secrets.h-2)
		start := 0
		if m.secretIdx >= visible {
			start = m.secretIdx - visible + 1
		}
		idx := start + rel
		if idx >= 0 && idx < len(idxs) {
			m.secretIdx = idx
			m.loadEditorFromSelection()
		}
	case layout.editor.contains(msg.X, msg.Y):
		m.enterEditor(msg.Y > layout.editor.y+2)
	case layout.status.contains(msg.X, msg.Y):
		m.setFocus(focusFilter)
	}
	return m, nil
}

func (m *Model) focusPanelAt(x, y int, layout layoutRegions) {
	switch {
	case layout.projects.contains(x, y):
		m.focus = focusProjects
	case layout.secrets.contains(x, y):
		m.focus = focusSecrets
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) editorInnerWidth() int {
	left := max(20, m.width/5)
	return max(20, m.width-left-8)
}

const helpText = `Global Keybinds:
    Tab / Shift+Tab  Cycle panes
    1 Focus Projects
    2 Focus Secrets
    / Filter secrets (status bar)
    ? Help
    q Exit

Themes:
    doppler tui --theme <name>
    Built-ins: default, cool, warm, mono
    Plus teleminator themes (catppuccin, nord,
    tokyo-night, gruvbox, dracula, ...)
    Choice is saved in your Doppler config

Projects (with configs):
    j / k   Move
    Space   Fold / unfold project or env
    Enter   Select project or config
    Folded nodes still show the active
    config when it belongs under them

Secrets List:
    j / k   Move
    Enter/e Edit selected secret
    a       Add secret
    d       Delete / mark delete
    u       Undo changes
    y       Copy value to clipboard
    s       Save prompt

Filter:
    /       Edit filter in status bar
    Enter / Esc / Tab  Apply and return
    Active filter shows on the right
    of the status bar

Editor:
    Tab     Next pane (name → value → …)
    Esc     Return to secrets list

Save Prompt:
    Enter   Confirm
    Esc / q Cancel

Mouse:
    Click lists to focus/select
    Click editor to edit
    Click status bar to filter
    Scroll wheel to navigate`
