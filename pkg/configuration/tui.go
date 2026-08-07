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
	Theme             string
	SidebarWidth      int    // 0 = auto
	SidebarPosition   string // left | right
	PageLines         int    // 0 = viewport height
	ScrollLines       int    // mouse wheel step
	Border            bool
	CaseMode          string // sensitive | insensitive | smart
	NameColumnPercent int    // 1-99
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
		Theme:             configContents.TUI.Theme,
		SidebarWidth:      configContents.TUI.SidebarWidth,
		SidebarPosition:   configContents.TUI.SidebarPosition,
		PageLines:         configContents.TUI.PageLines,
		ScrollLines:       configContents.TUI.ScrollLines,
		Border:            boolOrDefault(configContents.TUI.Border, true),
		CaseMode:          configContents.TUI.CaseMode,
		NameColumnPercent: configContents.TUI.NameColumnPercent,
	})
}

// NormalizeTUISettings applies defaults for zero/invalid values.
// Border is left as-is; use TUIConfig() (or set Border explicitly) for the true default.
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
