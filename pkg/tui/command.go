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
)

type commandInfo struct {
	name string
	help string
}

var commandCatalog = []commandInfo{
	{"quit", "Exit the TUI"},
	{"help", "Show help"},
	{"nav up", "Move selection up"},
	{"nav down", "Move selection down"},
	{"nav top", "Jump to top"},
	{"nav bottom", "Jump to bottom"},
	{"nav page", "Page up or down"},
	{"nav page up", "Page up"},
	{"nav page down", "Page down"},
	{"nav left", "Focus name column"},
	{"nav right", "Focus value column"},
	{"focus cycle", "Cycle panes forward"},
	{"focus prev", "Cycle panes backward"},
	{"focus projects", "Focus projects sidebar"},
	{"focus secrets", "Focus secrets pane"},
	{"fold on", "Expand project or env"},
	{"fold off", "Collapse project or env"},
	{"fold toggle", "Fold/unfold project or env"},
	{"edit", "Edit selected secret cell"},
	{"secret change", "Clear selected secret cell and edit"},
	{"secret open", "Jump to a Doppler secret reference in the selected value, or edit"},
	{"secret add", "Add a secret"},
	{"secret delete", "Delete/mark delete the current secret"},
	{"delete", "Delete secrets with a motion, or a project/config"},
	{"secret undo", "Undo last secret change"},
	{"secret yank", "Copy focused cell"},
	{"yank", "Start yank operator"},
	{"yank name", "Copy config or project name"},
	{"yank yaml", "Copy secrets as YAML"},
	{"yank json", "Copy secrets as JSON"},
	{"yank env", "Copy secrets as env"},
	{"paste", "Paste secrets from clipboard"},
	{"secret save", "Open save prompt (root configs can apply to other environments)"},
	{"search", "Open search"},
	{"search global", "Filter secret names across configs"},
	{"search next", "Next search match (or next cached config)"},
	{"search prev", "Previous search match (or previous cached config)"},
	{"search clear", "Clear search"},
	{"filter", "Open local secrets filter"},
	{"filter local", "Open local secrets filter"},
	{"filter global", "Open workplace secrets filter"},
	{"config create", "Create a config"},
	{"config rename", "Rename selected config"},
	{"config refresh", "Reload the selected config's secrets, or a project's configs"},
	{"config branch", "Create a branch config in the selected config's environment"},
	{"config lock", "Lock or unlock the selected config"},
	{"config lock on", "Lock selected config"},
	{"config lock off", "Unlock selected config"},
	{"config lock toggle", "Toggle lock on selected config"},
	{"config load", "Load or unload the selected config"},
	{"config load on", "Load selected config"},
	{"config load off", "Unload selected config"},
	{"config load toggle", "Toggle load on selected config"},
	{"config delete", "Delete selected config"},
	{"config get", "Show TUI settings"},
	{"config set", "Change a TUI setting"},
	{"project delete", "Delete selected project"},
	{"sidebar on", "Show projects sidebar"},
	{"sidebar off", "Hide projects sidebar"},
	{"sidebar toggle", "Toggle projects sidebar"},
	{"command", "Open command prompt"},
	{"command clear", "Clear status / search"},
}

func (m *Model) beginCommand() {
	m.cancelOperators()
	m.motionCount = 0
	m.commandInput.SetValue("")
	m.statusMsg = ""
	m.errMsg = ""
	m.commandHistory.Reset()
	if m.focus != focusCommand {
		m.commandReturnFocus = m.focus
	}
	m.setFocus(focusCommand)
	m.refreshCompletions()
}

func (m *Model) restoreCommandFocus() {
	f := m.commandReturnFocus
	if f == focusProjects && !m.cfg.Sidebar {
		f = focusSecrets
	}
	if f != focusProjects && f != focusSecrets {
		f = focusSecrets
	}
	m.setFocus(f)
}

