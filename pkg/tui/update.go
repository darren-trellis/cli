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
)

type configWatchMsg struct{}

func watchConfigCmd() Cmd {
	return Tick(time.Second, func(time.Time) Msg {
		return configWatchMsg{}
	})
}

func (m Model) Update(msg Msg) (Model, Cmd) {
	next, cmd := m.update(msg)
	next.syncScrollOffsets()
	return next, Batch(cmd, next.scheduleHighlightLoad())
}

func (m Model) update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case windowSizeMsg:
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

	case highlightLoadDueMsg:
		return m.handleHighlightLoadDue(msg)

	case highlightLoadedMsg:
		return m.handleHighlightLoaded(msg)

	case configRefreshedMsg:
		return m.handleConfigRefreshed(msg)

	case spinnerTickMsg:
		m.spinner.advance()
		if m.fetching || m.workplaceIndexing || m.highlightLoading != "" {
			return m, m.spinner.Tick
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
		m.rememberLoadedSecrets(msg.activeProject, msg.activeConfig, msg.secrets)
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.persistSession()
		return m, m.startWorkplaceIndex(true)

	case projectSelectedMsg:
		m.fetching = false
		m.errMsg = ""
		m.clearUnsavedConfirm()
		m.projectConfigs[msg.project] = msg.configs
		m.expanded[msg.project] = true
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.secretCol = colName
		m.undoStack = nil
		m.activeProject = msg.project
		m.activeConfig = msg.config
		if msg.config != "" {
			m.rememberLoadedSecrets(msg.project, msg.config, msg.secrets)
		}
		m.pruneSecretsCache()
		m.createConfigProject = ""
		m.createConfigEnv = ""
		m.createConfigPrefix = ""
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
		} else {
			m.treeIdx = findTreeIndex(m.tree, treeProject, msg.project, "")
			m.setFocus(focusProjects)
		}
		m.applyPendingSecretName()
		m.persistSession()
		return m, nil

	case projectDeletedMsg:
		m.fetching = false
		m.errMsg = ""
		m.removeProjectState(msg.deleted)
		m.projects = msg.projects
		m.statusMsg = "Deleted project " + msg.deleted
		if msg.switched {
			m.projectConfigs[msg.project] = msg.configs
			m.expanded[msg.project] = true
			m.secrets = msg.secrets
			m.secretIdx = 0
			m.secretCol = colName
			m.undoStack = nil
			m.activeProject = msg.project
			m.activeConfig = msg.config
			if msg.config != "" {
				m.rememberLoadedSecrets(msg.project, msg.config, msg.secrets)
			}
			m.setFocus(focusProjects)
		} else if m.activeProject == msg.deleted {
			m.activeProject = ""
			m.activeConfig = ""
			m.secrets = nil
			m.undoStack = nil
			m.secretIdx = 0
			m.setFocus(focusProjects)
		}
		m.pruneSecretsCache()
		m.rebuildTree()
		if msg.highlight != "" {
			m.treeIdx = findTreeIndex(m.tree, treeProject, msg.highlight, "")
		} else if msg.switched && msg.project != "" {
			m.treeIdx = findTreeIndex(m.tree, treeProject, msg.project, "")
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

	case workplaceSearchMsg:
		return m.applyWorkplaceSearch(msg)

	case secretsLoadedMsg:
		m.fetching = false
		m.errMsg = ""
		m.clearUnsavedConfirm()
		m.secrets = msg.secrets
		m.secretIdx = 0
		m.secretCol = colName
		m.undoStack = nil
		m.activeProject = msg.activeProject
		m.activeConfig = msg.activeConfig
		m.rememberLoadedSecrets(msg.activeProject, msg.activeConfig, msg.secrets)
		if msg.saved {
			for _, name := range msg.applied {
				m.dropSecretsCache(msg.activeProject, name)
			}
			m.statusMsg = formatSaveStatus(msg.applied, msg.failed, msg.failErr)
		}
		if m.pendingSearchName != "" {
			m.applyPendingSecretName()
		} else if m.searchGlobal {
			m.syncLocalMatchesFromGlobal()
			m.setFocus(focusSecrets)
		}
		if _, ok := m.projectConfigs[msg.activeProject]; ok {
			m.expanded[msg.activeProject] = true
		}
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeConfig, msg.activeProject, msg.activeConfig)
		m.persistSession()
		return m, nil

	case quitNowMsg:
		return m, Quit

	case mouseMsg:
		return m.handleMouse(msg)

	case keyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg keyMsg) (Model, Cmd) {
	if m.fetching {
		switch msg.String() {
		case "C-c":
			return m, Quit
		case "esc":
			if m.searchGlobal {
				m.fetching = false
				m.searchGen++
				m.pendingSearchName = ""
				return m, nil
			}
			return m, nil
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
	case focusDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	case focusPropagate:
		return m.handlePropagateKey(msg)
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

func (m Model) handleHelpKey(msg keyMsg) (Model, Cmd) {
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
	case "C-c":
		return m, Quit
	case "j", "down":
		m.helpViewport.LineDown(1)
	case "k", "up":
		m.helpViewport.LineUp(1)
	case "pagedown":
		m.helpViewport.ViewDown()
	case "pageup":
		m.helpViewport.ViewUp()
	}
	return m, nil
}

func (m Model) handleSaveKey(msg keyMsg) (Model, Cmd) {
	buttons := m.saveModalButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "y":
		return m.requestSaveConfirm()
	case "n", "esc", "q":
		m.clearPropagate()
		m.setFocus(focusSecrets)
		m.pendingChanges = nil
		return m, nil
	case "C-c":
		return m, Quit
	}
	return m, nil
}

func (m Model) handleSwitchConfirmKey(msg keyMsg) (Model, Cmd) {
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
		m.clearUnsavedConfirm()
		m.setFocus(focusSecrets)
		return m, nil
	case "C-c":
		return m, Quit
	}
	return m, nil
}

