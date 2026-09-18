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

// Pane geometry. This is pure arithmetic on the model's size and settings, and
// it is the single source of truth shared by drawing, scroll math and mouse
// hit-testing — so a click always lands on the row that was painted there.

type rect struct {
	x, y, w, h int
}

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type layoutRegions struct {
	projects rect
	secrets  rect
	suggest  rect
	status   rect
}

func (m Model) computeLayout() layoutRegions {
	w := max(40, m.width)
	h := max(12, m.height)

	statusH := 1
	suggestH := 0
	if m.focus == focusCommand || (m.focus == focusSecretInsert && m.secretCol == colValue) {
		suggestH = m.completions.DesiredHeight(max(0, h-statusH-4))
	}
	topH := h - statusH - suggestH
	layout := layoutRegions{
		status: rect{0, topH + suggestH, w, statusH},
	}
	if suggestH > 0 {
		layout.suggest = rect{0, topH, w, suggestH}
	}
	if !m.cfg.Sidebar {
		layout.secrets = rect{0, 0, w, topH}
		return layout
	}

	sideW := m.sidebarWidth(w)
	mainW := w - sideW
	if m.cfg.SidebarPosition == "right" {
		layout.secrets = rect{0, 0, mainW, topH}
		layout.projects = rect{mainW, 0, sideW, topH}
	} else {
		layout.projects = rect{0, 0, sideW, topH}
		layout.secrets = rect{sideW, 0, mainW, topH}
	}
	return layout
}

func (m Model) sidebarWidth(totalW int) int {
	if m.cfg.SidebarWidth > 0 {
		w := clamp(m.cfg.SidebarWidth, 12, 60)
		maxW := max(12, totalW/2)
		return min(w, maxW)
	}
	return max(18, totalW/5)
}

const secretColSep = "│"

func (m Model) secretColumnWidths(panelW int) (nameW, valueW int) {
	chrome := m.panelChrome()
	inner := max(10, panelW-chrome)
	avail := max(9, inner-textWidth(secretColSep))
	pct := m.cfg.NameColumnPercent
	if pct < 1 || pct > 99 {
		pct = 40
	}
	nameW = max(12, avail*pct/100)
	valueW = max(8, avail-nameW)
	return nameW, valueW
}
