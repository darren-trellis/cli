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
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// textField is a single-line editable field. It replaces bubbles/textinput:
// the panes draw it themselves, so it only has to track text, cursor and the
// horizontal scroll needed to keep the cursor on screen.
type textField struct {
	Prompt      string
	Placeholder string
	CharLimit   int
	Width       int

	value   []rune
	cursor  int
	offset  int
	focused bool
}

func newTextField(prompt, placeholder string, charLimit int) textField {
	return textField{Prompt: prompt, Placeholder: placeholder, CharLimit: charLimit}
}

func (t textField) Value() string { return string(t.value) }

func (t *textField) SetValue(s string) {
	t.value = []rune(s)
	if t.CharLimit > 0 && len(t.value) > t.CharLimit {
		t.value = t.value[:t.CharLimit]
	}
	if t.cursor > len(t.value) {
		t.cursor = len(t.value)
	}
}

func (t textField) Cursor() int { return t.cursor }

func (t *textField) SetCursor(i int) {
	t.cursor = clamp(i, 0, len(t.value))
}

func (t *textField) CursorEnd() { t.cursor = len(t.value) }

func (t *textField) Focus() { t.focused = true }

func (t *textField) Blur() { t.focused = false }

func (t textField) Focused() bool { return t.focused }

func (t *textField) Reset() {
	t.value = nil
	t.cursor = 0
	t.offset = 0
}

// Update applies an editing key. It reports whether the key was consumed so
// callers can fall through to their own bindings.
func (t *textField) Update(msg keyMsg) bool {
	switch msg.Chord {
	case "left":
		if t.cursor > 0 {
			t.cursor--
		}
		return true
	case "right":
		if t.cursor < len(t.value) {
			t.cursor++
		}
		return true
	case "home", "C-a":
		t.cursor = 0
		return true
	case "end", "C-e":
		t.cursor = len(t.value)
		return true
	case "backspace":
		if t.cursor > 0 {
			t.value = append(t.value[:t.cursor-1], t.value[t.cursor:]...)
			t.cursor--
		}
		return true
	case "delete-key", "C-d":
		if t.cursor < len(t.value) {
			t.value = append(t.value[:t.cursor], t.value[t.cursor+1:]...)
		}
		return true
	case "C-u":
		t.value = append([]rune{}, t.value[t.cursor:]...)
		t.cursor = 0
		return true
	case "C-k":
		t.value = t.value[:t.cursor]
		return true
	case "C-w":
		t.deleteWordBackward()
		return true
	case "space":
		t.insert(' ')
		return true
	}

	if msg.isTextRune() {
		t.insert(msg.Rune)
		return true
	}
	return false
}

func (t *textField) insert(r rune) {
	if t.CharLimit > 0 && len(t.value) >= t.CharLimit {
		return
	}
	t.value = append(t.value, 0)
	copy(t.value[t.cursor+1:], t.value[t.cursor:])
	t.value[t.cursor] = r
	t.cursor++
}

func (t *textField) deleteWordBackward() {
	i := t.cursor
	for i > 0 && unicode.IsSpace(t.value[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(t.value[i-1]) {
		i--
	}
	t.value = append(t.value[:i], t.value[t.cursor:]...)
	t.cursor = i
}

// window returns the slice of text visible in w columns along with the cursor's
// column within that slice, scrolling horizontally to keep the cursor in view.
func (t *textField) window(w int) (string, int) {
	if w <= 0 {
		return "", 0
	}
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	// Leave one column for the cursor when it sits past the last rune.
	if t.cursor >= t.offset+w {
		t.offset = t.cursor - w + 1
	}
	if t.offset > len(t.value) {
		t.offset = len(t.value)
	}
	if t.offset < 0 {
		t.offset = 0
	}
	end := min(t.offset+w, len(t.value))
	return string(t.value[t.offset:end]), t.cursor - t.offset
}

// draw renders the field at (x, y) and returns the screen column the cursor
// landed on, or -1 when the field is not focused.
func (t *textField) draw(sc tcell.Screen, x, y, w int, style, promptStyle, placeholderStyle tcell.Style) int {
	if w <= 0 {
		return -1
	}
	promptW := drawText(sc, x, y, w, t.Prompt, promptStyle)
	availW := w - promptW
	if availW <= 0 {
		return -1
	}

	fillRow(sc, x+promptW, y, availW, style)

	if len(t.value) == 0 && !t.focused && t.Placeholder != "" {
		drawText(sc, x+promptW, y, availW, t.Placeholder, placeholderStyle)
		return -1
	}

	text, cursorCol := t.window(availW)
	drawText(sc, x+promptW, y, availW, text, style)

	if !t.focused {
		return -1
	}
	return x + promptW + clamp(cursorCol, 0, availW-1)
}
