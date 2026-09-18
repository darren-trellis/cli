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
	"github.com/DopplerHQ/cli/pkg/models"
)

type secretsCacheEntry struct {
	secrets   []secretRow
	secretIdx int
	secretCol secretCol
	undoStack []int
	filter    string
}

type dirtyGroup struct {
	project string
	config  string
	changes []models.ChangeRequest
}

func secretsCacheKey(project, config string) string {
	return project + "\x00" + config
}

func (m *Model) putSecretsCache(project, config string, entry secretsCacheEntry) {
	if project == "" || config == "" {
		return
	}
	if m.secretsCache == nil {
		m.secretsCache = map[string]secretsCacheEntry{}
	}
	m.secretsCache[secretsCacheKey(project, config)] = entry
}

func (m *Model) rememberLoadedSecrets(project, config string, secrets []secretRow) {
	m.putSecretsCache(project, config, secretsCacheEntry{secrets: secrets})
	if project == "" || config == "" {
		return
	}
	if m.secretNames == nil {
		m.secretNames = map[string][]string{}
	}
	m.secretNames[secretsCacheKey(project, config)] = namesFromSecrets(secrets)
}

func (m Model) visibleFoldedConfigs() map[string]bool {
	out := map[string]bool{}
	for k := range m.secretsCache {
		out[k] = true
	}
	if m.activeProject != "" && m.activeConfig != "" {
		out[secretsCacheKey(m.activeProject, m.activeConfig)] = true
	}
	return out
}

func (m Model) configIsCached(project, config string) bool {
	if project == "" || config == "" {
		return false
	}
	_, ok := m.secretsCache[secretsCacheKey(project, config)]
	return ok
}

func (m Model) configSecrets(project, config string) []secretRow {
	if project == m.activeProject && config == m.activeConfig {
		return m.secrets
	}
	if e, ok := m.secretsCache[secretsCacheKey(project, config)]; ok {
		return e.secrets
	}
	return nil
}

func (m Model) configIsDirty(project, config string) bool {
	return len(collectChanges(m.configSecrets(project, config))) > 0
}

func (m *Model) stashCurrentSecrets() {
	if m.activeProject == "" || m.activeConfig == "" {
		return
	}
	if m.inSecretInsert() {
		m.applyCellToSelection()
	}
	m.putSecretsCache(m.activeProject, m.activeConfig, secretsCacheEntry{
		secrets:   m.secrets,
		secretIdx: m.secretIdx,
		secretCol: m.secretCol,
		undoStack: append([]int(nil), m.undoStack...),
		filter:    m.filter,
	})
}

func (m *Model) applyCachedSecrets(project, config string) bool {
	e, ok := m.secretsCache[secretsCacheKey(project, config)]
	if !ok {
		return false
	}
	m.secrets = e.secrets
	m.secretIdx = e.secretIdx
	m.secretCol = e.secretCol
	m.undoStack = append([]int(nil), e.undoStack...)
	m.filter = e.filter
	if !m.filterGlobal {
		m.filterInput.SetValue(e.filter)
	}
	m.activeProject = project
	m.activeConfig = config
	m.clampSecretIdx()
	m.rebuildTree()
	if m.searchPane == focusSecrets && m.searchRe != nil {
		m.refreshSearchMatches()
	}
	m.persistSession()
	return true
}

func (m *Model) revealHighlightedConfig() {
	row, ok := m.currentTreeRow()
	if !ok || row.kind != treeConfig || row.config == "" {
		return
	}
	if row.project == m.activeProject && row.config == m.activeConfig {
		return
	}
	if !m.configIsCached(row.project, row.config) {
		return
	}
	m.stashCurrentSecrets()
	m.applyCachedSecrets(row.project, row.config)
}

func (m *Model) dropSecretsCache(project, config string) {
	if m.secretsCache == nil || project == "" || config == "" {
		return
	}
	delete(m.secretsCache, secretsCacheKey(project, config))
}

func (m *Model) clearLoadedSecrets() {
	m.secrets = nil
	m.secretIdx = 0
	m.secretCol = colName
	m.undoStack = nil
	m.filter = ""
	m.filterInput.SetValue("")
	m.activeConfig = ""
}

