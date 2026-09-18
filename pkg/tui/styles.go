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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Raw theme colours, kept separately from the composed styles below because a
// few call sites (borders, scrollbars) need the colour rather than a style.
var (
	accent      tcell.Color
	dim         tcell.Color
	background  tcell.Color
	textColor   tcell.Color
	borderColor tcell.Color
	selectionBg tcell.Color
	activeEnv   tcell.Color

	baseStyle           tcell.Style
	dimStyle            tcell.Style
	dirtyStyle          tcell.Style
	deleteStyle         tcell.Style
	errorStyle          tcell.Style
	statusStyle         tcell.Style
	helpStyle           tcell.Style
	titleStyle          tcell.Style
	activeTitleStyle    tcell.Style
	selectedStyle       tcell.Style
	selectedActiveStyle tcell.Style
	selectedCachedStyle tcell.Style
	activeEnvStyle      tcell.Style
	cachedConfigStyle   tcell.Style
	searchHitStyle      tcell.Style
	borderStyle         tcell.Style
	activeBorderStyle   tcell.Style
	buttonStyle         tcell.Style
	buttonFocusStyle    tcell.Style
	scrollTrackStyle    tcell.Style
	scrollThumbStyle    tcell.Style

	currentTheme Theme
)

// fallbackSelectionBg is used when a theme leaves selection_bg unset; a
// selected row still has to read as selected on a default background.
const fallbackSelectionBg = 237

func rebuildStyles(theme Theme) {
	currentTheme = theme

	accent = theme.Accent
	dim = theme.Dim
	background = theme.Background
	textColor = theme.Text

	borderColor = theme.Border
	if borderColor == tcell.ColorDefault {
		borderColor = theme.Dim
	}

	selectionBg = theme.SelectionBg
	if selectionBg == tcell.ColorDefault {
		selectionBg = tcell.PaletteColor(fallbackSelectionBg)
	}

	activeEnv = theme.ActiveEnv
	if activeEnv == tcell.ColorDefault {
		activeEnv = theme.Accent
	}

	base := tcell.StyleDefault.Background(background)

	baseStyle = base.Foreground(theme.Text)
	dimStyle = base.Foreground(theme.Dim)
	dirtyStyle = base.Foreground(theme.Dirty)
	deleteStyle = base.Foreground(theme.Delete)
	errorStyle = base.Foreground(theme.Error)
	statusStyle = base.Foreground(theme.Text)
	helpStyle = base.Foreground(theme.Dim)

	titleStyle = base.Foreground(theme.Title).Bold(true)
	activeTitleStyle = base.Foreground(theme.Accent).Bold(true)

	borderStyle = base.Foreground(borderColor)
	activeBorderStyle = base.Foreground(theme.Accent)

	selectedStyle = tcell.StyleDefault.
		Background(selectionBg).
		Foreground(theme.SelectionFg).
		Bold(true)
	selectedActiveStyle = tcell.StyleDefault.
		Background(selectionBg).
		Foreground(activeEnv).
		Bold(true)
	selectedCachedStyle = tcell.StyleDefault.
		Background(selectionBg).
		Foreground(theme.Accent).
		Bold(true)

	activeEnvStyle = base.Foreground(activeEnv).Bold(true)
	cachedConfigStyle = base.Foreground(theme.Accent).Bold(true)

	hitBg := theme.SearchMatchBg
	if hitBg == tcell.ColorDefault {
		hitBg = theme.Accent
	}
	hitFg := theme.SearchMatchFg
	if hitFg == tcell.ColorDefault {
		hitFg = theme.SelectionFg
	}
	if hitFg == tcell.ColorDefault {
		hitFg = theme.Text
	}
	searchHitStyle = tcell.StyleDefault.Background(hitBg).Foreground(hitFg)

	buttonStyle = base.Foreground(theme.Dim)
	buttonFocusStyle = tcell.StyleDefault.
		Background(selectionBg).
		Foreground(theme.SelectionFg).
		Bold(true)

	scrollTrackStyle = base.Foreground(theme.Dim)
	scrollThumbStyle = base.Foreground(theme.Accent)

	applyTviewStyles(theme)
}

// applyTviewStyles points tview's own chrome (box borders, titles, default
// backgrounds) at the active theme so built-in widgets match the panes we draw
// ourselves.
func applyTviewStyles(theme Theme) {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    theme.Background,
		ContrastBackgroundColor:     selectionBg,
		MoreContrastBackgroundColor: selectionBg,
		BorderColor:                 borderColor,
		TitleColor:                  theme.Title,
		GraphicsColor:               borderColor,
		PrimaryTextColor:            theme.Text,
		SecondaryTextColor:          theme.Accent,
		TertiaryTextColor:           theme.Dim,
		InverseTextColor:            theme.SelectionFg,
		ContrastSecondaryTextColor:  theme.Dim,
	}
}
