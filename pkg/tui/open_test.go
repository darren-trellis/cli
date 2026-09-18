package tui

import (
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstSecretRef(t *testing.T) {
	got, ok := firstSecretRef("${api.dev.TOKEN}")
	require.True(t, ok)
	assert.Equal(t, secretRef{project: "api", config: "dev", name: "TOKEN"}, got)

	got, ok = firstSecretRef("{backend-ts.dev_personal.API_URL}")
	require.True(t, ok)
	assert.Equal(t, secretRef{project: "backend-ts", config: "dev_personal", name: "API_URL"}, got)

	got, ok = firstSecretRef("prefix ${web.prd.STRIPE_KEY} suffix")
	require.True(t, ok)
	assert.Equal(t, "web", got.project)
	assert.Equal(t, "prd", got.config)
	assert.Equal(t, "STRIPE_KEY", got.name)

	_, ok = firstSecretRef("https://example.com")
	assert.False(t, ok)
	_, ok = firstSecretRef("{not-a-ref}")
	assert.False(t, ok)
	_, ok = firstSecretRef("api.dev.TOKEN")
	assert.False(t, ok)
}

func TestEnterFollowsCachedSecretRef(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("LINK", "${api.prd.PRD}", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)

	next, cmd := m.Update(namedKey(tcell.KeyEnter))
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, next.focus)
	assert.Equal(t, "prd", next.activeConfig)
	assert.Equal(t, colValue, next.secretCol)
	idx, ok := next.selectedSecretIndex()
	require.True(t, ok)
	assert.Equal(t, "PRD", next.secrets[idx].name)
}

func TestEnterFollowsSameConfigSecretRef(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.secretCol = colValue
	m.secrets = []secretRow{
		newSecretRow("API_URL", "https://x", "masked"),
		newSecretRow("REF", "{api.dev.API_URL}", "masked"),
	}
	m.secretIdx = 1

	next, cmd := m.Update(namedKey(tcell.KeyEnter))
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, next.focus)
	idx, ok := next.selectedSecretIndex()
	require.True(t, ok)
	assert.Equal(t, "API_URL", next.secrets[idx].name)
	assert.Equal(t, colValue, next.secretCol)
}

func TestEnterEditsValueWithoutSecretRef(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("TOKEN", "super-secret", "masked")}

	next, _ := m.Update(namedKey(tcell.KeyEnter))
	assert.Equal(t, focusSecretInsert, next.focus)
}

func TestEnterOnNameCellEditsWhenValueIsSecretRef(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secretCol = colName
	m.secrets = []secretRow{newSecretRow("REF", "${api.dev.TOKEN}", "masked")}

	next, _ := m.Update(namedKey(tcell.KeyEnter))
	assert.Equal(t, focusSecretInsert, next.focus)
}

func TestInsertStillEditsSecretRefValue(t *testing.T) {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.fetching = false
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("REF", "${api.prd.PRD}", "masked")}

	next, _ := m.Update(runeKey('i'))
	assert.Equal(t, focusSecretInsert, next.focus)
}

func TestEnterFetchesUncachedSecretRef(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("REF", "${web.dev.TOKEN}", "masked")}
	m.rememberLoadedSecrets("api", "dev", m.secrets)

	next, cmd := m.Update(namedKey(tcell.KeyEnter))
	assert.True(t, next.fetching)
	assert.Equal(t, "TOKEN", next.pendingSearchName)
	assert.NotNil(t, cmd)
}

func TestEnterDoesNotFollowRestrictedSecretRef(t *testing.T) {
	m := cachedSidebarModel()
	m.focus = focusSecrets
	m.secretCol = colValue
	m.secrets = []secretRow{newSecretRow("REF", "${api.prd.PRD}", "restricted")}

	next, cmd := m.Update(namedKey(tcell.KeyEnter))
	assert.Equal(t, focusSecretInsert, next.focus)
	assert.Nil(t, cmd)
	assert.Equal(t, "dev", next.activeConfig)
}
