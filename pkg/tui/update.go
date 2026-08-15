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
	"os"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type configWatchMsg struct{}

func watchConfigCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return configWatchMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	mod := next.(Model)
	mod.syncScrollOffsets()
	return mod, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filterInput.Width = max(10, m.width-2)
		m.searchInput.Width = max(10, m.width-2)
		m.createConfigInput.Width = max(10, m.width-2)
		m.commandInput.Width = max(10, m.width-2)
		m.cellInput.Width = max(10, m.width/3)
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

	case configWatchMsg:
		return m.handleConfigWatch()

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
		m.secretCol = colName
		m.undoStack = nil
		m.activeProject = msg.activeProject
		m.activeConfig = msg.activeConfig
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.persistSession()
		return m, nil

	case projectSelectedMsg:
		m.fetching = false
		m.errMsg = ""
		m.clearPendingSwitch()
		m.projectConfigs[msg.project] = msg.configs
		m.expanded[msg.project] = true
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.secretCol = colName
		m.undoStack = nil
		m.activeProject = msg.project
		m.activeConfig = msg.config
		m.createConfigProject = ""
		m.createConfigEnv = ""
		m.renameFromConfig = ""
		m.configPromptMode = configPromptCreate
		if msg.config != "" {
			for _, c := range msg.configs {
				if c.name == msg.config && c.environment != "" {
					for _, root := range msg.configs {
						if root.environment == c.environment && root.root {
							m.expandedEnvs[envKey(msg.project, root.name)] = true
							break
						}
					}
					break
				}
			}
		}
		m.rebuildTree()
		if msg.config != "" {
			m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.project, msg.config)
			m.setFocus(focusSecrets)
		} else {
			m.treeIdx = findTreeIndex(m.tree, treeProject, msg.project, "")
			m.setFocus(focusProjects)
		}
		m.persistSession()
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

	case configLockMsg:
		m.fetching = false
		m.errMsg = ""
		m.projectConfigs[msg.project] = msg.configs
		m.expanded[msg.project] = true
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.project, msg.config)
		if msg.locked {
			m.statusMsg = "Locked " + msg.config
		} else {
			m.statusMsg = "Unlocked " + msg.config
		}
		m.setFocus(focusProjects)
		return m, nil

	case secretsLoadedMsg:
		m.fetching = false
		m.errMsg = ""
		m.clearPendingSwitch()
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.secretCol = colName
		m.undoStack = nil
		m.activeProject = msg.activeProject
		m.activeConfig = msg.activeConfig
		if _, ok := m.projectConfigs[msg.activeProject]; ok {
			m.expanded[msg.activeProject] = true
		}
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.setFocus(focusSecrets)
		m.persistSession()
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
	case focusHelp:
		return m.handleHelpKey(msg)
	case focusSave:
		return m.handleSaveKey(msg)
	case focusSwitchConfirm:
		return m.handleSwitchConfirmKey(msg)
	case focusFilter:
		return m.handleFilterKey(msg)
	case focusSearch:
		return m.handleSearchKey(msg)
	case focusCreateConfig:
		return m.handleCreateConfigKey(msg)
	case focusCommand:
		return m.handleCommandKey(msg)
	case focusSecretInsert:
		return m.handleInsertKey(msg)
	default:
		return m.handleNavKey(msg)
	}
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	buttons := m.helpModalButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "esc", "q":
		m.setFocus(focusSecrets)
	case "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.helpViewport.LineDown(1)
	case "k", "up":
		m.helpViewport.LineUp(1)
	case "pgdown":
		m.helpViewport.ViewDown()
	case "pgup":
		m.helpViewport.ViewUp()
	}
	return m, nil
}

