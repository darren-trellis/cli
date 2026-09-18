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
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Drawing helpers. Everything the panes render goes through these so that
// truncation, padding and wide-rune handling stay in one place.

// textWidth is the number of terminal columns s occupies.
func textWidth(s string) int { return runewidth.StringWidth(s) }

// drawText writes s at (x, y), clipped to maxW columns, and returns the number
// of columns actually written. A rune that would straddle the right edge is
// dropped rather than split.
func drawText(sc tcell.Screen, x, y, maxW int, s string, style tcell.Style) int {
	if maxW <= 0 {
		return 0
	}
	col := 0
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if w == 0 {
			// Combining mark: attach it to the previous cell.
			continue
		}
		if col+w > maxW {
			break
		}
		sc.SetContent(x+col, y, r, nil, style)
		for i := 1; i < w; i++ {
			sc.SetContent(x+col+i, y, ' ', nil, style)
		}
		col += w
	}
	return col
}

// fillRow paints w columns of background at (x, y).
func fillRow(sc tcell.Screen, x, y, w int, style tcell.Style) {
	for i := 0; i < w; i++ {
		sc.SetContent(x+i, y, ' ', nil, style)
	}
}

// fillRect paints a w×h block of background at (x, y).
func fillRect(sc tcell.Screen, x, y, w, h int, style tcell.Style) {
	for row := 0; row < h; row++ {
		fillRow(sc, x, y+row, w, style)
	}
}

// drawCell draws s into exactly w columns, truncating with an ellipsis and
// padding the remainder so the whole cell carries the style's background.
func drawCell(sc tcell.Screen, x, y, w int, s string, style tcell.Style) {
	if w <= 0 {
		return
	}
	fillRow(sc, x, y, w, style)
	drawText(sc, x, y, w, truncate(s, w), style)
}

// drawHighlighted draws s into w columns with every match of re painted in
// hitStyle. offset shifts the regex's view of the string, which is how a
// truncated cell still highlights the right runes.
func drawHighlighted(sc tcell.Screen, x, y, w int, s string, base, hit tcell.Style, re *regexp.Regexp) {
	if w <= 0 {
		return
	}
	fillRow(sc, x, y, w, base)
	s = truncate(s, w)
	if re == nil {
		drawText(sc, x, y, w, s, base)
		return
	}

	spans := matchRuneSpans(s, re)
	if len(spans) == 0 {
		drawText(sc, x, y, w, s, base)
		return
	}

	col := 0
	for i, r := range []rune(s) {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			continue
		}
		if col+rw > w {
			break
		}
		style := base
		if inSpans(spans, i) {
			style = hit
		}
		sc.SetContent(x+col, y, r, nil, style)
		for j := 1; j < rw; j++ {
			sc.SetContent(x+col+j, y, ' ', nil, style)
		}
		col += rw
	}
}

type runeSpan struct{ start, end int } // rune indexes, end exclusive

// matchRuneSpans converts regexp byte offsets into rune indexes.
func matchRuneSpans(s string, re *regexp.Regexp) []runeSpan {
	if re == nil {
		return nil
	}
	locs := re.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return nil
	}
	// Map byte offset -> rune index once, then translate each match.
	byteToRune := make(map[int]int, len(s)+1)
	ri := 0
	for bi := range s {
		byteToRune[bi] = ri
		ri++
	}
	byteToRune[len(s)] = ri

	spans := make([]runeSpan, 0, len(locs))
	for _, loc := range locs {
		if loc[0] == loc[1] {
			continue
		}
		start, ok1 := byteToRune[loc[0]]
		end, ok2 := byteToRune[loc[1]]
		if !ok1 || !ok2 {
			continue
		}
		spans = append(spans, runeSpan{start, end})
	}
	return spans
}

func inSpans(spans []runeSpan, i int) bool {
	for _, sp := range spans {
		if i >= sp.start && i < sp.end {
			return true
		}
	}
	return false
}

// spansForNeedle finds re's matches inside needle and maps them onto display,
// which is the same text with decoration (tree glyphs, markers) prepended.
// Returns nil when the needle is not present in the display string.
func spansForNeedle(display, needle string, re *regexp.Regexp) []runeSpan {
	if re == nil || needle == "" {
		return nil
	}
	byteIdx := strings.Index(display, needle)
	if byteIdx < 0 {
		return nil
	}
	prefixRunes := len([]rune(display[:byteIdx]))
	spans := matchRuneSpans(needle, re)
	for i := range spans {
		spans[i].start += prefixRunes
		spans[i].end += prefixRunes
	}
	return spans
}

// drawSpans draws s into w columns, painting the given rune spans in hitStyle.
func drawSpans(sc tcell.Screen, x, y, w int, s string, base, hit tcell.Style, spans []runeSpan) {
	if w <= 0 {
		return
	}
	fillRow(sc, x, y, w, base)
	col := 0
	for i, r := range []rune(s) {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			continue
		}
		if col+rw > w {
			break
		}
		style := base
		if inSpans(spans, i) {
			style = hit
		}
		sc.SetContent(x+col, y, r, nil, style)
		for j := 1; j < rw; j++ {
			sc.SetContent(x+col+j, y, ' ', nil, style)
		}
		col += rw
	}
}

// drawVerticalScrollbar paints a one-column scrollbar at x spanning h rows.
func drawVerticalScrollbar(sc tcell.Screen, x, y, h, total, offset, visible int) {
	if h <= 0 || visible <= 0 || total <= visible {
		return
	}
	thumbH := max(1, h*visible/total)
	maxOffset := max(1, total-visible)
	travel := max(0, h-thumbH)
	thumbStart := travel * offset / maxOffset
	if thumbStart+thumbH > h {
		thumbStart = h - thumbH
	}
	for i := 0; i < h; i++ {
		if i >= thumbStart && i < thumbStart+thumbH {
			sc.SetContent(x, y+i, '▐', nil, scrollThumbStyle)
		} else {
			sc.SetContent(x, y+i, '│', nil, scrollTrackStyle)
		}
	}
}

// truncate clips s to width columns, using an ellipsis when it does not fit.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if textWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return runewidth.Truncate(s, width, "…")
}

// padRight pads s with spaces to width columns.
func padRight(s string, width int) string {
	w := textWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// padCenter centres s within width columns.
func padCenter(s string, width int) string {
	w := textWidth(s)
	if width <= 0 {
		return s
	}
	if w >= width {
		return truncate(s, width)
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}
