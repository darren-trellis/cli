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

	"github.com/DopplerHQ/cli/pkg/configuration"
)

func sessionEnvKey(project, rootConfig string) string {
	return project + "/" + rootConfig
}

func parseSessionEnvKey(key string) (string, string, bool) {
	project, config, ok := strings.Cut(key, "/")
	if !ok || project == "" || config == "" {
		return "", "", false
	}
	return project, config, true
}

func (m *Model) applySession(session configuration.TUISession) {
	if m.expandedEnvs == nil {
		m.expandedEnvs = map[string]bool{}
	}
	for key, expanded := range session.ExpandedEnvs {
		project, config, ok := parseSessionEnvKey(key)
		if !ok {
			continue
		}
		m.expandedEnvs[envKey(project, config)] = expanded
	}
}

func (m Model) currentSession() configuration.TUISession {
	envs := map[string]bool{}
	for key, expanded := range m.expandedEnvs {
		project, config, ok := strings.Cut(key, "\x00")
		if !ok || project == "" || config == "" {
			continue
		}
		envs[sessionEnvKey(project, config)] = expanded
	}
	return configuration.TUISession{
		Project:      m.activeProject,
		Config:       m.activeConfig,
		ExpandedEnvs: envs,
	}
}

func (m Model) persistSession() {
	if !m.sessionEnabled {
		return
	}
	if m.activeProject == "" || m.activeConfig == "" {
		return
	}
	configuration.SaveTUISession(m.currentSession())
}