func (m Model) handleSaveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	buttons := m.saveModalButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "esc", "q":
		m.setFocus(focusSecrets)
		m.pendingChanges = nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleSwitchConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	buttons := m.switchConfirmButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "s":
		return m.confirmSwitchSave()
	case "d":
		return m.confirmSwitchDiscard()
	case "c", "esc", "q":
		m.clearPendingSwitch()
		m.setFocus(focusSecrets)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) activateFocusedModalButton() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusHelp:
		m.setFocus(focusSecrets)
		return m, nil
	case focusSave:
		if len(m.pendingChanges) == 0 || m.modalBtnIdx == 1 {
			m.setFocus(focusSecrets)
			m.pendingChanges = nil
			return m, nil
		}
		m.fetching = true
		m.statusMsg = ""
		m.errMsg = ""
		m.setFocus(focusSecrets)
		return m, tea.Batch(m.spinner.Tick, saveSecretsCmd(m.opts, m.activeProject, m.activeConfig, m.pendingChanges))
	case focusSwitchConfirm:
		switch m.modalBtnIdx {
		case 1:
			return m.confirmSwitchDiscard()
		case 2:
			m.clearPendingSwitch()
			m.setFocus(focusSecrets)
			return m, nil
		default:
			return m.confirmSwitchSave()
		}
	default:
		return m, nil
	}
}

func (m *Model) clearPendingSwitch() {
	m.pendingSwitchProject = ""
	m.pendingSwitchConfig = ""
	m.pendingChanges = nil
}

func (m Model) confirmSwitchSave() (tea.Model, tea.Cmd) {
	changes := m.pendingChanges
	if len(changes) == 0 {
		changes = collectChanges(m.secrets)
	}
	toProject := m.pendingSwitchProject
	toConfig := m.pendingSwitchConfig
	fromProject := m.activeProject
	fromConfig := m.activeConfig
	m.clearPendingSwitch()
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusSecrets)
	return m, tea.Batch(m.spinner.Tick, saveAndNavigateCmd(m.opts, fromProject, fromConfig, changes, toProject, toConfig))
}

func (m Model) confirmSwitchDiscard() (tea.Model, tea.Cmd) {
	toProject := m.pendingSwitchProject
	toConfig := m.pendingSwitchConfig
	m.clearPendingSwitch()
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusProjects)
	if toConfig != "" {
		return m, tea.Batch(m.spinner.Tick, selectConfigCmd(m.opts, toProject, toConfig))
	}
	return m, tea.Batch(m.spinner.Tick, selectProjectCmd(m.opts, toProject, m.activeConfig))
}

func (m Model) openSwitchConfirm(project, config string) (tea.Model, tea.Cmd) {
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.pendingSwitchProject = project
	m.pendingSwitchConfig = config
	m.pendingChanges = collectChanges(m.secrets)
	m.modalBtnIdx = 0
	m.focus = focusSwitchConfirm
	m.errMsg = ""
	m.statusMsg = ""
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "enter", "esc":
		m.filter = m.filterInput.Value()
		m.secretIdx = 0
		m.setFocus(focusSecrets)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m.clampSecretIdx()
	if m.searchPane == focusSecrets && m.searchRe != nil {
		m.refreshSearchMatches()
	}
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.clearSearch()
		m.setFocus(m.searchPane)
		return m, nil
	case "enter":
		query := m.searchInput.Value()
		if err := m.compileSearch(query); err != nil {
			m.errMsg = "Invalid regex"
			m.statusMsg = ""
			return m, nil
		}
		m.errMsg = ""
		m.setFocus(m.searchPane)
		return m, nil
	case "tab", "shift+tab":
		m.setFocus(m.searchPane)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	_ = m.applySearch(m.searchInput.Value(), false)
	return m, cmd
}

func (m Model) handleCreateConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.createConfigInput.SetValue("")
		m.createConfigProject = ""
		m.createConfigEnv = ""
		m.renameFromConfig = ""
		m.configPromptMode = configPromptCreate
		m.setFocus(focusProjects)
		return m, nil
	case "enter":
		if m.configPromptMode == configPromptRename {
			return m.submitRenameConfig()
		}
		return m.submitCreateConfig()
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.createConfigInput, cmd = m.createConfigInput.Update(msg)
	return m, cmd
}

