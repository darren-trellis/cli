package tui

import (
	"strings"
	"testing"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderFixture(cfg configuration.TUISettings) Model {
	m := newModel(models.ScopedOptions{}, cfg)
	m.fetching = false
	m.width = 90
	m.height = 20
	m.projects = []string{"api", "web"}
	m.activeProject = "api"
	m.activeConfig = "dev"
	m.rebuildTree()
	m.secrets = []secretRow{
		newSecretRow("ALPHA_KEY", "one", "masked"),
		newSecretRow("BETA_KEY", "two", "masked"),
	}
	return m
}

func TestRendersWithBorders(t *testing.T) {
	out := renderModel(t, renderFixture(configuration.TUISettings{Border: true, Sidebar: true}))
	assert.Contains(t, out, "Projects (2)")
	assert.Contains(t, out, "Secrets (2)")
	assert.Contains(t, out, "ALPHA_KEY")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "╭", "bordered panes draw a rounded frame")
}

// Borderless mode uses a plain title row, so panelChrome differs by one; this
// catches the two paths drifting apart.
func TestRendersWithoutBorders(t *testing.T) {
	out := renderModel(t, renderFixture(configuration.TUISettings{Border: false, Sidebar: true}))
	assert.Contains(t, out, "Projects (2)")
	assert.Contains(t, out, "ALPHA_KEY")
	assert.NotContains(t, out, "╭")
}

func TestSidebarPositionRightRenders(t *testing.T) {
	cfg := configuration.TUISettings{Border: true, Sidebar: true, SidebarPosition: "right"}
	m := renderFixture(cfg)
	layout := m.computeLayout()
	require.Greater(t, layout.projects.x, layout.secrets.x)

	out := renderRegion(t, m, layout.projects)
	assert.Contains(t, out, "Projects (2)")
	assert.NotContains(t, out, "ALPHA_KEY")
}

func TestSidebarHiddenGivesSecretsFullWidth(t *testing.T) {
	m := renderFixture(configuration.TUISettings{Border: true, Sidebar: false})
	assert.Equal(t, 0, m.computeLayout().projects.w)

	out := renderModel(t, m)
	assert.NotContains(t, out, "Projects (")
	assert.Contains(t, out, "ALPHA_KEY")
}

// Every theme must survive a full draw: a colour the drawing code cannot use
// would otherwise only show up at runtime.
func TestAllThemesDrawWithoutPanic(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, applyTheme(defaultThemeName)) })

	m := renderFixture(configuration.TUISettings{Border: true, Sidebar: true})
	for _, name := range themeNames() {
		require.NoError(t, applyTheme(name), name)
		out := renderModel(t, m)
		assert.Contains(t, out, "ALPHA_KEY", "theme %s", name)
	}
}

// A light theme has to paint an explicit background, otherwise dark terminals
// show through behind the panes.
func TestLightThemePaintsBackground(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, applyTheme(defaultThemeName)) })
	require.NoError(t, applyTheme("rose-pine-dawn"))

	m := renderFixture(configuration.TUISettings{Border: true, Sidebar: true})
	sc := drawModel(t, m)
	cells, w, _ := sc.GetContents()

	_, bg, _ := cells[5*w+5].Style.Decompose()
	assert.Equal(t, tcell.GetColor("#faf4ed"), bg)
}

func TestGlobalSearchRenderHidesNonMatches(t *testing.T) {
	m := renderFixture(configuration.TUISettings{Border: true, Sidebar: true})
	m.rememberLoadedSecrets("api", "dev", m.secrets)
	m.putSecretsCache("web", "dev", secretsCacheEntry{
		secrets: []secretRow{newSecretRow("WEB_ONLY", "z", "masked")},
	})
	m.projectConfigs["web"] = []configRow{{name: "dev"}}
	m.rebuildTree()

	m.beginGlobalSearch()
	require.NoError(t, m.applySearch("ALPHA", false))

	out := renderModel(t, m)
	assert.Contains(t, out, "ALPHA_KEY")
	assert.NotContains(t, out, "BETA_KEY")
	assert.NotContains(t, out, "web")
	assert.Contains(t, out, "Projects (1)")
	assert.Contains(t, out, "Secrets (1)")
}

func TestStatusBarShowsFilterAndSearchLabels(t *testing.T) {
	m := renderFixture(configuration.TUISettings{Border: true, Sidebar: true})
	m.filter = "ALP"
	m.globalFilter = "KEY"

	layout := m.computeLayout()
	status := renderRegion(t, m, layout.status)
	assert.True(t, strings.Contains(status, "f ALP"), "status: %q", status)
	assert.True(t, strings.Contains(status, "F KEY"), "status: %q", status)
}
