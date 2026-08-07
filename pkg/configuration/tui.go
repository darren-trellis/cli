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
package configuration

import "strings"

// TUISettings are runtime preferences for the Doppler TUI.
type TUISettings struct {
	Theme                    string
	Sidebar                  bool
	SidebarWidth             int    // 0 = auto
	SidebarPosition          string // left | right
	PageLines                int    // 0 = viewport height
	ScrollLines              int    // mouse wheel step
	Border                   bool
	CaseMode                 string // sensitive | insensitive | smart
	NameColumnPercent        int    // 1-99
	ListScrollbarVertical    bool
	SidebarScrollbarVertical bool
	Autosave                 bool
	Autoreload               bool
}

var CURRENT_INTRO_VERSION = 1

func TUIShouldShowIntro() bool {
	return configContents.TUI.IntroVersionSeen != CURRENT_INTRO_VERSION
}

func TUIMarkIntroSeen() {
	configContents.TUI.IntroVersionSeen = CURRENT_INTRO_VERSION
	writeConfig(configContents)
}

func TUITheme() string {
	return configContents.TUI.Theme
}

func TUISetTheme(theme string) {
	configContents.TUI.Theme = theme
	writeConfig(configContents)
}

func TUIConfig() TUISettings {
	return NormalizeTUISettings(TUISettings{
		Theme:                    configContents.TUI.Theme,
		Sidebar:                  boolOrDefault(configContents.TUI.Sidebar, true),
		SidebarWidth:             configContents.TUI.SidebarWidth,
		SidebarPosition:          configContents.TUI.SidebarPosition,
		PageLines:                configContents.TUI.PageLines,
		ScrollLines:              configContents.TUI.ScrollLines,
		Border:                   boolOrDefault(configContents.TUI.Border, true),
		CaseMode:                 configContents.TUI.CaseMode,
		NameColumnPercent:        configContents.TUI.NameColumnPercent,
		ListScrollbarVertical:    boolOrDefault(configContents.TUI.ListScrollbarVertical, true),
		SidebarScrollbarVertical: boolOrDefault(configContents.TUI.SidebarScrollbarVertical, true),
		Autosave:                 boolOrDefault(configContents.TUI.Autosave, true),
		Autoreload:               boolOrDefault(configContents.TUI.Autoreload, true),
	})
}

// NormalizeTUISettings applies defaults for zero/invalid string/int values.
// Bools are left as-is; use TUIConfig() for YAML defaults.
func NormalizeTUISettings(s TUISettings) TUISettings {
	s.SidebarPosition = strings.ToLower(strings.TrimSpace(s.SidebarPosition))
	if s.SidebarPosition != "right" {
		s.SidebarPosition = "left"
	}
	if s.ScrollLines < 1 {
		s.ScrollLines = 1
	}
	s.CaseMode = strings.ToLower(strings.TrimSpace(s.CaseMode))
	switch s.CaseMode {
	case "sensitive", "insensitive", "smart":
	default:
		s.CaseMode = "smart"
	}
	if s.NameColumnPercent < 1 || s.NameColumnPercent > 99 {
		s.NameColumnPercent = 40
	}
	return s
}

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func boolPtr(v bool) *bool {
	return &v
}

// TUISaveSettings writes the full runtime settings to disk.
func TUISaveSettings(s TUISettings) {
	s = NormalizeTUISettings(s)
	configContents.TUI.Theme = s.Theme
	configContents.TUI.Sidebar = boolPtr(s.Sidebar)
	configContents.TUI.SidebarWidth = s.SidebarWidth
	configContents.TUI.SidebarPosition = s.SidebarPosition
	configContents.TUI.PageLines = s.PageLines
	configContents.TUI.ScrollLines = s.ScrollLines
	configContents.TUI.Border = boolPtr(s.Border)
	configContents.TUI.CaseMode = s.CaseMode
	configContents.TUI.NameColumnPercent = s.NameColumnPercent
	configContents.TUI.ListScrollbarVertical = boolPtr(s.ListScrollbarVertical)
	configContents.TUI.SidebarScrollbarVertical = boolPtr(s.SidebarScrollbarVertical)
	configContents.TUI.Autosave = boolPtr(s.Autosave)
	configContents.TUI.Autoreload = boolPtr(s.Autoreload)
	writeConfig(configContents)
}

func TUISetSidebarWidth(width int) {
	configContents.TUI.SidebarWidth = width
	writeConfig(configContents)
}

func TUISetPageLines(lines int) {
	configContents.TUI.PageLines = lines
	writeConfig(configContents)
}

func TUISetSidebarPosition(pos string) {
	configContents.TUI.SidebarPosition = pos
	writeConfig(configContents)
}

func TUISetScrollLines(lines int) {
	configContents.TUI.ScrollLines = lines
	writeConfig(configContents)
}

func TUISetBorder(border bool) {
	configContents.TUI.Border = boolPtr(border)
	writeConfig(configContents)
}

func TUISetCaseMode(mode string) {
	configContents.TUI.CaseMode = mode
	writeConfig(configContents)
}

func TUISetNameColumnPercent(pct int) {
	configContents.TUI.NameColumnPercent = pct
	writeConfig(configContents)
}

func TUISetSidebar(enabled bool) {
	configContents.TUI.Sidebar = boolPtr(enabled)
	writeConfig(configContents)
}

func TUISetListScrollbarVertical(enabled bool) {
	configContents.TUI.ListScrollbarVertical = boolPtr(enabled)
	writeConfig(configContents)
}

func TUISetSidebarScrollbarVertical(enabled bool) {
	configContents.TUI.SidebarScrollbarVertical = boolPtr(enabled)
	writeConfig(configContents)
}

func TUISetAutosave(enabled bool) {
	configContents.TUI.Autosave = boolPtr(enabled)
	writeConfig(configContents)
}

func TUISetAutoreload(enabled bool) {
	configContents.TUI.Autoreload = boolPtr(enabled)
	writeConfig(configContents)
}

// ReloadConfigFromDisk re-reads the user config file into memory.
func ReloadConfigFromDisk() {
	configContents, _, _ = readConfig()
}