func (m Model) submitCreateConfig() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.createConfigInput.Value())
	project := m.createConfigProject
	environment := m.createConfigEnv
	if environment == "" {
		environment = inferEnvironmentFromConfigName(name)
	}
	if project == "" {
		m.errMsg = "Select a project first"
		m.statusMsg = ""
		return m, nil
	}
	if name == "" {
		m.errMsg = "Config name required"
		m.statusMsg = ""
		return m, nil
	}
	if environment == "" {
		m.errMsg = "Need environment (select a config or use env_name)"
		m.statusMsg = ""
		return m, nil
	}

	m.errMsg = ""
	m.statusMsg = ""
	m.fetching = true
	m.createConfigInput.SetValue("")
	m.renameFromConfig = ""
	m.configPromptMode = configPromptCreate
	m.setFocus(focusProjects)
	return m, tea.Batch(m.spinner.Tick, createConfigCmd(m.opts, project, name, environment))
}

func (m Model) submitRenameConfig() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.createConfigInput.Value())
	project := m.createConfigProject
	from := m.renameFromConfig
	if project == "" || from == "" {
		m.errMsg = "Select a config to rename"
		m.statusMsg = ""
		return m, nil
	}
	if name == "" {
		m.errMsg = "Config name required"
		m.statusMsg = ""
		return m, nil
	}
	if name == from {
		m.createConfigInput.SetValue("")
		m.renameFromConfig = ""
		m.configPromptMode = configPromptCreate
		m.setFocus(focusProjects)
		return m, nil
	}

	m.errMsg = ""
	m.statusMsg = ""
	m.fetching = true
	m.createConfigInput.SetValue("")
	m.renameFromConfig = ""
	m.configPromptMode = configPromptCreate
	m.setFocus(focusProjects)
	return m, tea.Batch(m.spinner.Tick, renameConfigCmd(m.opts, project, from, name))
}

func inferEnvironmentFromConfigName(name string) string {
	idx := strings.Index(name, "_")
	if idx <= 0 {
		return ""
	}
	return name[:idx]
}

func (m Model) handleInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "enter":
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "tab":
		m.applyCellToSelection()
		if m.secretCol == colName {
			m.secretCol = colValue
		} else {
			m.secretCol = colName
		}
		m.loadCellFromSelection()
		m.cellInput.Focus()
		return m, nil
	case "shift+tab":
		m.applyCellToSelection()
		if m.secretCol == colValue {
			m.secretCol = colName
		} else {
			m.secretCol = colValue
		}
		m.loadCellFromSelection()
		m.cellInput.Focus()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	idx, ok := m.selectedSecretIndex()
	if ok && m.secretCol == colValue &&
		m.secrets[idx].originalVisibility == "restricted" && !m.secrets[idx].isTouched {
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
			m.secrets[idx].isTouched = true
			m.cellInput.SetValue("")
		}
	}

	var cmd tea.Cmd
	m.cellInput, cmd = m.cellInput.Update(msg)
	if m.secretCol == colName {
		normalized := normalizeSecretName(m.cellInput.Value())
		if normalized != m.cellInput.Value() {
			m.cellInput.SetValue(normalized)
			m.cellInput.SetCursor(len(normalized))
		}
	}
	m.applyCellToSelection()
	return m, cmd
}

