package tui

import (
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wiredApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.projects = []string{"api", "web"}
	m.activeProject = "api"
	m.rebuildTree()
	m.secrets = []secretRow{
		newSecretRow("ALPHA", "1", "masked"),
		newSecretRow("BETA", "2", "masked"),
	}

	a := newApp(m)
	sc := tcell.NewSimulationScreen("UTF-8")
	require.NoError(t, sc.Init())
	t.Cleanup(sc.Fini)
	a.tapp.SetScreen(sc)
	// SetScreen re-initialises the screen, so size it afterwards.
	sc.SetSize(80, 24)
	a.root.SetRect(0, 0, 80, 24)
	return a, sc
}

// TestRootPaneAdoptsScreenSize covers the resize path: the model learns its
// dimensions from the screen on the first draw, with no explicit resize event.
func TestRootPaneAdoptsScreenSize(t *testing.T) {
	a, sc := wiredApp(t)
	assert.Equal(t, 0, a.model.width)

	a.root.Draw(sc)
	sc.Show()

	assert.Equal(t, 80, a.model.width)
	assert.Equal(t, 24, a.model.height)
	assert.Contains(t, screenText(sc), "Projects")
	assert.Contains(t, screenText(sc), "ALPHA")
}

func TestRootPaneInputHandlerDrivesModel(t *testing.T) {
	a, sc := wiredApp(t)
	a.root.Draw(sc)
	require.Equal(t, 0, a.model.secretIdx)

	handler := a.root.InputHandler()
	require.NotNil(t, handler)
	handler(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone), func(tview.Primitive) {})
	assert.Equal(t, 1, a.model.secretIdx, "j should move down the secrets list")

	handler(tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModNone), func(tview.Primitive) {})
	assert.Equal(t, 0, a.model.secretIdx)

	// Tab cycles panes, which is bound through the same chord table.
	handler(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), func(tview.Primitive) {})
	assert.Equal(t, focusProjects, a.model.focus)
}

func TestRootPaneMouseHandlerFocusesPane(t *testing.T) {
	a, sc := wiredApp(t)
	a.root.Draw(sc)
	a.model.focus = focusSecrets

	layout := a.model.computeLayout()
	handler := a.root.MouseHandler()
	require.NotNil(t, handler)

	ev := tcell.NewEventMouse(layout.projects.x+2, layout.projects.y+1, tcell.Button1, tcell.ModNone)
	consumed, _ := handler(tview.MouseLeftClick, ev, func(tview.Primitive) {})
	assert.True(t, consumed)
	assert.Equal(t, focusProjects, a.model.focus, "clicking the sidebar should focus it")

	// An action the pane does not handle is passed on rather than swallowed.
	consumed, _ = handler(tview.MouseMove, ev, func(tview.Primitive) {})
	assert.False(t, consumed)
}

// TestEditingShowsCursor checks the cursor is placed in the edited cell, which
// is what tells the terminal where to blink.
func TestEditingShowsCursor(t *testing.T) {
	a, sc := wiredApp(t)
	a.root.Draw(sc)

	handler := a.root.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyRune, 'i', tcell.ModNone), func(tview.Primitive) {})
	require.Equal(t, focusSecretInsert, a.model.focus)

	a.root.Draw(sc)
	sc.Show()
	x, y, visible := sc.GetCursor()
	assert.True(t, visible, "editing a cell should show the cursor")
	assert.Greater(t, x, 0)
	assert.Greater(t, y, 0)
}

func screenText(sc tcell.SimulationScreen) string {
	cells, w, h := sc.GetContents()
	out := make([]rune, 0, w*h+h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 && c.Runes[0] != 0 {
				out = append(out, c.Runes[0])
			} else {
				out = append(out, ' ')
			}
		}
		out = append(out, '\n')
	}
	return string(out)
}