func (m Model) otherCachedConfig(skipProject, skipConfig string) (string, string, bool) {
	idxs := m.cachedTreeIndexes()
	if len(idxs) == 0 {
		return "", "", false
	}
	start := 0
	for i, idx := range idxs {
		if idx >= m.treeIdx {
			start = i
			break
		}
	}
	for i := 0; i < len(idxs); i++ {
		idx := idxs[(start+i)%len(idxs)]
		row := m.tree[idx]
		if row.project == skipProject && row.config == skipConfig {
			continue
		}
		return row.project, row.config, true
	}
	return "", "", false
}

func (m Model) unloadHighlightedConfig() (Model, Cmd) {
	if m.focus != focusProjects {
		m.errMsg = "Unload is only available in Projects"
		return m, nil
	}
	row, ok := m.currentTreeRow()
	if !ok || row.kind != treeConfig || row.config == "" {
		m.errMsg = "Select a loaded config"
		return m, nil
	}
	loaded := m.configIsCached(row.project, row.config) ||
		(row.project == m.activeProject && row.config == m.activeConfig)
	if !loaded {
		m.errMsg = "Config is not loaded"
		return m, nil
	}
	if m.configIsDirty(row.project, row.config) {
		m.errMsg = "Save or discard unsaved changes first"
		return m, nil
	}

	project, config := row.project, row.config
	wasActive := project == m.activeProject && config == m.activeConfig
	m.dropSecretsCache(project, config)
	if wasActive {
		if nextProject, nextConfig, ok := m.otherCachedConfig(project, config); ok {
			m.applyCachedSecrets(nextProject, nextConfig)
		} else {
			m.clearLoadedSecrets()
		}
	}
	m.rebuildTree()
	m.treeIdx = findTreeIndex(m.tree, treeConfig, project, config)
	m.statusMsg = "Unloaded " + project + " / " + config
	m.errMsg = ""
	return m, nil
}

func (m *Model) pruneSecretsCache() {
	if len(m.secretsCache) == 0 {
		return
	}
	valid := map[string]bool{}
	for project, configs := range m.projectConfigs {
		for _, c := range configs {
			valid[secretsCacheKey(project, c.name)] = true
		}
	}
	for k := range m.secretsCache {
		if !valid[k] {
			delete(m.secretsCache, k)
		}
	}
}

func (m Model) cachedTreeIndexes() []int {
	var out []int
	for i, row := range m.tree {
		if row.kind == treeConfig && row.config != "" && m.configIsCached(row.project, row.config) {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) stepCachedConfig(delta int) {
	idxs := m.cachedTreeIndexes()
	if len(idxs) == 0 {
		m.statusMsg = "No cached configs"
		return
	}
	cur := -1
	for i, idx := range idxs {
		row := m.tree[idx]
		if row.project == m.activeProject && row.config == m.activeConfig {
			cur = i
			break
		}
	}
	if cur < 0 {
		cur = 0
		for i, idx := range idxs {
			if idx == m.treeIdx {
				cur = i
				break
			}
		}
	}
	n := len(idxs)
	next := (cur + delta%n + n) % n
	m.treeIdx = idxs[next]
	row := m.tree[m.treeIdx]
	if row.project == m.activeProject && row.config == m.activeConfig {
		return
	}
	m.stashCurrentSecrets()
	m.applyCachedSecrets(row.project, row.config)
	m.statusMsg = row.project + " / " + row.config
}

func (m Model) dirtyGroups() []dirtyGroup {
	seen := map[string]bool{}
	var groups []dirtyGroup
	add := func(project, config string, secrets []secretRow) {
		if project == "" || config == "" {
			return
		}
		key := secretsCacheKey(project, config)
		if seen[key] {
			return
		}
		seen[key] = true
		changes := collectChanges(secrets)
		if len(changes) == 0 {
			return
		}
		groups = append(groups, dirtyGroup{project: project, config: config, changes: changes})
	}
	add(m.activeProject, m.activeConfig, m.secrets)
	for _, row := range m.tree {
		if row.kind != treeConfig {
			continue
		}
		if e, ok := m.secretsCache[secretsCacheKey(row.project, row.config)]; ok {
			add(row.project, row.config, e.secrets)
		}
	}
	for key, e := range m.secretsCache {
		if seen[key] {
			continue
		}
		project, config, ok := splitSecretsCacheKey(key)
		if !ok {
			continue
		}
		add(project, config, e.secrets)
	}
	return groups
}

func splitSecretsCacheKey(key string) (string, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}
