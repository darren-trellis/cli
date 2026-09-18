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

import "regexp"

var secretRefRe = regexp.MustCompile(`\$?\{([A-Za-z0-9][A-Za-z0-9_-]*)\.([A-Za-z0-9][A-Za-z0-9_-]*)\.([A-Za-z0-9][A-Za-z0-9_]*)\}`)

type secretRef struct {
	project string
	config  string
	name    string
}

func firstSecretRef(s string) (secretRef, bool) {
	m := secretRefRe.FindStringSubmatch(s)
	if len(m) != 4 {
		return secretRef{}, false
	}
	return secretRef{project: m[1], config: m[2], name: m[3]}, true
}

func (m Model) openSecretLink() (Model, Cmd) {
	if m.focus != focusSecrets {
		return m, nil
	}
	ref, ok := m.selectedSecretRef()
	if !ok {
		m.enterInsert()
		return m, nil
	}
	if idx, selected := m.selectedSecretIndex(); selected &&
		ref.project == m.activeProject && ref.config == m.activeConfig && m.secrets[idx].name == ref.name {
		m.enterInsert()
		return m, nil
	}
	m.errMsg = ""
	cmd := m.jumpToSecretRef(ref)
	return m, cmd
}

func (m Model) selectedSecretRef() (secretRef, bool) {
	if m.secretCol != colValue {
		return secretRef{}, false
	}
	idx, ok := m.selectedSecretIndex()
	if !ok {
		return secretRef{}, false
	}
	s := m.secrets[idx]
	if s.originalVisibility == "restricted" && !s.isTouched {
		return secretRef{}, false
	}
	return firstSecretRef(s.value)
}

func (m *Model) jumpToSecretRef(ref secretRef) Cmd {
	if ref.project == m.activeProject && ref.config == m.activeConfig {
		if m.selectSecretByName(ref.name) {
			m.secretCol = colValue
			m.setFocus(focusSecrets)
			m.statusMsg = ""
			return nil
		}
		m.statusMsg = "Secret " + ref.name + " not found"
		return nil
	}

	if m.configIsCached(ref.project, ref.config) {
		m.stashCurrentSecrets()
		if m.applyCachedSecrets(ref.project, ref.config) {
			if m.selectSecretByName(ref.name) {
				m.secretCol = colValue
				m.statusMsg = ""
			} else {
				m.statusMsg = "Secret " + ref.name + " not found"
			}
			m.setFocus(focusSecrets)
			m.expanded[ref.project] = true
			m.rebuildTree()
			m.treeIdx = findTreeIndex(m.tree, treeConfig, ref.project, ref.config)
		}
		return nil
	}

	m.pendingSearchName = ref.name
	m.fetching = true
	m.statusMsg = ""
	m.errMsg = ""
	m.stashCurrentSecrets()
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	m.expanded[ref.project] = true
	if _, ok := m.projectConfigs[ref.project]; !ok {
		return Batch(m.spinner.Tick, selectProjectCmd(m.opts, ref.project, ref.config))
	}
	return Batch(m.spinner.Tick, selectConfigCmd(m.opts, ref.project, ref.config))
}

func (m *Model) applyPendingSecretName() {
	if m.pendingSearchName == "" {
		return
	}
	name := m.pendingSearchName
	m.pendingSearchName = ""
	if m.selectSecretByName(name) {
		m.secretCol = colValue
		m.setFocus(focusSecrets)
		if m.searchGlobal {
			m.syncLocalMatchesFromGlobal()
		}
		return
	}
	m.statusMsg = "Secret " + name + " not found"
}
