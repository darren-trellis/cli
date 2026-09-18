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
	"fmt"
	"log"
	"os"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/common"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App drives the model with tview: tview owns the screen, the event loop and
// the terminal lifecycle, while every state transition still goes through
// Model.Update on the UI goroutine.
type App struct {
	tapp  *tview.Application
	root  *rootPane
	model Model
}

func newApp(m Model) *App {
	a := &App{tapp: tview.NewApplication(), model: m}
	a.root = &rootPane{Box: tview.NewBox(), app: a}
	a.tapp.SetRoot(a.root, true).EnableMouse(true).SetFocus(a.root)
	return a
}

func (a *App) Run() error {
	a.exec(a.model.Init())
	return a.tapp.Run()
}

// dispatch applies a message to the model. It must run on the UI goroutine.
func (a *App) dispatch(msg Msg) {
	switch m := msg.(type) {
	case nil:
		return
	case quitMsg:
		a.tapp.Stop()
		return
	case batchMsg:
		for _, cmd := range m {
			a.exec(cmd)
		}
		return
	}

	var cmd Cmd
	a.model, cmd = a.model.Update(msg)
	a.exec(cmd)
}

// exec runs a command off the UI goroutine and feeds its result back in.
func (a *App) exec(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if msg == nil {
			return
		}
		a.tapp.QueueUpdateDraw(func() { a.dispatch(msg) })
	}()
}

// rootPane is the single tview primitive the app draws into. The panes inside
// it are positioned by computeLayout rather than by nested widgets, so drawing
// and mouse hit-testing share one description of the layout.
type rootPane struct {
	*tview.Box
	app *App
}

func (r *rootPane) Draw(screen tcell.Screen) {
	w, h := screen.Size()
	if w != r.app.model.width || h != r.app.model.height {
		r.app.model, _ = r.app.model.Update(windowSizeMsg{Width: w, Height: h})
	}
	if x, y := r.app.model.draw(screen); x >= 0 {
		screen.ShowCursor(x, y)
	} else {
		screen.HideCursor()
	}
}

func (r *rootPane) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return r.WrapInputHandler(func(event *tcell.EventKey, _ func(tview.Primitive)) {
		r.app.dispatch(encodeKeyEvent(event))
	})
}

func (r *rootPane) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return r.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, _ func(tview.Primitive)) (bool, tview.Primitive) {
		var a mouseAction
		switch action {
		case tview.MouseLeftClick:
			a = mouseLeftClick
		case tview.MouseScrollUp:
			a = mouseWheelUp
		case tview.MouseScrollDown:
			a = mouseWheelDown
		default:
			return false, nil
		}
		x, y := event.Position()
		r.app.dispatch(mouseMsg{Action: a, X: x, Y: y})
		return true, nil
	})
}

func Start(opts models.ScopedOptions, cfg configuration.TUISettings) {
	cmn, err := common.NewCommon(opts)
	if err != nil {
		log.Fatal(err)
	}

	if cmn.Opts.EnclaveProject.Value == "" {
		utils.Log("You must run `doppler setup` prior to launching the TUI")
		os.Exit(1)
	}

	if err := applyTheme(cfg.Theme); err != nil {
		utils.Log(err.Error() + "; using " + defaultThemeName)
		cfg.Theme = defaultThemeName
		_ = applyTheme(defaultThemeName)
		configuration.TUISetTheme(defaultThemeName)
	}

	m := newModel(cmn.Opts, cfg)
	m.sessionEnabled = true
	m.loadNamesIndex()
	hist := configuration.LoadTUIHistory()
	m.commandHistory = newInputHistory(hist.Commands)
	m.searchHistory = newInputHistory(hist.Searches)
	if session, ok := configuration.LoadTUISession(); ok {
		m.applySession(session)
	}

	if err := newApp(m).Run(); err != nil {
		fmt.Println("Error running TUI:", err)
		os.Exit(1)
	}
}
