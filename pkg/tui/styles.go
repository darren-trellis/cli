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
	magenta = lipgloss.Color("5")
	yellow  = lipgloss.Color("3")
	red     = lipgloss.Color("1")
	dim     = lipgloss.Color("8")
	white   = lipgloss.Color("15")

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dim).
			Padding(0, 1)

	activePanelStyle = panelStyle.BorderForeground(magenta)

	titleStyle = lipgloss.NewStyle().Foreground(white).Bold(true)

	activeTitleStyle = titleStyle.Foreground(magenta)

	roundedBorder = lipgloss.RoundedBorder()

	selectedStyle = lipgloss.NewStyle().Foreground(magenta).Bold(true)

	dimStyle = lipgloss.NewStyle().Foreground(dim)

	dirtyStyle = lipgloss.NewStyle().Foreground(yellow)

	deleteStyle = lipgloss.NewStyle().Foreground(red)

	errorStyle = lipgloss.NewStyle().Foreground(red)

	statusStyle = lipgloss.NewStyle().Foreground(white)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(magenta).
			Padding(1, 2).
			Width(60)

	helpStyle = lipgloss.NewStyle().Foreground(dim)
)