func (m Model) handleNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pendingYank {
		return m.handleYankMotion(msg)
	}
	chord, ok := encodeKey(msg)
	if !ok {
		return m, nil
	}
	cmd, ok := m.keys.Resolve(m.focus, chord)
	if !ok {
		return m, nil
	}
	return m.executeCommand(cmd)
}

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.commandInput.SetValue("")
		m.completions.Clear()
		m.setFocus(focusSecrets)
		return m, nil
	case "enter":
		if m.completions.Browsed && m.completions.SelectedItem() != nil {
			m.applySelectedCompletion()
			return m, nil
		}
		line := m.commandInput.Value()
		m.commandInput.SetValue("")
		m.completions.Clear()
		m.setFocus(focusSecrets)
		return m.executeCommand(line)
	case "tab":
		m.tabComplete(true)
		return m, nil
	case "shift+tab":
		m.tabComplete(false)
		return m, nil
	case "down":
		if len(m.completions.Items) > 0 {
			m.completions.SelectNext()
			m.completions.Browsed = true
			return m, nil
		}
	case "up":
		if len(m.completions.Items) > 0 {
			m.completions.SelectPrev()
			m.completions.Browsed = true
			return m, nil
		}
	case "ctrl+c":
		return m, tea.Quit
	}
	if m.completions.SelectedItem() != nil && !m.selectionApplied() {
		m.applySelectedCompletion()
	}
	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	m.refreshCompletions()
	return m, cmd
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
	}
}

func (m *Model) jumpListEdge(bottom bool) {
	switch m.focus {
	case focusProjects:
		if len(m.tree) == 0 {
			return
		}
		if bottom {
			m.treeIdx = len(m.tree) - 1
		} else {
			m.treeIdx = 0
		}
	case focusSecrets:
		idxs := m.filteredIndexes()
		if len(idxs) == 0 {
			return
		}
		if bottom {
			m.secretIdx = len(idxs) - 1
		} else {
			m.secretIdx = 0
		}
	}
}

