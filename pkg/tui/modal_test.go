package tui

import (
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModalButtonsFillWidth(t *testing.T) {
	buttons := []modalButton{
		{Label: "Save", Key: "y"},
		{Label: "Cancel", Key: "n"},
	}
	rects := buttonRects(buttons, 0, 5, 40)
	require.Len(t, rects, 2)

	total := 0
	for _, r := range rects {
		total += r.w
	}
	// Buttons plus the single-column gap between them span the full width.
	assert.Equal(t, 40, total+len(rects)-1)
	assert.GreaterOrEqual(t, rects[0].w, textWidth("Save (y)"))
	assert.GreaterOrEqual(t, rects[1].w, textWidth("Cancel (n)"))
}

func TestModalButtonsStayOnOneLine(t *testing.T) {
	buttons := []modalButton{
		{Label: "Apply", Key: "y"},
		{Label: "Cancel", Key: "c"},
	}
	for _, w := range []int{28, 40, 56} {
		rects := buttonRects(buttons, 0, 3, w)
		require.Len(t, rects, 2, "width %d", w)
		for _, r := range rects {
			assert.Equal(t, 3, r.y, "width %d should stay on one row", w)
		}
		assert.LessOrEqual(t, rects[1].x+rects[1].w, w, "width %d overflowed", w)
		assert.Less(t, rects[0].x, rects[1].x, "width %d order", w)
	}
}

func TestButtonRectsAreCenteredAndDisjoint(t *testing.T) {
	buttons := []modalButton{
		{Label: "Save", Key: "s"},
		{Label: "Discard", Key: "d"},
		{Label: "Cancel", Key: "c"},
	}
	rects := buttonRects(buttons, 3, 5, 40)
	require.Len(t, rects, 3)
	assert.Equal(t, 5, rects[0].y)
	assert.True(t, rects[0].x < rects[1].x)
	assert.True(t, rects[1].x < rects[2].x)
	assert.True(t, rects[1].contains(rects[1].x, 5))
	assert.False(t, rects[0].contains(rects[1].x, 5))
	assert.GreaterOrEqual(t, rects[0].x, 3, "buttons stay inside the modal")
}

func TestModalGeometryKeepsButtonsInsideBorder(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.focus = focusSwitchConfirm
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}

	spec, ok := m.currentModalSpec()
	require.True(t, ok)
	g := spec.geometry(m.width, m.height, true)

	require.NotEmpty(t, g.buttons)
	for i, r := range g.buttons {
		assert.GreaterOrEqual(t, r.x, g.outer.x+1, "button %d past left border", i)
		assert.LessOrEqual(t, r.x+r.w, g.outer.x+g.outer.w-1, "button %d past right border", i)
		assert.Less(t, r.y, g.outer.y+g.outer.h-1, "button %d on the bottom border", i)
	}
}

func TestPropagateModalRowRectsMatchTargets(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.focus = focusPropagate
	m.propagateTargets = []propagateTarget{{name: "stg"}, {name: "prd"}}

	spec, ok := m.currentModalSpec()
	require.True(t, ok)
	g := spec.geometry(m.width, m.height, true)

	require.Len(t, g.rows, 2)
	assert.NotEqual(t, g.rows[0].y, g.rows[1].y)
	for i, r := range g.rows {
		assert.True(t, r.contains(r.x, r.y), "row %d should be hit-testable", i)
	}
}
