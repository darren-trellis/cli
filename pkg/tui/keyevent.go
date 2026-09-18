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

// encodeKeyEvent normalizes a tcell key event into the chord vocabulary that
// keys.go binds against.
func encodeKeyEvent(ev *tcell.EventKey) keyMsg {
	return keyMsg{
		Chord: keyChord(ev.Key(), ev.Rune(), ev.Modifiers()),
		Key:   ev.Key(),
		Rune:  ev.Rune(),
		Mods:  ev.Modifiers(),
	}
}

// keyChord maps a tcell key to its binding name. Named keys are matched before
// the control-letter range on purpose: terminals deliver Tab, Enter, Backspace
// and Esc as C-i, C-m, C-h and C-[, and users expect the named binding to win.
func keyChord(k tcell.Key, r rune, mods tcell.ModMask) string {
	alt := mods&tcell.ModAlt != 0

	switch k {
	case tcell.KeyEnter:
		return "enter"
	case tcell.KeyEsc:
		return "esc"
	case tcell.KeyTab:
		return "tab"
	case tcell.KeyBacktab:
		return "backtab"
	case tcell.KeyUp:
		return "up"
	case tcell.KeyDown:
		return "down"
	case tcell.KeyLeft:
		return "left"
	case tcell.KeyRight:
		return "right"
	case tcell.KeyHome:
		return "home"
	case tcell.KeyEnd:
		return "end"
	case tcell.KeyPgUp:
		return "pageup"
	case tcell.KeyPgDn:
		return "pagedown"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "backspace"
	case tcell.KeyDelete:
		return "delete-key"
	case tcell.KeyRune:
		switch {
		case r == ' ' && alt:
			return "A-space"
		case r == ' ':
			return "space"
		case alt && unicode.IsLetter(r):
			return "A-" + string(unicode.ToLower(r))
		case alt:
			return "A-" + string(r)
		}
		return string(r)
	}

	if k >= tcell.KeyCtrlA && k <= tcell.KeyCtrlZ {
		return "C-" + string(rune('a'+int(k-tcell.KeyCtrlA)))
	}
	return ""
}

// chord returns the binding name for this key, and whether it has one.
func (k keyMsg) chord() (string, bool) { return k.Chord, k.Chord != "" }

// isTextRune reports whether the event should be inserted into a text field
// rather than treated as a binding.
func (k keyMsg) isTextRune() bool {
	if k.Key != tcell.KeyRune {
		return false
	}
	if k.Mods&(tcell.ModAlt|tcell.ModCtrl|tcell.ModMeta) != 0 {
		return false
	}
	return k.Rune >= ' ' && k.Rune != 0x7f
}