func (m Model) pageSize() int {
	if m.cfg.PageLines > 0 {
		return m.cfg.PageLines
	}
	layout := m.computeLayout()
	chrome := m.panelChrome()
	switch m.focus {
	case focusProjects:
		return max(1, layout.projects.h-chrome)
	case focusSecrets:
		return max(1, layout.secrets.h-chrome-1) // account for header row
	default:
		return 10
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
	m.persistSession()
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
	if m.focus != focusProjects {
		return m, nil
	}
	row, ok := m.currentTreeRow()
	if !ok {
		return m, nil
	}
	if row.kind == treeConfig {
		if row.project == "" || row.config == "" {
			return m, nil
		}
		if row.project == m.activeProject && row.config == m.activeConfig {
			m.setFocus(focusSecrets)
			return m, nil
		}
		if m.hasDirtySecrets() {
			return m.openSwitchConfirm(row.project, row.config)
		}
		m.fetching = true
		m.errMsg = ""
		return m, tea.Batch(m.spinner.Tick, selectConfigCmd(m.opts, row.project, row.config))
	}
	if row.project == "" {
		return m, nil
	}
	if row.project == m.activeProject {
		if m.expanded == nil {
			m.expanded = map[string]bool{}
		}
		m.expanded[row.project] = true
		m.rebuildTree()
		return m, nil
	}
	if m.hasDirtySecrets() {
		return m.openSwitchConfirm(row.project, "")
	}
	m.fetching = true
	m.errMsg = ""
	return m, tea.Batch(m.spinner.Tick, selectProjectCmd(m.opts, row.project, m.activeConfig))
}

func (m Model) addSecret() (tea.Model, tea.Cmd) {
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.secrets = append(m.secrets, newEmptySecretRow())
	m.filter = ""
	m.filterInput.SetValue("")
	idxs := m.filteredIndexes()
	m.secretIdx = len(idxs) - 1
	m.secretCol = colName
	m.noteSecretEdit(len(m.secrets) - 1)
	m.enterInsert()
	return m, nil
}

func (m Model) beginCreateConfig() (tea.Model, tea.Cmd) {
	if len(m.tree) == 0 || m.treeIdx < 0 || m.treeIdx >= len(m.tree) {
		m.errMsg = "Select a project first"
		return m, nil
	}
	row := m.tree[m.treeIdx]
	if row.project == "" {
		m.errMsg = "Select a project first"
		return m, nil
	}

	m.createConfigProject = row.project
	m.createConfigEnv = ""
	m.renameFromConfig = ""
	m.configPromptMode = configPromptCreate
	prefill := ""
	if row.kind == treeConfig && row.config != "" {
		m.createConfigEnv = m.configEnvironment(row.project, row.config)
		if m.createConfigEnv != "" {
			prefill = m.createConfigEnv + "_"
		}
	}
	m.createConfigInput.Prompt = "+ "
	m.createConfigInput.Placeholder = "New config (e.g. dev_personal)…"
	m.createConfigInput.SetValue(prefill)
	m.errMsg = ""
	m.statusMsg = ""
	m.setFocus(focusCreateConfig)
	return m, nil
}

func (m Model) beginRenameConfig() (tea.Model, tea.Cmd) {
	if len(m.tree) == 0 || m.treeIdx < 0 || m.treeIdx >= len(m.tree) {
		m.errMsg = "Select a config to rename"
		return m, nil
	}
	row := m.tree[m.treeIdx]
	if row.kind != treeConfig || row.config == "" {
		m.errMsg = "Select a config to rename"
		return m, nil
	}

	m.createConfigProject = row.project
	m.createConfigEnv = m.configEnvironment(row.project, row.config)
	m.renameFromConfig = row.config
	m.configPromptMode = configPromptRename
	m.createConfigInput.Prompt = "~ "
	m.createConfigInput.Placeholder = "Rename config…"
	m.createConfigInput.SetValue(row.config)
	m.errMsg = ""
	m.statusMsg = ""
	m.setFocus(focusCreateConfig)
	return m, nil
}

// setSelectedConfigLock locks/unlocks the selected config.
// lock == nil toggles based on current state.
func (m Model) setSelectedConfigLock(lock *bool) (tea.Model, tea.Cmd) {
	if len(m.tree) == 0 || m.treeIdx < 0 || m.treeIdx >= len(m.tree) {
		m.errMsg = "Select a config"
		return m, nil
	}
	row := m.tree[m.treeIdx]
	if row.kind != treeConfig || row.config == "" {
		m.errMsg = "Select a config"
		return m, nil
	}

	shouldLock := !row.locked
	if lock != nil {
		shouldLock = *lock
	}
	if shouldLock == row.locked {
		if shouldLock {
			m.statusMsg = row.config + " already locked"
		} else {
			m.statusMsg = row.config + " already unlocked"
		}
		m.errMsg = ""
		return m, nil
	}

	m.errMsg = ""
	m.statusMsg = ""
	m.fetching = true
	return m, tea.Batch(m.spinner.Tick, setConfigLockCmd(m.opts, row.project, row.config, shouldLock))
}

func boolPtr(v bool) *bool { return &v }

func (m Model) configEnvironment(project, configName string) string {
	for _, c := range m.projectConfigs[project] {
		if c.name == configName {
			return c.environment
		}
	}
	return ""
}

func (m Model) deleteSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	if m.secrets[idx].originalName == nil {
		m.secrets = append(m.secrets[:idx], m.secrets[idx+1:]...)
		m.adjustUndoStackForRemoval(idx)
		m.clampSecretIdx()
		return m, nil
	}
	m.secrets[idx].shouldDelete = true
	m.noteSecretEdit(idx)
	return m, nil
}

func (m Model) undoSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.popUndoTarget()
	if !ok {
		return m, nil
	}

	if m.secrets[idx].originalName == nil {
		m.secrets = append(m.secrets[:idx], m.secrets[idx+1:]...)
		m.adjustUndoStackForRemoval(idx)
		m.clampSecretIdx()
		m.setFocus(focusSecrets)
		return m, nil
	}

	m.secrets[idx].undo()
	m.selectSecretByStoreIndex(idx)
	m.setFocus(focusSecrets)
	return m, nil
}