func (m Model) activateFocusedModalButton() (Model, Cmd) {
	switch m.focus {
	case focusHelp:
		m.setFocus(focusSecrets)
		return m, nil
	case focusSave:
		if len(m.pendingChanges) == 0 || m.modalBtnIdx == 1 {
			m.clearPropagate()
			m.setFocus(focusSecrets)
			m.pendingChanges = nil
			return m, nil
		}
		return m.requestSaveConfirm()
	case focusPropagate:
		if m.modalBtnIdx == 1 {
			return m.cancelPropagate()
		}
		return m.confirmSave()
	case focusSwitchConfirm:
		switch m.modalBtnIdx {
		case 1:
			return m.confirmSwitchDiscard()
		case 2:
			m.clearUnsavedConfirm()
			m.setFocus(focusSecrets)
			return m, nil
		default:
			return m.confirmSwitchSave()
		}
	case focusDeleteConfirm:
		if m.modalBtnIdx == 1 {
			m.clearPendingDelete()
			m.setFocus(focusProjects)
			return m, nil
		}
		return m.confirmPendingDelete()
	default:
		return m, nil
	}
}

func (m *Model) clearUnsavedConfirm() {
	m.pendingChanges = nil
	m.pendingQuit = false
	m.quitDirty = nil
	m.clearPropagate()
}

func (m Model) confirmSwitchSave() (Model, Cmd) {
	if !m.pendingQuit {
		m.clearUnsavedConfirm()
		m.setFocus(focusSecrets)
		return m, nil
	}
	groups := m.quitDirty
	if len(groups) == 0 {
		groups = m.dirtyGroups()
	}
	m.clearUnsavedConfirm()
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusSecrets)
	return m, Batch(m.spinner.Tick, saveAllDirtyCmd(m.opts, groups))
}

func (m Model) confirmSwitchDiscard() (Model, Cmd) {
	m.clearUnsavedConfirm()
	return m, Quit
}

