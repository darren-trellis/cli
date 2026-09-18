package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/require"
)

// Rendering helpers for tests. Drawing into a simulation screen exercises the
// same code path the real terminal takes, so these assertions cover the actual
// pane output rather than a parallel string renderer.

func drawModel(t *testing.T, m Model) tcell.SimulationScreen {
	t.Helper()
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}
	sc := tcell.NewSimulationScreen("UTF-8")
	require.NoError(t, sc.Init())
	t.Cleanup(sc.Fini)
	sc.SetSize(m.width, m.height)
	m.draw(sc)
	// SimulationScreen only publishes cells to GetContents on Show.
	sc.Show()
	return sc
}

func screenRows(sc tcell.SimulationScreen) []string {
	cells, w, h := sc.GetContents()
	rows := make([]string, 0, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 && c.Runes[0] != 0 {
				b.WriteRune(c.Runes[0])
			} else {
				b.WriteRune(' ')
			}
		}
		rows = append(rows, b.String())
	}
	return rows
}

// renderModel returns everything on screen, with trailing blanks trimmed.
func renderModel(t *testing.T, m Model) string {
	t.Helper()
	rows := screenRows(drawModel(t, m))
	for i, r := range rows {
		rows[i] = strings.TrimRight(r, " ")
	}
	return strings.Join(rows, "\n")
}

// renderRegion returns just the rows and columns covered by r.
func renderRegion(t *testing.T, m Model, r rect) string {
	t.Helper()
	rows := screenRows(drawModel(t, m))
	var out []string
	for y := r.y; y < r.y+r.h && y < len(rows); y++ {
		line := []rune(rows[y])
		lo := min(r.x, len(line))
		hi := min(r.x+r.w, len(line))
		out = append(out, string(line[lo:hi]))
	}
	return strings.Join(out, "\n")
}

// renderSidebar returns the project pane's own text.
func renderSidebar(t *testing.T, m Model) string {
	t.Helper()
	return renderRegion(t, m, m.computeLayout().projects)
}

// modalText renders the current modal's title and body lines without a screen.
func modalText(m Model) string {
	spec, ok := m.currentModalSpec()
	if !ok {
		return ""
	}
	parts := []string{spec.title}
	if spec.scroll != nil {
		parts = append(parts, spec.scroll.visible()...)
	}
	for _, l := range spec.lines {
		parts = append(parts, l.text)
	}
	for _, b := range spec.buttons {
		parts = append(parts, b.text())
	}
	return strings.Join(parts, "\n")
}