func (m *Model) popUndoTarget() (int, bool) {
	for len(m.undoStack) > 0 {
		idx := m.undoStack[len(m.undoStack)-1]
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		if idx >= 0 && idx < len(m.secrets) && m.secrets[idx].isDirty() {
			return idx, true
		}
	}
	idx, ok := m.selectedSecretIndex()
	if !ok || !m.secrets[idx].isDirty() {
		return 0, false
	}
	return idx, true
}

func (m Model) yankSecret() (tea.Model, tea.Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	text := m.secrets[idx].displayValue()
	if m.secretCol == colName {
		text = m.secrets[idx].name
	}
	_ = utils.CopyToClipboard(text)
	m.statusMsg = "Copied to clipboard"
	return m, nil
}

func (m Model) openSave() (tea.Model, tea.Cmd) {
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.pendingChanges = collectChanges(m.secrets)
	m.modalBtnIdx = 0
	m.focus = focusSave
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.fetching {
		return m, nil
	}
	if m.inModal() {
		return m.handleModalMouse(msg)
	}
	if m.inSecretInsert() {
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
	}

	layout := m.computeLayout()

	scroll := m.cfg.ScrollLines
	if scroll < 1 {
		scroll = 1
	}
	if msg.Button == tea.MouseButtonWheelUp {
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(-scroll)
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(scroll)
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
			visible := max(1, layout.projects.h-m.panelChrome())
			start := clampScrollOffset(m.treeOffset, m.treeIdx, visible, len(m.tree))
			idx := start + rel
			if idx >= 0 && idx < len(m.tree) {
				m.treeIdx = idx
				return m.activateSelection()
			}
		}
	case layout.secrets.contains(msg.X, msg.Y):
		m.setFocus(focusSecrets)
		rel := msg.Y - layout.secrets.y - 2 // title + header
		idxs := m.filteredIndexes()
		visible := max(1, layout.secrets.h-m.panelChrome()-1)
		start := clampScrollOffset(m.secretOffset, m.secretIdx, visible, len(idxs))
		idx := start + rel
		if idx >= 0 && idx < len(idxs) {
			m.secretIdx = idx
			nameW, _ := m.secretColumnWidths(layout.secrets.w)
			relX := msg.X - layout.secrets.x - 1
			if relX >= nameW+lipgloss.Width(secretColSep) {
				m.secretCol = colValue
			} else {
				m.secretCol = colName
			}
		}
	case layout.status.contains(msg.X, msg.Y):
		m.beginSearch()
	}
	return m, nil
}

func (m Model) handleModalMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	buttons := m.currentModalButtons()
	modal := m.currentModalView()
	if modal == "" || len(buttons) == 0 {
		return m, nil
	}
	ox := max(0, (m.width-lipgloss.Width(modal))/2)
	oy := max(0, (m.height-lipgloss.Height(modal))/2)
	for i, r := range buttonHitRects(modal, buttons, ox, oy) {
		if r.contains(msg.X, msg.Y) {
			m.modalBtnIdx = i
			return m.activateFocusedModalButton()
		}
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

func (m Model) handleConfigWatch() (tea.Model, tea.Cmd) {
	if !m.cfg.Autoreload {
		return m, nil
	}
	info, err := os.Stat(configuration.UserConfigFile)
	if err != nil {
		return m, watchConfigCmd()
	}
	mod := info.ModTime()
	if m.configModTime.IsZero() {
		m.configModTime = mod
		return m, watchConfigCmd()
	}
	if !mod.After(m.configModTime) {
		return m, watchConfigCmd()
	}
	m.configModTime = mod
	configuration.ReloadConfigFromDisk()
	newCfg := configuration.TUIConfig()
	if strings.TrimSpace(newCfg.Theme) == "" {
		newCfg.Theme = m.cfg.Theme
	}
	if newCfg.Theme != m.cfg.Theme {
		_ = applyTheme(newCfg.Theme)
	}
	m.cfg = newCfg
	m.keys = MergeKeys(newCfg.Keys)
	if !m.cfg.Sidebar && m.focus == focusProjects {
		m.focus = focusSecrets
	}
	return m, watchConfigCmd()
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