func (m Model) openQuitConfirm() (Model, Cmd) {
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.stashCurrentSecrets()
	m.pendingQuit = true
	m.quitDirty = m.dirtyGroups()
	m.pendingChanges = nil
	m.modalBtnIdx = 0
	m.focus = focusSwitchConfirm
	m.errMsg = ""
	m.statusMsg = ""
	return m, nil
}

func (m Model) requestQuit() (Model, Cmd) {
	m.stashCurrentSecrets()
	if !m.hasDirtySecrets() {
		return m, Quit
	}
	return m.openQuitConfirm()
}

func (m *Model) beginFilter(global bool) {
	m.filterGlobal = global
	if global {
		m.filterInput.Prompt = "F "
		m.filterInput.Placeholder = "Filter all configs…"
	} else {
		m.filterInput.Prompt = "f "
		m.filterInput.Placeholder = "Filter this config…"
	}
	m.setFocus(focusFilter)
}

func (m *Model) commitFilterInput() {
	if m.filterGlobal {
		m.globalFilter = m.filterInput.Value()
	} else {
		m.filter = m.filterInput.Value()
	}
}

func (m Model) handleFilterKey(msg keyMsg) (Model, Cmd) {
	switch msg.String() {
	case "tab", "backtab", "enter", "esc":
		m.commitFilterInput()
		m.secretIdx = 0
		m.setFocus(focusSecrets)
		return m, nil
	case "C-c":
		return m, Quit
	}

	m.filterInput.Update(msg)
	m.commitFilterInput()
	m.clampSecretIdx()
	if m.searchPane == focusSecrets && m.searchRe != nil {
		m.refreshSearchMatches()
	}
	return m, nil
}

func (m Model) handleSearchKey(msg keyMsg) (Model, Cmd) {
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
		m.recordSearchHistory(query)
		m.errMsg = ""
		if m.searchGlobal {
			cmd := Batch(m.startNamesIndex(), m.jumpToGlobalHit(m.searchMatchIdx))
			m.setFocus(focusSecrets)
			return m, cmd
		}
		m.setFocus(m.searchPane)
		return m, nil
	case "tab", "backtab":
		m.setFocus(m.searchPane)
		return m, nil
	case "C-c":
		return m, Quit
	}
	if step := historyStep(msg.String()); step != 0 {
		current := m.searchInput.Value()
		next := current
		ok := false
		if step < 0 {
			next, ok = m.searchHistory.Prev(current)
		} else {
			next, ok = m.searchHistory.Next(current)
		}
		if ok {
			applyHistoryValue(&m.searchInput, next)
			_ = m.applySearch(next, false)
		}
		if m.searchGlobal {
			return m, m.startNamesIndex()
		}
		return m, nil
	}

	m.searchHistory.Reset()
	m.searchInput.Update(msg)
	_ = m.applySearch(m.searchInput.Value(), false)
	if m.searchGlobal {
		return m, m.startNamesIndex()
	}
	return m, nil
}

func (m Model) handleCreateConfigKey(msg keyMsg) (Model, Cmd) {
	switch msg.String() {
	case "esc":
		m.createConfigInput.SetValue("")
		m.createConfigProject = ""
		m.createConfigEnv = ""
		m.createConfigPrefix = ""
		m.renameFromConfig = ""
		m.configPromptMode = configPromptCreate
		m.setFocus(focusProjects)
		return m, nil
	case "enter":
		if m.configPromptMode == configPromptRename {
			return m.submitRenameConfig()
		}
		return m.submitCreateConfig()
	case "C-c":
		return m, Quit
	}

	m.createConfigInput.Update(msg)
	return m, nil
}

func (m Model) createConfigName() string {
	name := strings.TrimSpace(m.createConfigInput.Value())
	if prefix := m.createConfigPrefix; prefix != "" && name != "" && !strings.HasPrefix(name, prefix) {
		name = prefix + name
	}
	return name
}

