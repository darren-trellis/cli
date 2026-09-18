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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rootConfigs() []models.ConfigInfo {
	return []models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "stg", Environment: "stg", Root: true},
		{Name: "prd", Environment: "prd", Root: true, Locked: true},
		{Name: "dev_personal", Environment: "dev", Root: false},
	}
}

func saveModalModel(config string, configs []models.ConfigInfo) Model {
	m := newModel(models.ScopedOptions{}, configuration.TUISettings{Border: true, Sidebar: true})
	m.width = 80
	m.height = 24
	m.fetching = false
	m.activeProject = "api"
	m.activeConfig = config
	m.projectConfigs["api"] = buildConfigTree(configs)
	m.focus = focusSave
	m.pendingChanges = []models.ChangeRequest{{Name: "FOO"}}
	return m
}

func TestSiblingRootConfigsIgnoresBranchesAndSelf(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	names := configNames(m.siblingRootConfigs())
	assert.Equal(t, []string{"stg", "prd"}, names)
}

func TestSaveModalYOpensPropagateWhenRootHasSiblings(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())

	next, cmd := m.handleSaveKey(runeKey('y'))
	mod := next
	assert.Nil(t, cmd)
	assert.Equal(t, focusPropagate, mod.focus)
	assert.Equal(t, []models.ChangeRequest{{Name: "FOO"}}, mod.pendingChanges)
	assert.False(t, mod.fetching)
	require.Len(t, mod.propagateTargets, 2)
	assert.Equal(t, "stg", mod.propagateTargets[0].name)
	assert.False(t, mod.propagateTargets[0].on)
	assert.Equal(t, "prd", mod.propagateTargets[1].name)
	assert.True(t, mod.propagateTargets[1].locked)
	assert.False(t, mod.propagateTargets[1].on)
}

func TestSaveModalYSavesImmediatelyForBranchConfig(t *testing.T) {
	m := saveModalModel("dev_personal", rootConfigs())

	next, cmd := m.handleSaveKey(runeKey('y'))
	mod := next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingChanges)
	assert.Empty(t, mod.propagateTargets)
}

func TestPropagateSpaceTogglesUnlockedOnly(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next

	next, _ = mod.handlePropagateKey(runeKey(' '))
	mod = next
	assert.True(t, mod.propagateTargets[0].on)
	assert.Equal(t, []string{"stg"}, mod.selectedPropagateConfigs())

	next, _ = mod.handlePropagateKey(namedKey(tcell.KeyDown))
	mod = next
	next, _ = mod.handlePropagateKey(runeKey(' '))
	mod = next
	assert.True(t, mod.propagateTargets[1].on)
	assert.Equal(t, []string{"stg", "prd"}, mod.selectedPropagateConfigs())
}

func TestPropagateKeysViaUpdate(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())

	next, _ := m.Update(runeKey('y'))
	mod := next
	require.Equal(t, focusPropagate, mod.focus)

	next, _ = mod.Update(runeKey(' '))
	mod = next
	assert.True(t, mod.propagateTargets[0].on)
	assert.Equal(t, []string{"stg"}, mod.selectedPropagateConfigs())

	next, _ = mod.Update(runeKey('a'))
	mod = next
	assert.Equal(t, []string{"stg", "prd"}, mod.selectedPropagateConfigs())

	next, _ = mod.Update(runeKey('A'))
	mod = next
	assert.Empty(t, mod.selectedPropagateConfigs())
}

func TestPropagateATogglesAllUnlocked(t *testing.T) {
	m := saveModalModel("dev", []models.ConfigInfo{
		{Name: "dev", Environment: "dev", Root: true},
		{Name: "stg", Environment: "stg", Root: true},
		{Name: "ci", Environment: "ci", Root: true},
		{Name: "prd", Environment: "prd", Root: true, Locked: true},
	})
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next

	next, _ = mod.handlePropagateKey(runeKey('a'))
	mod = next
	assert.Equal(t, []string{"stg", "ci", "prd"}, mod.selectedPropagateConfigs())
	assert.True(t, mod.propagateTargets[2].on)

	next, _ = mod.handlePropagateKey(runeKey('a'))
	mod = next
	assert.Empty(t, mod.selectedPropagateConfigs())
}