func (m Model) executeCommand(line string) (Model, Cmd) {
	line = strings.TrimSpace(line)
	if line == "" {
		return m, nil
	}
	fields := strings.Fields(strings.ToLower(line))
	verb := fields[0]
	args := fields[1:]

	switch verb {
	case "quit":
		return m.requestQuit()
	case "help":
		m.focus = focusHelp
		m.modalBtnIdx = 0
		m.helpViewport.SetContent(m.renderHelpText())
		m.helpViewport.GotoTop()
		return m, nil
	case "nav":
		return m.execNav(args)
	case "focus":
		return m.execFocus(args)
	case "fold":
		return m.execOnOffToggle(args, "fold", m.foldOn, m.foldOff, m.toggleFold)
	case "select":
		return m.activateSelection()
	case "edit":
		if m.focus == focusSecrets {
			m.enterInsert()
		}
		return m, nil
	case "secret":
		return m.execSecret(args)
	case "yank":
		return m.execYank(args)
	case "delete":
		return m.execDelete()
	case "unload":
		return m.unloadHighlightedConfig()
	case "project":
		return m.execProject(args)
	case "paste":
		return m.pasteSecrets()
	case "search":
		return m.execSearch(args)
	case "filter":
		return m.execFilter(args)
	case "config":
		if len(args) == 0 {
			m.errMsg = "usage: config create|branch|rename|refresh|lock|load|delete|get|set"
			return m, nil
		}
		switch args[0] {
		case "create":
			return m.beginCreateConfig()
		case "rename":
			return m.beginRenameConfig()
		case "branch":
			return m.beginBranchConfig()
		case "refresh":
			return m.refreshSelection()
		case "lock":
			return m.execOnOffToggle(args[1:], "config lock", m.lockOn, m.lockOff, m.lockToggle)
		case "unlock":
			return m.lockOff()
		case "load":
			return m.execOnOffToggle(args[1:], "config load", m.loadOn, m.loadOff, m.loadToggle)
		case "delete":
			return m.beginDeleteConfig()
		case "get":
			return m.execConfigGet(args[1:])
		case "set":
			return m.execConfigSet(args[1:])
		default:
			m.errMsg = "unknown config command"
			return m, nil
		}
	case "sidebar":
		return m.execOnOffToggle(args, "sidebar", m.sidebarOn, m.sidebarOff, m.sidebarToggle)
	case "command":
		if len(args) > 0 && args[0] == "clear" {
			m.statusMsg = ""
			m.errMsg = ""
			m.clearSearch()
			m.cancelOperators()
			m.motionCount = 0
			return m, nil
		}
		if len(args) > 0 {
			m.errMsg = "unknown command subcommand"
			return m, nil
		}
		m.beginCommand()
		return m, nil
	default:
		m.errMsg = fmt.Sprintf("unknown command: %s", line)
		m.statusMsg = ""
		return m, nil
	}
}

func parseOnOffToggle(args []string) (on *bool, ok bool) {
	if len(args) == 0 {
		return nil, false
	}
	switch args[0] {
	case "on":
		return boolPtr(true), true
	case "off":
		return boolPtr(false), true
	case "toggle":
		return nil, true
	default:
		return nil, false
	}
}

func (m Model) execOnOffToggle(args []string, usage string, on, off, toggle func() (Model, Cmd)) (Model, Cmd) {
	mode, ok := parseOnOffToggle(args)
	if !ok {
		m.errMsg = "usage: " + usage + " on|off|toggle"
		return m, nil
	}
	if mode == nil {
		return toggle()
	}
	if *mode {
		return on()
	}
	return off()
}

func (m Model) lockOn() (Model, Cmd) {
	return m.setSelectedConfigLock(boolPtr(true))
}

func (m Model) lockOff() (Model, Cmd) {
	return m.setSelectedConfigLock(boolPtr(false))
}

func (m Model) lockToggle() (Model, Cmd) {
	return m.setSelectedConfigLock(nil)
}

func (m Model) loadOn() (Model, Cmd) {
	return m.activateSelection()
}

func (m Model) loadOff() (Model, Cmd) {
	return m.unloadHighlightedConfig()
}

func (m Model) loadToggle() (Model, Cmd) {
	if m.focus != focusProjects {
		return m.activateSelection()
	}
	row, ok := m.currentTreeRow()
	if !ok || row.kind != treeConfig || row.config == "" {
		return m.activateSelection()
	}
	loaded := m.configIsCached(row.project, row.config) ||
		(row.project == m.activeProject && row.config == m.activeConfig)
	if loaded {
		return m.unloadHighlightedConfig()
	}
	return m.activateSelection()
}

func (m Model) foldOn() (Model, Cmd) {
	return m.setFold(boolPtr(true))
}

func (m Model) foldOff() (Model, Cmd) {
	return m.setFold(boolPtr(false))
}

func (m Model) sidebarOn() (Model, Cmd) {
	m.setSidebar(boolPtr(true))
	return m, nil
}

func (m Model) sidebarOff() (Model, Cmd) {
	m.setSidebar(boolPtr(false))
	return m, nil
}

