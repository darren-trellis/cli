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
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceOverlayCompositesOntoBackground(t *testing.T) {
	bg := strings.Join([]string{
		"AAAAAAAAAA",
		"BBBBBBBBBB",
		"CCCCCCCCCC",
		"DDDDDDDDDD",
	}, "\n")
	fg := strings.Join([]string{
		"XX",
		"YY",
	}, "\n")

	got := placeOverlay(2, 1, fg, bg)
	lines := strings.Split(got, "\n")
	assert.Equal(t, "AAAAAAAAAA", lines[0])
	assert.Equal(t, "BBXXBBBBBB", lines[1])
	assert.Equal(t, "CCYYCCCCCC", lines[2])
	assert.Equal(t, "DDDDDDDDDD", lines[3])
}

func TestModalButtonsFillWidth(t *testing.T) {
	m := Model{modalBtnIdx: 0}
	buttons := []modalButton{
		{Label: "Save", Key: "y"},
		{Label: "Cancel", Key: "n"},
	}
	row := m.renderModalButtons(buttons, 40)
	assert.Equal(t, 40, lipgloss.Width(ansi.Strip(row)))
	assert.Contains(t, ansi.Strip(row), "Save (y)")
	assert.Contains(t, ansi.Strip(row), "Cancel (n)")
}

func TestModalButtonsStayOnOneLine(t *testing.T) {
	m := Model{modalBtnIdx: 0}
	buttons := []modalButton{
		{Label: "Apply", Key: "y"},
		{Label: "This config only", Key: "n"},
		{Label: "Back", Key: "esc"},
	}
	for _, w := range []int{28, 40, 56} {
		row := m.renderModalButtons(buttons, w)
		plain := ansi.Strip(row)
		assert.Equal(t, 1, strings.Count(row, "\n")+1, "width %d wrapped", w)
		assert.LessOrEqual(t, lipgloss.Width(plain), w, "width %d overflowed", w)
		assert.Equal(t, w, lipgloss.Width(plain), "width %d should fill", w)
	}
	wide := ansi.Strip(m.renderModalButtons(buttons, 56))
	assert.Contains(t, wide, "Apply (y)")
	assert.Contains(t, wide, "This config only (n)")
	assert.Contains(t, wide, "Back (esc)")
}

func TestButtonHitRectsFindsCenteredRow(t *testing.T) {
	m := Model{modalBtnIdx: 0}
	buttons := []modalButton{
		{Label: "Save", Key: "s"},
		{Label: "Discard", Key: "d"},
		{Label: "Cancel", Key: "c"},
	}
	row := m.renderModalButtons(buttons, 40)
	hits := buttonHitRects(row, buttons, 3, 5)
	require.Len(t, hits, 3)
	assert.Equal(t, 5, hits[0].y)
	assert.True(t, hits[0].x < hits[1].x)
	assert.True(t, hits[1].x < hits[2].x)
	assert.True(t, hits[1].contains(hits[1].x, 5))
	assert.False(t, hits[0].contains(hits[1].x, 5))
}

func TestFillLineBackgroundPadsShortLines(t *testing.T) {
	got := fillLineBackground("ab\ncd", 4)
	lines := strings.Split(got, "\n")
	assert.Equal(t, 4, len([]rune(lines[0])))
	assert.Equal(t, 4, len([]rune(lines[1])))
}
