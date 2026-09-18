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
	"time"

	"github.com/gdamore/tcell/v2"
)

// The TUI keeps the message/command shape it had under Bubble Tea: Update is a
// pure function from (state, message) to (state, side effect), and the tview
// driver in app.go is the only thing that runs those side effects. Keeping the
// state machine pure is what lets the whole suite drive it without a screen.

// Msg is anything Update knows how to handle.
type Msg any

// Cmd is a side effect that produces a Msg. It runs off the UI goroutine.
type Cmd func() Msg

type batchMsg []Cmd

type quitMsg struct{}

// Quit asks the driver to shut the application down.
func Quit() Msg { return quitMsg{} }

// Batch runs several commands concurrently. Nil commands are dropped, and a
// batch with nothing in it is itself nil so callers can pass it around freely.
func Batch(cmds ...Cmd) Cmd {
	var valid []Cmd
	for _, c := range cmds {
		if c != nil {
			valid = append(valid, c)
		}
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return valid[0]
	}
	return func() Msg { return batchMsg(valid) }
}

// Tick produces a Msg after d has elapsed.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	return func() Msg {
		t := time.NewTimer(d)
		defer t.Stop()
		return fn(<-t.C)
	}
}

// windowSizeMsg is delivered whenever the terminal is resized.
type windowSizeMsg struct {
	Width  int
	Height int
}

// keyMsg is a normalized key press. Chord is the binding form used by
// keys.go ("j", "C-f", "backtab"); Rune/Key carry the raw event for the text
// inputs, which need more than the chord.
type keyMsg struct {
	Chord string
	Key   tcell.Key
	Rune  rune
	Mods  tcell.ModMask
}

func (k keyMsg) String() string { return k.Chord }

// mouseAction is the subset of mouse behaviour the panes react to.
type mouseAction int

const (
	mouseLeftClick mouseAction = iota
	mouseWheelUp
	mouseWheelDown
)

// mouseMsg is a normalized mouse event in screen coordinates.
type mouseMsg struct {
	Action mouseAction
	X      int
	Y      int
}
