package tui

import (
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncompleteSecretRef(t *testing.T) {
	from, stage, parts, partial, ok := incompleteSecretRef("{", 1)
	require.True(t, ok)
	assert.Equal(t, 0, stage)
	assert.Empty(t, parts)
	assert.Equal(t, "", partial)
	assert.Equal(t, 1, from)

	from, stage, parts, partial, ok = incompleteSecretRef("${ap", 4)
	require.True(t, ok)
	assert.Equal(t, 0, stage)
	assert.Equal(t, "ap", partial)
	assert.Equal(t, 2, from)

	from, stage, parts, partial, ok = incompleteSecretRef("{api.", 5)
	require.True(t, ok)
	assert.Equal(t, 1, stage)
	assert.Equal(t, []string{"api"}, parts)
	assert.Equal(t, "", partial)
	assert.Equal(t, 5, from)

	from, stage, parts, partial, ok = incompleteSecretRef("{api.dev.TO", 11)
	require.True(t, ok)
	assert.Equal(t, 2, stage)
	assert.Equal(t, []string{"api", "dev"}, parts)
	assert.Equal(t, "TO", partial)

	_, _, _, _, ok = incompleteSecretRef("{api.dev.TOKEN}", 15)
	assert.False(t, ok)

	_, _, _, _, ok = incompleteSecretRef("nope", 4)
	assert.False(t, ok)
}

func TestSecretRefSuggestionsStages(t *testing.T) {
	m := cachedSidebarModel()
	m.projects = []string{"api", "web"}
	m.secretNames[secretsCacheKey("web", "dev")] = []string{"WEB_TOKEN"}

	items := m.secretRefSuggestions("{", 1)
	labels := suggestionLabels(items)
	assert.True(t, labels["api"])
	assert.True(t, labels["web"])
	assert.Equal(t, "api.", firstSuggestionText(items))

	items = m.secretRefSuggestions("{ap", 3)
	labels = suggestionLabels(items)
	assert.True(t, labels["api"])
	assert.False(t, labels["web"])

	items = m.secretRefSuggestions("{api.", 5)
	labels = suggestionLabels(items)
	assert.True(t, labels["dev"])
	assert.True(t, labels["prd"])
	assert.Equal(t, "dev.", firstSuggestionText(items))

	items = m.secretRefSuggestions("{api.prd.", 9)
	labels = suggestionLabels(items)
	assert.True(t, labels["PRD"])
	assert.Equal(t, "PRD}", firstSuggestionText(items))
}

func TestTypingBraceShowsSecretRefSuggestions(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("LINK", "", "masked")}

	next, _ := m.Update(runeKey('i'))
	mod := next
	assert.Equal(t, focusSecretInsert, mod.focus)

	next, _ = mod.Update(runeKey('{'))
	mod = next
	require.NotEmpty(t, mod.completions.Items)
	assert.True(t, suggestionLabels(mod.completions.Items)["api"])
}

func TestTabCompletesSecretRef(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("LINK", "", "masked")}

	next, _ := m.Update(runeKey('i'))
	mod := next
	next, _ = mod.Update(runeKey('{'))
	mod = next
	next, _ = mod.Update(namedKey(tcell.KeyTab))
	mod = next
	assert.Equal(t, "{api.", mod.cellInput.Value())
	assert.Equal(t, "{api.", mod.secrets[0].value)
	require.NotEmpty(t, mod.completions.Items)
	assert.True(t, suggestionLabels(mod.completions.Items)["dev"])

	next, _ = mod.Update(namedKey(tcell.KeyTab))
	mod = next
	assert.Equal(t, "{api.dev.", mod.cellInput.Value())

	next, _ = mod.Update(namedKey(tcell.KeyTab))
	mod = next
	assert.Equal(t, "{api.dev.LINK}", mod.cellInput.Value())
	assert.Empty(t, mod.completions.Items)
	assert.Equal(t, focusSecretInsert, mod.focus)
}

func TestTabWithoutSecretRefStillSwitchesColumns(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secretCol = colName
	m.secrets = []secretRow{newSecretRow("A", "val", "masked")}

	next, _ := m.Update(runeKey('i'))
	mod := next
	next, _ = mod.Update(namedKey(tcell.KeyTab))
	mod = next
	assert.Equal(t, focusSecretInsert, mod.focus)
	assert.Equal(t, colValue, mod.secretCol)
	assert.Equal(t, "val", mod.cellInput.Value())
}

func suggestionLabels(items []Suggestion) map[string]bool {
	out := map[string]bool{}
	for _, item := range items {
		out[item.Label] = true
	}
	return out
}

func firstSuggestionText(items []Suggestion) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Text
}
