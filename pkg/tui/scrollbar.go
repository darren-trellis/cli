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
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderVerticalScrollbar(height, total, offset, visible int) string {
	if height <= 0 || total <= visible || visible <= 0 {
		return ""
	}
	trackStyle := lipgloss.NewStyle().Foreground(dim)
	thumbStyle := lipgloss.NewStyle().Foreground(accent)
	if background != "" {
		trackStyle = trackStyle.Background(background)
		thumbStyle = thumbStyle.Background(background)
	}

	thumbH := max(1, height*visible/total)
	maxOffset := max(1, total-visible)
	travel := max(0, height-thumbH)
	thumbStart := travel * offset / maxOffset
	if thumbStart+thumbH > height {
		thumbStart = height - thumbH
	}

	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i >= thumbStart && i < thumbStart+thumbH {
			b.WriteString(thumbStyle.Render("▐"))
		} else {
			b.WriteString(trackStyle.Render("│"))
		}
	}
	return b.String()
}

func joinWithScrollbar(content string, height, total, offset, visible int, enabled bool) string {
	if !enabled {
		return content
	}
	bar := renderVerticalScrollbar(height, total, offset, visible)
	if bar == "" {
		return content
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, content, bar)
}