func (m Model) submitCreateConfig() (Model, Cmd) {
	name := m.createConfigName()
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
	m.createConfigPrefix = ""
	m.renameFromConfig = ""
	m.configPromptMode = configPromptCreate
	m.setFocus(focusProjects)
	return m, Batch(m.spinner.Tick, createConfigCmd(m.opts, project, name, environment))
}

func (m Model) submitRenameConfig() (Model, Cmd) {
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
	return m, Batch(m.spinner.Tick, renameConfigCmd(m.opts, project, from, name))
}

func inferEnvironmentFromConfigName(name string) string {
	idx := strings.Index(name, "_")
	if idx <= 0 {
		return ""
	}
	return name[:idx]
}

func (m Model) handleInsertKey(msg keyMsg) (Model, Cmd) {
	switch msg.String() {
	case "esc":
		m.completions.Clear()
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "enter":
		if m.secretCol == colValue && m.completions.Browsed && m.completions.SelectedItem() != nil {
			m.applyCompletionTo(&m.cellInput)
			m.applyCellToSelection()
			m.refreshSecretRefCompletions()
			return m, nil
		}
		m.completions.Clear()
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
		return m, nil
	case "tab":
		if m.secretCol == colValue && len(m.completions.Items) > 0 {
			m.tabCompleteField(&m.cellInput, true)
			m.applyCellToSelection()
			m.refreshSecretRefCompletions()
			return m, nil
		}
		m.completions.Clear()
		m.applyCellToSelection()
		if m.secretCol == colName {
			m.secretCol = colValue
		} else {
			m.secretCol = colName
		}
		m.loadCellFromSelection()
		m.cellInput.Focus()
		return m, nil
	case "backtab":
		if m.secretCol == colValue && len(m.completions.Items) > 0 {
			m.tabCompleteField(&m.cellInput, false)
			m.applyCellToSelection()
			m.refreshSecretRefCompletions()
			return m, nil
		}
		m.completions.Clear()
		m.applyCellToSelection()
		if m.secretCol == colValue {
			m.secretCol = colName
		} else {
			m.secretCol = colValue
		}
		m.loadCellFromSelection()
		m.cellInput.Focus()
		return m, nil
	case "up":
		if m.secretCol == colValue && len(m.completions.Items) > 0 {
			m.completions.Step(-1)
			return m, nil
		}
	case "down":
		if m.secretCol == colValue && len(m.completions.Items) > 0 {
			m.completions.Step(1)
			return m, nil
		}
	case "C-c":
		return m, Quit
	}

	idx, ok := m.selectedSecretIndex()
	if ok && m.secretCol == colValue &&
		m.secrets[idx].originalVisibility == "restricted" && !m.secrets[idx].isTouched {
		if msg.isTextRune() || msg.Chord == "space" || msg.Chord == "backspace" || msg.Chord == "delete-key" {
			m.secrets[idx].isTouched = true
			m.cellInput.SetValue("")
		}
	}

	m.cellInput.Update(msg)
	if m.secretCol == colName {
		normalized := normalizeSecretName(m.cellInput.Value())
		if normalized != m.cellInput.Value() {
			m.cellInput.SetValue(normalized)
			m.cellInput.SetCursor(len(normalized))
		}
		m.completions.Clear()
	} else {
		m.refreshSecretRefCompletions()
	}
	m.applyCellToSelection()
	return m, nil
}

func (m Model) handleNavKey(msg keyMsg) (Model, Cmd) {
	if m.pendingYank {
		return m.handleYankMotion(msg)
	}
	if m.pendingSecretDelete {
		return m.handleSecretDeleteMotion(msg)
	}
	chord, ok := msg.chord()
	if !ok {
		return m, nil
	}
	if d, isDigit := countDigit(chord); isDigit {
		if d == 0 && m.motionCount == 0 {
			return m, nil
		}
		m.motionCount = m.motionCount*10 + d
		if m.motionCount > 9999 {
			m.motionCount = 9999
		}
		return m, nil
	}
	cmd, ok := m.keys.Resolve(m.focus, chord)
	if !ok {
		m.motionCount = 0
		return m, nil
	}
	if cmd == "config branch" && chord == "n" && m.searchQuery != "" {
		cmd = "search next"
	}
	if !commandUsesMotionCount(cmd) {
		m.motionCount = 0
	}
	return m.executeCommand(cmd)
}

