package tui

import "github.com/gdamore/tcell/v2"

// Key constructors for tests. They go through encodeKeyEvent so the tests
// exercise the same normalization the running app does.

func runeKey(r rune) keyMsg {
	return encodeKeyEvent(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
}

func namedKey(k tcell.Key) keyMsg {
	return encodeKeyEvent(tcell.NewEventKey(k, 0, tcell.ModNone))
}

// chordKey builds the key event a binding chord refers to, so table-driven
// tests can drive the model with the same strings users put in their config.
func chordKey(chord string) keyMsg {
	switch chord {
	case "space":
		return runeKey(' ')
	case "enter":
		return namedKey(tcell.KeyEnter)
	case "esc":
		return namedKey(tcell.KeyEsc)
	case "tab":
		return namedKey(tcell.KeyTab)
	case "backtab":
		return namedKey(tcell.KeyBacktab)
	case "up":
		return namedKey(tcell.KeyUp)
	case "down":
		return namedKey(tcell.KeyDown)
	case "left":
		return namedKey(tcell.KeyLeft)
	case "right":
		return namedKey(tcell.KeyRight)
	case "home":
		return namedKey(tcell.KeyHome)
	case "end":
		return namedKey(tcell.KeyEnd)
	case "pageup":
		return namedKey(tcell.KeyPgUp)
	case "pagedown":
		return namedKey(tcell.KeyPgDn)
	case "backspace":
		return namedKey(tcell.KeyBackspace2)
	case "delete-key":
		return namedKey(tcell.KeyDelete)
	}
	if len(chord) == 3 && chord[:2] == "C-" {
		return namedKey(tcell.KeyCtrlA + tcell.Key(chord[2]-'a'))
	}
	runes := []rune(chord)
	if len(runes) == 1 {
		return runeKey(runes[0])
	}
	return keyMsg{Chord: chord}
}

func leftClick(x, y int) mouseMsg {
	return mouseMsg{Action: mouseLeftClick, X: x, Y: y}
}