func TestPropagateYWithNoneSelectedSavesCurrentOnly(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next
	assert.Empty(t, mod.selectedPropagateConfigs())

	next, cmd := mod.handlePropagateKey(runeKey('y'))
	mod = next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingChanges)
	assert.Empty(t, mod.propagateTargets)
	assert.Empty(t, mod.selectedPropagateConfigs())
}

func TestPropagateYKeepsSelectedSiblings(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next
	mod.togglePropagateAt(0)
	assert.Equal(t, []string{"stg"}, mod.selectedPropagateConfigs())

	next, cmd := mod.handlePropagateKey(runeKey('y'))
	mod = next
	require.NotNil(t, cmd)
	assert.True(t, mod.fetching)
	assert.Empty(t, mod.pendingChanges)
}

func TestPropagateCancelAbandonsSave(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next
	mod.togglePropagateAt(0)

	next, cmd := mod.handlePropagateKey(runeKey('c'))
	mod = next
	assert.Nil(t, cmd)
	assert.Equal(t, focusSecrets, mod.focus)
	assert.Empty(t, mod.pendingChanges)
	assert.Empty(t, mod.propagateTargets)
}

func TestPropagateSkipsDirtySiblings(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	stg := []secretRow{newSecretRow("BAR", "old", "masked")}
	stg[0].value = "new"
	m.putSecretsCache("api", "stg", secretsCacheEntry{secrets: stg})

	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next
	require.True(t, mod.propagateTargets[0].dirty)
	mod.togglePropagateAt(0)
	assert.False(t, mod.propagateTargets[0].on)
	assert.Empty(t, mod.selectedPropagateConfigs())
}

func TestPropagateModalCopy(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next

	view := modalText(mod)
	assert.Contains(t, view, "Apply to other environments")
	assert.Contains(t, view, "stg")
	assert.Contains(t, view, "prd")
	assert.NotContains(t, view, "$prd")
	assert.Contains(t, view, "Apply (y)")
	assert.Contains(t, view, "Cancel (c)")
	assert.NotContains(t, view, "This config only")
	assert.NotContains(t, view, "dev_personal")
	assert.Less(t, strings.Count(view, "\n")+1, 16)
}

func TestFormatSaveStatus(t *testing.T) {
	assert.Equal(t, "Saved", formatSaveStatus(nil, nil))
	assert.Equal(t, "Saved · applied to stg, prd", formatSaveStatus([]string{"stg", "prd"}, nil))
	assert.Equal(t, "Saved · failed on prd", formatSaveStatus(nil, []string{"prd"}))
	assert.Equal(t, "Saved · applied to stg · failed on prd", formatSaveStatus([]string{"stg"}, []string{"prd"}))
}

func TestPropagateClickTogglesRow(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	next, _ := m.handleSaveKey(runeKey('y'))
	mod := next

	spec, ok := mod.currentModalSpec()
	require.True(t, ok)
	hits := spec.geometry(mod.width, mod.height, mod.cfg.Border).rows
	require.Len(t, hits, 2)

	next, cmd := mod.handleMouse(leftClick(hits[0].x, hits[0].y))
	mod = next
	assert.Nil(t, cmd)
	assert.True(t, mod.propagateTargets[0].on)
	assert.Equal(t, []string{"stg"}, mod.selectedPropagateConfigs())
}

func TestSecretsLoadedMsgDropsAppliedCaches(t *testing.T) {
	m := saveModalModel("dev", rootConfigs())
	m.focus = focusSecrets
	m.pendingChanges = nil
	m.rememberLoadedSecrets("api", "stg", []secretRow{newSecretRow("FOO", "1", "masked")})
	m.rememberLoadedSecrets("api", "prd", []secretRow{newSecretRow("FOO", "1", "masked")})

	next, _ := m.Update(secretsLoadedMsg{
		secrets:       []secretRow{newSecretRow("FOO", "2", "masked")},
		activeProject: "api",
		activeConfig:  "dev",
		saved:         true,
		applied:       []string{"stg"},
		failed:        []string{"prd"},
	})
	mod := next
	assert.False(t, mod.configIsCached("api", "stg"))
	assert.True(t, mod.configIsCached("api", "prd"))
	assert.Equal(t, "Saved · applied to stg · failed on prd", mod.statusMsg)
}