func countDigit(chord string) (int, bool) {
	if len(chord) != 1 || chord[0] < '0' || chord[0] > '9' {
		return 0, false
	}
	return int(chord[0] - '0'), true
}

func commandUsesMotionCount(cmd string) bool {
	switch cmd {
	case "nav up", "nav down", "nav page up", "nav page down", "nav bottom", "search next", "search prev", "yank", "delete":
		return true
	default:
		return false
	}
}

func (m *Model) takeMotionCount() int {
	n, _ := m.takeMotionCountExplicit()
	if n < 1 {
		return 1
	}
	return n
}

func (m *Model) takeMotionCountExplicit() (int, bool) {
	n := m.motionCount
	m.motionCount = 0
	if n < 1 {
		return 1, false
	}
	return n, true
}

func (m Model) handleCommandKey(msg keyMsg) (Model, Cmd) {
	switch msg.String() {
	case "esc":
		m.commandInput.SetValue("")
		m.completions.Clear()
		m.restoreCommandFocus()
		return m, nil
	case "enter":
		if m.completions.Browsed && m.completions.SelectedItem() != nil {
			m.applySelectedCompletion()
			return m, nil
		}
		line := m.commandInput.Value()
		m.commandInput.SetValue("")
		m.completions.Clear()
		m.restoreCommandFocus()
		m.recordCommandHistory(line)
		return m.executeCommand(line)
	case "tab":
		m.tabComplete(true)
		m.completions.Browsed = len(m.completions.Items) > 0
		return m, nil
	case "backtab":
		m.tabComplete(false)
		m.completions.Browsed = len(m.completions.Items) > 0
		return m, nil
	case "C-c":
		return m, Quit
	}
	if step := historyStep(msg.String()); step != 0 {
		if m.stepCommandCompletions(step) {
			return m, nil
		}
		current := m.commandInput.Value()
		next := current
		ok := false
		if step < 0 {
			next, ok = m.commandHistory.Prev(current)
		} else {
			next, ok = m.commandHistory.Next(current)
		}
		if ok {
			applyHistoryValue(&m.commandInput, next)
			m.refreshCompletions()
		}
		return m, nil
	}
	if m.completions.SelectedItem() != nil && !m.selectionApplied() {
		m.applySelectedCompletion()
	}
	m.commandHistory.Reset()
	m.commandInput.Update(msg)
	m.refreshCompletions()
	return m, nil
}

func (m *Model) stepCommandCompletions(step int) bool {
	if len(m.completions.Items) == 0 {
		return false
	}
	if m.completions.Browsed {
		m.completions.Step(step)
		return true
	}
	if step > 0 && !m.commandHistory.Browsing() {
		m.completions.Step(1)
		return true
	}
	return false
}

func (m *Model) moveList(delta int) {
	switch m.focus {
	case focusProjects:
		if len(m.tree) == 0 {
			return
		}
		m.treeIdx = clamp(m.treeIdx+delta, 0, len(m.tree)-1)
		m.revealHighlightedConfig()
	case focusSecrets:
		idxs := m.filteredIndexes()
		if len(idxs) == 0 {
			return
		}
		m.secretIdx = clamp(m.secretIdx+delta, 0, len(idxs)-1)
	}
}