func (m Model) sidebarToggle() (Model, Cmd) {
	m.setSidebar(nil)
	return m, nil
}

func (m Model) execNav(args []string) (Model, Cmd) {
	if len(args) == 0 {
		m.errMsg = "nav requires a direction"
		return m, nil
	}
	switch args[0] {
	case "up":
		m.moveList(-m.takeMotionCount())
	case "down":
		m.moveList(m.takeMotionCount())
	case "top":
		m.motionCount = 0
		m.jumpListEdge(false)
	case "bottom":
		if n, ok := m.takeMotionCountExplicit(); ok {
			m.jumpListIndex(n - 1)
		} else {
			m.jumpListEdge(true)
		}
	case "page":
		if len(args) < 2 {
			m.errMsg = "nav page requires up or down"
			return m, nil
		}
		switch args[1] {
		case "up":
			m.moveList(-m.pageSize() * m.takeMotionCount())
		case "down":
			m.moveList(m.pageSize() * m.takeMotionCount())
		default:
			m.errMsg = "nav page requires up or down"
		}
	case "left":
		if m.focus == focusSecrets {
			m.secretCol = colName
		}
	case "right":
		if m.focus == focusSecrets {
			m.secretCol = colValue
		}
	default:
		m.errMsg = "unknown nav command"
	}
	return m, nil
}

func (m Model) execFocus(args []string) (Model, Cmd) {
	if len(args) == 0 {
		m.errMsg = "focus requires a target"
		return m, nil
	}
	switch args[0] {
	case "cycle", "next":
		m.cyclePane(1)
	case "prev":
		m.cyclePane(-1)
	case "projects":
		if m.cfg.Sidebar {
			m.setFocus(focusProjects)
		}
	case "secrets":
		m.setFocus(focusSecrets)
	default:
		m.errMsg = "unknown focus command"
	}
	return m, nil
}

func (m Model) execSecret(args []string) (Model, Cmd) {
	if len(args) == 0 {
		m.errMsg = "secret requires an action"
		return m, nil
	}
	switch args[0] {
	case "add":
		return m.addSecret()
	case "change":
		if m.focus == focusSecrets {
			m.enterChange()
		}
		return m, nil
	case "open":
		return m.openSecretLink()
	case "delete":
		return m.deleteSecret()
	case "undo":
		return m.undoSecret()
	case "yank":
		return m.yankSecret()
	case "save":
		return m.openSave()
	default:
		m.errMsg = "unknown secret command"
		return m, nil
	}
}

func (m Model) execDelete() (Model, Cmd) {
	switch m.focus {
	case focusSecrets:
		m.beginSecretDelete()
		return m, nil
	case focusProjects:
		m.motionCount = 0
		return m.beginDeleteSelection()
	default:
		m.errMsg = "delete is only available in Projects or Secrets"
		return m, nil
	}
}

func (m Model) execProject(args []string) (Model, Cmd) {
	if len(args) == 0 {
		m.errMsg = "usage: project delete"
		return m, nil
	}
	switch args[0] {
	case "delete":
		m.motionCount = 0
		return m.beginDeleteProject()
	default:
		m.errMsg = "unknown project command"
		return m, nil
	}
}

func (m Model) execFilter(args []string) (Model, Cmd) {
	global := false
	if len(args) > 0 {
		switch args[0] {
		case "global":
			global = true
		case "local":
			global = false
		default:
			m.errMsg = "usage: filter [local|global]"
			return m, nil
		}
	}
	m.beginFilter(global)
	return m, nil
}

func (m Model) execSearch(args []string) (Model, Cmd) {
	if len(args) == 0 {
		m.beginSearch()
		return m, nil
	}
	switch args[0] {
	case "next":
		if m.searchQuery == "" {
			for i := 0; i < m.takeMotionCount(); i++ {
				m.stepCachedConfig(1)
			}
			return m, nil
		}
		var cmd Cmd
		for i := 0; i < m.takeMotionCount(); i++ {
			cmd = m.stepSearchMatch(1)
		}
		return m, cmd
	case "prev":
		if m.searchQuery == "" {
			for i := 0; i < m.takeMotionCount(); i++ {
				m.stepCachedConfig(-1)
			}
			return m, nil
		}
		var cmd Cmd
		for i := 0; i < m.takeMotionCount(); i++ {
			cmd = m.stepSearchMatch(-1)
		}
		return m, cmd
	case "global":
		return m, m.beginGlobalSearch()
	case "clear":
		m.clearSearch()
		m.statusMsg = ""
		m.errMsg = ""
	default:
		m.errMsg = "unknown search command"
	}
	return m, nil
}

