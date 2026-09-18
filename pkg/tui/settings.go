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
	"strconv"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
)

type tuiSetting struct {
	name   string
	get    func(Model) string
	set    func(*Model, string) error
	values func() []string
}

var tuiSettings = []tuiSetting{
	{
		name: "theme",
		get: func(m Model) string {
			t := strings.TrimSpace(m.cfg.Theme)
			if t == "" {
				return defaultThemeName
			}
			return t
		},
		set: func(m *Model, value string) error {
			if err := applyTheme(value); err != nil {
				return err
			}
			name := strings.TrimSpace(strings.ToLower(value))
			if name == "" {
				name = defaultThemeName
			}
			m.cfg.Theme = name
			return nil
		},
		values: themeNames,
	},
	{
		name: "sidebar",
		get:  func(m Model) string { return formatOnOff(m.cfg.Sidebar) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.Sidebar = on
			if !m.cfg.Sidebar && m.focus == focusProjects {
				m.focus = focusSecrets
			}
			return nil
		},
		values: onOffValues,
	},
	{
		name: "sidebar-width",
		get:  func(m Model) string { return strconv.Itoa(m.cfg.SidebarWidth) },
		set: func(m *Model, value string) error {
			n, err := parseNonNegativeInt(value)
			if err != nil {
				return err
			}
			m.cfg.SidebarWidth = n
			return nil
		},
	},
	{
		name: "sidebar-position",
		get:  func(m Model) string { return m.cfg.SidebarPosition },
		set: func(m *Model, value string) error {
			switch value {
			case "left", "right":
				m.cfg.SidebarPosition = value
				return nil
			default:
				return fmt.Errorf("sidebar-position must be left or right")
			}
		},
		values: func() []string { return []string{"left", "right"} },
	},
	{
		name: "page-lines",
		get:  func(m Model) string { return strconv.Itoa(m.cfg.PageLines) },
		set: func(m *Model, value string) error {
			n, err := parseNonNegativeInt(value)
			if err != nil {
				return err
			}
			m.cfg.PageLines = n
			return nil
		},
	},
	{
		name: "scroll-lines",
		get:  func(m Model) string { return strconv.Itoa(m.cfg.ScrollLines) },
		set: func(m *Model, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fmt.Errorf("scroll-lines must be an integer >= 1")
			}
			m.cfg.ScrollLines = n
			return nil
		},
	},
	{
		name: "border",
		get:  func(m Model) string { return formatOnOff(m.cfg.Border) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.Border = on
			return nil
		},
		values: onOffValues,
	},
	{
		name: "case-mode",
		get:  func(m Model) string { return m.cfg.CaseMode },
		set: func(m *Model, value string) error {
			switch value {
			case "sensitive", "insensitive", "smart":
				m.cfg.CaseMode = value
				return nil
			default:
				return fmt.Errorf("case-mode must be sensitive, insensitive, or smart")
			}
		},
		values: func() []string { return []string{"sensitive", "insensitive", "smart"} },
	},
	{
		name: "name-column-percent",
		get:  func(m Model) string { return strconv.Itoa(m.cfg.NameColumnPercent) },
		set: func(m *Model, value string) error {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 99 {
				return fmt.Errorf("name-column-percent must be 1-99")
			}
			m.cfg.NameColumnPercent = n
			return nil
		},
	},
	{
		name: "list-scrollbar",
		get:  func(m Model) string { return formatOnOff(m.cfg.ListScrollbarVertical) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.ListScrollbarVertical = on
			return nil
		},
		values: onOffValues,
	},
	{
		name: "sidebar-scrollbar",
		get:  func(m Model) string { return formatOnOff(m.cfg.SidebarScrollbarVertical) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.SidebarScrollbarVertical = on
			return nil
		},
		values: onOffValues,
	},
	{
		name: "autosave",
		get:  func(m Model) string { return formatOnOff(m.cfg.Autosave) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.Autosave = on
			return nil
		},
		values: onOffValues,
	},
	{
		name: "autoreload",
		get:  func(m Model) string { return formatOnOff(m.cfg.Autoreload) },
		set: func(m *Model, value string) error {
			on, err := parseOnOffValue(value)
			if err != nil {
				return err
			}
			m.cfg.Autoreload = on
			return nil
		},
		values: onOffValues,
	},
}

func lookupTUISetting(name string) (tuiSetting, bool) {
	for _, s := range tuiSettings {
		if s.name == name {
			return s, true
		}
	}
	return tuiSetting{}, false
}

func tuiSettingNames() []string {
	names := make([]string, 0, len(tuiSettings))
	for _, s := range tuiSettings {
		names = append(names, s.name)
	}
	return names
}

func unknownSettingError(name string) string {
	return fmt.Sprintf("unknown setting %q (available: %s)", name, strings.Join(tuiSettingNames(), ", "))
}

func (m Model) execConfigGet(args []string) (Model, Cmd) {
	if len(args) > 1 {
		m.errMsg = "usage: config get [name]"
		return m, nil
	}
	if len(args) == 0 {
		parts := make([]string, 0, len(tuiSettings))
		for _, s := range tuiSettings {
			parts = append(parts, s.name+"="+s.get(m))
		}
		m.statusMsg = strings.Join(parts, "  ")
		m.errMsg = ""
		return m, nil
	}
	s, ok := lookupTUISetting(args[0])
	if !ok {
		m.errMsg = unknownSettingError(args[0])
		return m, nil
	}
	m.statusMsg = s.name + "=" + s.get(m)
	m.errMsg = ""
	return m, nil
}

func (m Model) execConfigSet(args []string) (Model, Cmd) {
	if len(args) != 2 {
		m.errMsg = "usage: config set <name> <value>"
		return m, nil
	}
	s, ok := lookupTUISetting(args[0])
	if !ok {
		m.errMsg = unknownSettingError(args[0])
		return m, nil
	}
	if err := s.set(&m, args[1]); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	if m.cfg.Autosave {
		configuration.TUISaveSettings(m.cfg)
	}
	m.statusMsg = s.name + "=" + s.get(m)
	m.errMsg = ""
	return m, nil
}

func extraTokenSuggestions(complete []string, partial string, replaceFrom int, leadSpace bool) []Suggestion {
	var choices []string
	helpFor := func(next string) string { return next }
	if len(complete) == 2 && strings.EqualFold(complete[0], "config") {
		switch strings.ToLower(complete[1]) {
		case "get":
			choices = tuiSettingNames()
			helpFor = func(next string) string { return "Show " + next }
		case "set":
			choices = tuiSettingNames()
			helpFor = func(next string) string { return "Set " + next }
		}
	}
	if len(complete) == 3 && strings.EqualFold(complete[0], "config") && strings.EqualFold(complete[1], "set") {
		if s, ok := lookupTUISetting(strings.ToLower(complete[2])); ok && s.values != nil {
			choices = s.values()
		}
	}

	partialL := strings.ToLower(partial)
	var items []Suggestion
	for _, next := range choices {
		if partialL != "" && !strings.HasPrefix(strings.ToLower(next), partialL) {
			continue
		}
		text := next
		if leadSpace {
			text = " " + next
		}
		items = append(items, Suggestion{
			Text:        text,
			Label:       next,
			Help:        helpFor(next),
			ReplaceFrom: replaceFrom,
		})
	}
	return items
}

func formatOnOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func parseOnOffValue(s string) (bool, error) {
	switch s {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func onOffValues() []string {
	return []string{"on", "off"}
}

func parseNonNegativeInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("expected an integer >= 0")
	}
	return n, nil
}
