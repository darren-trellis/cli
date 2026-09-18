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

import "strings"

// textScroll is the scrollable body of the help modal. It replaces
// bubbles/viewport with just the operations the modal actually uses.
type textScroll struct {
	Width  int
	Height int

	lines  []string
	offset int
}

func (v *textScroll) SetContent(s string) {
	v.lines = strings.Split(s, "\n")
	v.clamp()
}

func (v *textScroll) GotoTop() { v.offset = 0 }

func (v *textScroll) LineDown(n int) {
	v.offset += n
	v.clamp()
}

func (v *textScroll) LineUp(n int) {
	v.offset -= n
	v.clamp()
}

func (v *textScroll) ViewDown() { v.LineDown(max(1, v.Height)) }

func (v *textScroll) ViewUp() { v.LineUp(max(1, v.Height)) }

func (v *textScroll) clamp() {
	maxOffset := max(0, len(v.lines)-max(1, v.Height))
	v.offset = clamp(v.offset, 0, maxOffset)
}

// visible returns exactly Height lines, padded with blanks so the modal keeps
// a stable size as the user scrolls.
func (v *textScroll) visible() []string {
	v.clamp()
	h := max(1, v.Height)
	out := make([]string, 0, h)
	for i := v.offset; i < len(v.lines) && len(out) < h; i++ {
		out = append(out, v.lines[i])
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out
}

func (v *textScroll) totalLines() int { return len(v.lines) }

func (v *textScroll) scrollOffset() int { return v.offset }