func (m *Model) jumpListIndex(idx int) {
	switch m.focus {
	case focusProjects:
		if len(m.tree) == 0 {
			return
		}
		m.treeIdx = clamp(idx, 0, len(m.tree)-1)
		m.revealHighlightedConfig()
	case focusSecrets:
		idxs := m.filteredIndexes()
		if len(idxs) == 0 {
			return
		}
		m.secretIdx = clamp(idx, 0, len(idxs)-1)
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
		m.revealHighlightedConfig()
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

func (m Model) toggleFold() (Model, Cmd) {
	return m.setFold(nil)
}

func (m Model) setFold(on *bool) (Model, Cmd) {
	row, ok := m.currentTreeRow()
	if !ok {
		return m, nil
	}

	switch {
	case row.kind == treeProject:
		return m.setProjectFold(row.project, on)
	case row.kind == treeConfig && row.hasChildren:
		return m.setEnvFold(row.project, row.foldRoot, on)
	default:
		return m, nil
	}
}

func (m Model) setEnvFold(project, rootConfig string, on *bool) (Model, Cmd) {
	current := isEnvExpanded(m.expandedEnvs, project, rootConfig)
	want := !current
	if on != nil {
		want = *on
	}
	if want == current {
		return m, nil
	}
	if m.expandedEnvs == nil {
		m.expandedEnvs = map[string]bool{}
	}
	m.expandedEnvs[envKey(project, rootConfig)] = want
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, project, rootConfig)
	m.persistSession()
	return m, nil
}

func (m Model) setProjectFold(project string, on *bool) (Model, Cmd) {
	if project == "" {
		return m, nil
	}

	current := m.expanded[project]
	want := !current
	if on != nil {
		want = *on
	}
	if !want {
		if !current {
			return m, nil
		}
		m.expanded[project] = false
		m.rebuildTree()
		m.treeIdx = findTreeIndex(m.tree, treeProject, project, "")
		return m, nil
	}
	if current {
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
	return m, Batch(m.spinner.Tick, fetchProjectConfigsCmd(m.opts, project))
}

func (m Model) activateSelection() (Model, Cmd) {
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
			return m, nil
		}
		m.stashCurrentSecrets()
		if m.applyCachedSecrets(row.project, row.config) {
			return m, nil
		}
		m.fetching = true
		m.errMsg = ""
		return m, Batch(m.spinner.Tick, selectConfigCmd(m.opts, row.project, row.config))
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
	m.stashCurrentSecrets()
	if configs, ok := m.projectConfigs[row.project]; ok && len(configs) > 0 {
		preferred := configs[indexOfConfig(configs, m.activeConfig)].name
		if m.applyCachedSecrets(row.project, preferred) {
			m.expanded[row.project] = true
			m.rebuildTree()
			m.treeIdx = findTreeIndex(m.tree, treeConfig, row.project, preferred)
			return m, nil
		}
		m.fetching = true
		m.errMsg = ""
		return m, Batch(m.spinner.Tick, selectConfigCmd(m.opts, row.project, preferred))
	}
	m.fetching = true
	m.errMsg = ""
	return m, Batch(m.spinner.Tick, selectProjectCmd(m.opts, row.project, m.activeConfig))
}

func (m Model) addSecret() (Model, Cmd) {
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

func (m Model) beginCreateConfig() (Model, Cmd) {
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
	m.createConfigPrefix = ""
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

func (m Model) beginBranchConfig() (Model, Cmd) {
	row, ok := m.currentTreeRow()
	if !ok || row.kind != treeConfig || row.config == "" {
		m.errMsg = "Select a config to branch from"
		return m, nil
	}
	env := m.configEnvironment(row.project, row.config)
	if env == "" {
		m.errMsg = "Config has no environment to branch in"
		return m, nil
	}

	m.createConfigProject = row.project
	m.createConfigEnv = env
	m.createConfigPrefix = env + "_"
	m.renameFromConfig = ""
	m.configPromptMode = configPromptCreate
	m.createConfigInput.Prompt = "+ " + m.createConfigPrefix
	m.createConfigInput.Placeholder = "branch name…"
	m.createConfigInput.SetValue("")
	m.errMsg = ""
	m.statusMsg = ""
	m.setFocus(focusCreateConfig)
	return m, nil
}

func (m Model) beginRenameConfig() (Model, Cmd) {
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
	m.createConfigPrefix = ""
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

func (m Model) beginDeleteSelection() (Model, Cmd) {
	row, ok := m.currentTreeRow()
	if !ok {
		m.errMsg = "Nothing to delete"
		return m, nil
	}
	if row.kind == treeProject {
		return m.beginDeleteProject()
	}
	return m.beginDeleteConfig()
}

func (m Model) beginDeleteProject() (Model, Cmd) {
	row, ok := m.currentTreeRow()
	if !ok || row.project == "" {
		m.errMsg = "Select a project to delete"
		return m, nil
	}
	m.pendingDeleteProject = row.project
	m.pendingDeleteConfig = ""
	m.modalBtnIdx = 0
	m.focus = focusDeleteConfirm
	m.errMsg = ""
	m.statusMsg = ""
	return m, nil
}

func (m Model) beginDeleteConfig() (Model, Cmd) {
	row, ok := m.currentTreeRow()
	if !ok || row.kind != treeConfig || row.config == "" {
		m.errMsg = "Select a config to delete"
		return m, nil
	}
	if m.configIsRoot(row.project, row.config) {
		m.errMsg = "Cannot delete a root config"
		return m, nil
	}
	m.pendingDeleteProject = row.project
	m.pendingDeleteConfig = row.config
	m.modalBtnIdx = 0
	m.focus = focusDeleteConfirm
	m.errMsg = ""
	m.statusMsg = ""
	return m, nil
}

func (m Model) deletingProject() bool {
	return m.pendingDeleteProject != "" && m.pendingDeleteConfig == ""
}

func (m Model) handleDeleteConfirmKey(msg keyMsg) (Model, Cmd) {
	buttons := m.deleteConfirmButtons()
	if delta, ok := modalCycleDelta(msg.String()); ok {
		m.cycleModalButton(len(buttons), delta)
		return m, nil
	}
	switch msg.String() {
	case "enter":
		return m.activateFocusedModalButton()
	case "d":
		return m.confirmPendingDelete()
	case "c", "esc", "q":
		m.clearPendingDelete()
		m.setFocus(focusProjects)
		return m, nil
	case "C-c":
		return m, Quit
	}
	return m, nil
}

func (m *Model) clearPendingDelete() {
	m.pendingDeleteProject = ""
	m.pendingDeleteConfig = ""
}

func (m Model) confirmPendingDelete() (Model, Cmd) {
	if m.deletingProject() {
		return m.confirmDeleteProject()
	}
	return m.confirmDeleteConfig()
}

func (m Model) confirmDeleteProject() (Model, Cmd) {
	project := m.pendingDeleteProject
	stayProject := m.activeProject
	oldProjects := append([]string(nil), m.projects...)
	m.clearPendingDelete()
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusProjects)
	return m, Batch(m.spinner.Tick, deleteProjectCmd(m.opts, project, oldProjects, stayProject))
}

func (m Model) confirmDeleteConfig() (Model, Cmd) {
	project := m.pendingDeleteProject
	config := m.pendingDeleteConfig
	env := m.configEnvironment(project, config)
	stay := m.activeConfig
	m.clearPendingDelete()
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusProjects)
	return m, Batch(m.spinner.Tick, deleteConfigCmd(m.opts, project, config, stay, env))
}

func (m Model) configIsRoot(project, name string) bool {
	for _, c := range m.projectConfigs[project] {
		if c.name == name {
			return c.root
		}
	}
	return false
}

func (m *Model) removeProjectState(project string) {
	delete(m.projectConfigs, project)
	delete(m.expanded, project)
	prefix := project + "\x00"
	for k := range m.expandedEnvs {
		if strings.HasPrefix(k, prefix) {
			delete(m.expandedEnvs, k)
		}
	}
}

// setSelectedConfigLock locks/unlocks the selected config.
// lock == nil toggles based on current state.
func (m Model) setSelectedConfigLock(lock *bool) (Model, Cmd) {
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
	return m, Batch(m.spinner.Tick, setConfigLockCmd(m.opts, row.project, row.config, shouldLock))
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

func (m *Model) deleteSecretAt(idx int) {
	if idx < 0 || idx >= len(m.secrets) {
		return
	}
	if m.secrets[idx].originalName == nil {
		m.secrets = append(m.secrets[:idx], m.secrets[idx+1:]...)
		m.adjustUndoStackForRemoval(idx)
		return
	}
	m.secrets[idx].shouldDelete = true
	m.noteSecretEdit(idx)
}

func (m Model) deleteSecret() (Model, Cmd) {
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return m, nil
	}
	m.deleteSecretAt(idx)
	m.clampSecretIdx()
	return m, nil
}

func (m Model) undoSecret() (Model, Cmd) {
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

func (m Model) yankSecret() (Model, Cmd) {
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

func (m Model) confirmSave() (Model, Cmd) {
	if len(m.pendingChanges) == 0 {
		m.setFocus(focusSecrets)
		m.pendingChanges = nil
		return m, nil
	}
	changes := m.pendingChanges
	extras := m.selectedPropagateConfigs()
	rewriteRefs := m.propagateRewriteRefs
	envs := m.rootConfigNames(m.activeProject)
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.pendingChanges = nil
	m.clearPropagate()
	m.setFocus(focusSecrets)
	return m, Batch(m.spinner.Tick, saveSecretsCmd(m.opts, m.activeProject, m.activeConfig, changes, extras, rewriteRefs, envs))
}

func (m Model) openSave() (Model, Cmd) {
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.pendingChanges = collectChanges(m.secrets)
	m.clearPropagate()
	m.modalBtnIdx = 0
	m.focus = focusSave
	return m, nil
}

func (m Model) handleMouse(msg mouseMsg) (Model, Cmd) {
	if m.fetching {
		return m, nil
	}
	m.motionCount = 0
	m.cancelOperators()
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
	switch msg.Action {
	case mouseWheelUp:
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(-scroll)
		return m, nil
	case mouseWheelDown:
		m.focusPanelAt(msg.X, msg.Y, layout)
		m.moveList(scroll)
		return m, nil
	case mouseLeftClick:
	default:
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
				if m.noteSidebarClick(idx, msg.X, msg.Y) {
					return m.activateSelection()
				}
				m.revealHighlightedConfig()
				return m, nil
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
			if relX >= nameW+textWidth(secretColSep) {
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

type sidebarClick struct {
	x, y int
	idx  int
	at   time.Time
}

const sidebarDoubleClick = 400 * time.Millisecond

func (m *Model) noteSidebarClick(idx, x, y int) bool {
	now := time.Now()
	prev := m.lastSidebarClick
	double := prev.idx == idx &&
		absInt(prev.x-x) <= 1 &&
		absInt(prev.y-y) <= 1 &&
		!prev.at.IsZero() &&
		now.Sub(prev.at) <= sidebarDoubleClick
	if double {
		m.lastSidebarClick = sidebarClick{}
		return true
	}
	m.lastSidebarClick = sidebarClick{x: x, y: y, idx: idx, at: now}
	return false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (m Model) handleModalMouse(msg mouseMsg) (Model, Cmd) {
	if msg.Action != mouseLeftClick {
		return m, nil
	}
	spec, ok := m.currentModalSpec()
	if !ok {
		return m, nil
	}
	g := spec.geometry(m.width, m.height, m.cfg.Border)
	for i, r := range g.buttons {
		if r.contains(msg.X, msg.Y) {
			m.modalBtnIdx = i
			return m.activateFocusedModalButton()
		}
	}
	if m.focus == focusPropagate {
		for i, r := range g.rows {
			if r.contains(msg.X, msg.Y) {
				m.propagateIdx = i
				m.togglePropagateAt(i)
				return m, nil
			}
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

func (m Model) handleConfigWatch() (Model, Cmd) {
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
		if err := applyTheme(newCfg.Theme); err != nil {
			newCfg.Theme = m.cfg.Theme
		}
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
