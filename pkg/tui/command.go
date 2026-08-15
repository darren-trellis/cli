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

	tea "github.com/charmbracelet/bubbletea"
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
	{"nav page up", "Page up"},
	{"nav page down", "Page down"},
	{"nav left", "Focus name column"},
	{"nav right", "Focus value column"},
	{"focus cycle", "Cycle panes forward"},
	{"focus prev", "Cycle panes backward"},
	{"focus projects", "Focus projects sidebar"},
	{"focus secrets", "Focus secrets pane"},
	{"fold toggle", "Fold/unfold project or env"},
	{"select", "Select project or config"},
	{"edit", "Edit selected secret cell"},
	{"secret add", "Add a secret"},
	{"secret delete", "Delete/mark delete secret"},
	{"secret undo", "Undo last secret change"},
	{"secret yank", "Copy active cell"},
	{"yank", "Start yank in Projects (n/y/j/e)"},
	{"yank name", "Copy config or project name"},
	{"yank yaml", "Copy secrets as YAML"},
	{"yank json", "Copy secrets as JSON"},
	{"yank env", "Copy secrets as env"},
	{"paste", "Paste secrets from clipboard"},
	{"secret save", "Open save prompt"},
	{"search", "Open search"},
	{"search next", "Next search match"},
	{"search prev", "Previous search match"},
	{"search clear", "Clear search"},
	{"filter", "Open secrets filter"},
	{"config create", "Create a config"},
	{"config rename", "Rename selected config"},
	{"config lock", "Lock selected config"},
	{"config unlock", "Unlock selected config"},
	{"config lock toggle", "Toggle lock on selected config"},
	{"config delete", "Delete selected config"},
	{"sidebar toggle", "Toggle projects sidebar"},
	{"command", "Open command prompt"},
	{"command clear", "Clear status / search"},
}

func (m *Model) beginCommand() {
	m.cancelYank()
	m.motionCount = 0
	m.commandInput.SetValue("")
	m.statusMsg = ""
	m.errMsg = ""
	m.setFocus(focusCommand)
	m.refreshCompletions()
}

func (m Model) executeCommand(line string) (tea.Model, tea.Cmd) {
	line = strings.TrimSpace(line)
	if line == "" {
		return m, nil
	}
	fields := strings.Fields(strings.ToLower(line))
	verb := fields[0]
	args := fields[1:]

	switch verb {
	case "quit":
		return m, tea.Quit
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
		if len(args) == 0 || args[0] == "toggle" {
			return m.toggleFold()
		}
		m.errMsg = "unknown fold command"
		return m, nil
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
	case "paste":
		return m.pasteSecrets()
	case "search":
		return m.execSearch(args)
	case "filter":
		m.setFocus(focusFilter)
		return m, nil
	case "config":
		if len(args) == 0 {
			m.errMsg = "usage: config create|rename|lock|unlock|delete"
			return m, nil
		}
		switch args[0] {
		case "create":
			return m.beginCreateConfig()
		case "rename":
			return m.beginRenameConfig()
		case "lock":
			if len(args) > 1 && args[1] == "toggle" {
				return m.setSelectedConfigLock(nil)
			}
			return m.setSelectedConfigLock(boolPtr(true))
		case "unlock":
			return m.setSelectedConfigLock(boolPtr(false))
		case "delete":
			return m.beginDeleteConfig()
		default:
			m.errMsg = "unknown config command"
			return m, nil
		}
	case "sidebar":
		if len(args) > 0 && args[0] == "toggle" {
			m.toggleSidebar()
			return m, nil
		}
		m.errMsg = "unknown sidebar command"
		return m, nil
	case "command":
		if len(args) > 0 && args[0] == "clear" {
			m.statusMsg = ""
			m.errMsg = ""
			m.clearSearch()
			m.cancelYank()
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

func (m Model) execNav(args []string) (tea.Model, tea.Cmd) {
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

func (m Model) execFocus(args []string) (tea.Model, tea.Cmd) {
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

func (m Model) execSecret(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.errMsg = "secret requires an action"
		return m, nil
	}
	switch args[0] {
	case "add":
		return m.addSecret()
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

func (m Model) execSearch(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.beginSearch()
		return m, nil
	}
	switch args[0] {
	case "next":
		for i := 0; i < m.takeMotionCount(); i++ {
			m.stepSearchMatch(1)
		}
	case "prev":
		for i := 0; i < m.takeMotionCount(); i++ {
			m.stepSearchMatch(-1)
		}
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

func (m Model) renderHelpText() string {
	var b strings.Builder
	b.WriteString("Commands are bound to keys under tui.keys in ~/.doppler/.doppler.yaml\n")
	b.WriteString("Type : to run a command by name.\n\n")

	writeSection := func(title string, focus focusArea, cmds []string) {
		b.WriteString(title)
		b.WriteByte('\n')
		for _, cmd := range cmds {
			keys := formatKeyList(m.keys.BindingsForCommand(focus, cmd))
			if keys == "" {
				keys = "(unbound)"
			}
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
		"quit", "help", "command", "command clear", "search", "search next", "search prev", "search clear",
		"filter", "sidebar toggle", "focus cycle", "focus prev", "focus projects", "focus secrets",
	})
	writeSection("Navigation:", focusSecrets, []string{
		"nav up", "nav down", "nav top", "nav bottom", "nav page up", "nav page down", "nav left", "nav right",
	})
	writeSection("Projects:", focusProjects, []string{
		"fold toggle", "select", "config create", "config rename", "config lock toggle", "config delete",
		"yank name", "yank yaml", "yank json", "yank env",
	})
	writeSection("Secrets:", focusSecrets, []string{
		"edit", "secret add", "secret delete", "secret undo", "secret yank", "paste", "secret save",
	})
	b.WriteString("Typing modes (search/filter/insert/create/rename) use Esc/Enter locally.\n")
	b.WriteString("Counts: 7j / 3k / 10G (G with a count jumps to that row).\n")
	b.WriteString("Projects yank: y then n/y/j/e (name / yaml / json / env).\n")
	b.WriteString("Secrets paste: p imports yaml/json/env from the clipboard.\n")
	return b.String()
}
