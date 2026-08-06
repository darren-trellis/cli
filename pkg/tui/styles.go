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

import "github.com/charmbracelet/lipgloss"

var (
	accent     lipgloss.Color
	dim        lipgloss.Color
	background lipgloss.Color
	textColor  lipgloss.Color

	panelStyle       lipgloss.Style
	activePanelStyle lipgloss.Style
	titleStyle       lipgloss.Style
	activeTitleStyle lipgloss.Style
	selectedStyle    lipgloss.Style
	activeEnvStyle   lipgloss.Style
	dimStyle         lipgloss.Style
	dirtyStyle       lipgloss.Style
	deleteStyle      lipgloss.Style
	errorStyle       lipgloss.Style
	statusStyle      lipgloss.Style
	modalStyle       lipgloss.Style
	helpStyle        lipgloss.Style
	appStyle         lipgloss.Style

	roundedBorder = lipgloss.RoundedBorder()
)

func rebuildStyles(theme Theme) {
	accent = theme.Accent
	dim = theme.Dim
	background = theme.Background
	textColor = theme.Text

	borderColor := theme.Border
	if borderColor == "" {
		borderColor = theme.Dim
	}

	panelStyle = lipgloss.NewStyle().
		Border(roundedBorder).
		BorderForeground(borderColor).
		Padding(0, 1)

	activePanelStyle = panelStyle.BorderForeground(theme.Accent)

	titleStyle = lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	activeTitleStyle = titleStyle.Foreground(theme.Accent)

	selectedStyle = lipgloss.NewStyle().Foreground(theme.SelectionFg).Bold(true)
	selectionBg := theme.SelectionBg
	if selectionBg == "" {
		selectionBg = lipgloss.Color("237")
	}
	selectedStyle = selectedStyle.Background(selectionBg)

	activeEnv := theme.ActiveEnv
	if activeEnv == "" {
		activeEnv = theme.Accent
	}
	activeEnvStyle = lipgloss.NewStyle().Foreground(activeEnv).Bold(true)

	dimStyle = lipgloss.NewStyle().Foreground(theme.Dim)
	dirtyStyle = lipgloss.NewStyle().Foreground(theme.Dirty)
	deleteStyle = lipgloss.NewStyle().Foreground(theme.Delete)
	errorStyle = lipgloss.NewStyle().Foreground(theme.Error)
	statusStyle = lipgloss.NewStyle().Foreground(theme.Text)
	helpStyle = lipgloss.NewStyle().Foreground(theme.Dim)

	modalStyle = lipgloss.NewStyle().
		Border(roundedBorder).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Width(60)

	appStyle = lipgloss.NewStyle().Foreground(theme.Text)
	if theme.Background != "" {
		bg := theme.Background
		appStyle = appStyle.Background(bg)
		panelStyle = panelStyle.Background(bg).BorderBackground(bg)
		activePanelStyle = activePanelStyle.Background(bg).BorderBackground(bg)
		modalStyle = modalStyle.Background(bg).BorderBackground(bg)
	}
}