func formatKeyList(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = displayKey(k)
	}
	return strings.Join(parts, " / ")
}

func (m Model) formatHelpKeys(focus focusArea, cmd string) string {
	if keys := formatKeyList(m.keys.BindingsForCommand(focus, cmd)); keys != "" {
		return keys
	}
	if hint := operatorKeyHint(focus, cmd); hint != "" {
		return hint
	}
	return ":" + cmd
}

func operatorKeyHint(focus focusArea, cmd string) string {
	switch {
	case focus == focusProjects && cmd == "yank name":
		return "y n"
	case focus == focusProjects && cmd == "yank yaml":
		return "y y"
	case focus == focusProjects && cmd == "yank json":
		return "y j"
	case focus == focusProjects && cmd == "yank env":
		return "y e"
	case focus == focusSecrets && cmd == "secret yank":
		return "y n"
	default:
		return ""
	}
}

func (m Model) renderHelpText() string {
	var b strings.Builder
	b.WriteString("Commands are bound to keys under tui.keys in ~/.doppler/.doppler.yaml\n")
	b.WriteString("Type : to run a command by name.\n\n")

	writeSection := func(title string, focus focusArea, cmds []string) {
		b.WriteString(title)
		b.WriteByte('\n')
		for _, cmd := range cmds {
			keys := m.formatHelpKeys(focus, cmd)
			help := ""
			for _, info := range commandCatalog {
				if info.name == cmd {
					help = info.help
					break
				}
			}
			if help == "" {
				help = cmd
			}
			b.WriteString(fmt.Sprintf("    %-18s %s\n", keys, help))
		}
		b.WriteByte('\n')
	}

	writeSection("Global:", focusSecrets, []string{
		"quit", "help", "command", "command clear", "search", "search global", "search next", "search prev", "search clear",
		"filter", "filter global", "sidebar toggle", "config get", "config set", "focus cycle", "focus prev", "focus projects", "focus secrets",
	})
	writeSection("Navigation:", focusSecrets, []string{
		"nav up", "nav down", "nav top", "nav bottom", "nav page up", "nav page down", "nav left", "nav right",
	})
	writeSection("Projects:", focusProjects, []string{
		"fold toggle", "config load on", "config load off", "config refresh", "config create", "config branch", "config rename", "config lock toggle", "delete",
		"yank name", "yank yaml", "yank json", "yank env",
	})
	writeSection("Secrets:", focusSecrets, []string{
		"edit", "secret change", "secret open", "secret add", "delete", "secret undo", "yank", "secret yank", "paste", "secret save",
	})
	b.WriteString("Typing modes (search/filter/insert/create/rename) use Esc/Enter locally.\n")
	b.WriteString("Command and search ↑/↓ (C-p/C-n) recall history; ↓ in : focuses suggestions.\n")
	b.WriteString("Counts: 7j / 3k / 10G (G with a count jumps to that row).\n")
	b.WriteString("Sidebar: ◇ expanded, ◆ folded; ◉ after a config means it is locked; loaded configs are coloured. Highlighting a config loads and shows its secrets. n/N hop search matches; in Projects without a search, n adds a branch config and N hops loaded configs. Backspace unloads a config. Click selects a sidebar row.\n")
	b.WriteString("Search: / filter secret names across configs (typeahead); :search highlights in the current pane.\n")
	b.WriteString("Filter: f this config; F every config (local applies after global).\n")
	b.WriteString("Projects yank: y then n/y/j/e (name / yaml / json / env).\n")
	b.WriteString("Projects delete: d on a project or branch config (root configs cannot be deleted).\n")
	b.WriteString("Secrets yank: y[j|e] then motion (yy line, yn cell, y2j current+2 down, yjy json line).\n")
	b.WriteString("Secrets delete: dd line, d2j current+2 down, 5dd 5 lines.\n")
	b.WriteString("Secrets paste: p imports yaml/json/env from the clipboard.\n")
	b.WriteString("Enter on a value like ${project.config.SECRET} jumps to that secret; i still edits.\n")
	b.WriteString("c clears the selected cell and edits.\n")
	b.WriteString("While editing a value, { opens project.config.secret suggestions at the cursor; Tab completes.\n")
	b.WriteString("Saving a root config asks whether to apply the same changes to other environments, and can rewrite Doppler references to match each environment.\n")
	return b.String()
}
