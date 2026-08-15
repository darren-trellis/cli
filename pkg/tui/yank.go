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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

func (m *Model) beginYank() {
	m.pendingYank = true
	m.statusMsg = "y"
	m.errMsg = ""
}

func (m *Model) cancelYank() {
	if m.pendingYank {
		m.pendingYank = false
		if m.statusMsg == "y" {
			m.statusMsg = ""
		}
	}
}

func (m Model) handleYankMotion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord, ok := encodeKey(msg)
	if !ok {
		return m, nil
	}
	m.pendingYank = false
	switch chord {
	case "n":
		return m.yankName()
	case "y":
		return m.yankSecretsFormat("yaml")
	case "j":
		return m.yankSecretsFormat("json")
	case "e":
		return m.yankSecretsFormat("env")
	case "c":
		return m.yankSecret()
	case "esc":
		m.statusMsg = ""
		return m, nil
	default:
		m.statusMsg = ""
		m.errMsg = "yank: n name · y yaml · j json · e env · c cell"
		return m, nil
	}
}

func (m Model) execYank(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.beginYank()
		return m, nil
	}
	switch args[0] {
	case "name":
		return m.yankName()
	case "yaml":
		return m.yankSecretsFormat("yaml")
	case "json":
		return m.yankSecretsFormat("json")
	case "env":
		return m.yankSecretsFormat("env")
	case "cell":
		return m.yankSecret()
	default:
		m.errMsg = "usage: yank name|yaml|json|env|cell"
		return m, nil
	}
}

func (m Model) yankName() (tea.Model, tea.Cmd) {
	var text string
	switch m.focus {
	case focusProjects:
		row, ok := m.currentTreeRow()
		if !ok {
			m.errMsg = "Nothing to copy"
			return m, nil
		}
		if row.kind == treeConfig && row.config != "" {
			text = row.config
		} else if row.project != "" {
			text = row.project
		}
	case focusSecrets:
		idx, ok := m.selectedSecretIndex()
		if !ok {
			m.errMsg = "Nothing to copy"
			return m, nil
		}
		text = m.secrets[idx].name
	default:
		if m.activeConfig != "" {
			text = m.activeConfig
		}
	}
	if text == "" {
		m.errMsg = "Nothing to copy"
		return m, nil
	}
	if err := utils.CopyToClipboard(text); err != nil {
		m.errMsg = "Clipboard unavailable"
		return m, nil
	}
	m.statusMsg = "Copied name"
	m.errMsg = ""
	return m, nil
}

func (m Model) yankSecretsFormat(format string) (tea.Model, tea.Cmd) {
	secrets, skipped := m.secretsMapForCopy()
	if len(secrets) == 0 {
		m.errMsg = "No secrets to copy"
		if skipped > 0 {
			m.errMsg = fmt.Sprintf("No secrets to copy (%d restricted)", skipped)
		}
		m.statusMsg = ""
		return m, nil
	}

	text, err := formatSecretsCopy(secrets, format)
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	if err := utils.CopyToClipboard(text); err != nil {
		m.errMsg = "Clipboard unavailable"
		return m, nil
	}
	m.statusMsg = fmt.Sprintf("Copied %d secrets (%s)", len(secrets), format)
	if skipped > 0 {
		m.statusMsg += fmt.Sprintf(" · skipped %d restricted", skipped)
	}
	m.errMsg = ""
	return m, nil
}

func (m Model) secretsMapForCopy() (map[string]string, int) {
	out := map[string]string{}
	skipped := 0
	for _, s := range m.secrets {
		if s.shouldDelete || strings.TrimSpace(s.name) == "" {
			continue
		}
		if s.originalVisibility == "restricted" && !s.isTouched {
			skipped++
			continue
		}
		out[s.name] = s.value
	}
	return out, skipped
}

func formatSecretsCopy(secrets map[string]string, format string) (string, error) {
	switch format {
	case "env":
		return strings.Join(utils.MapToEnvFormat(secrets, true), "\n") + "\n", nil
	case "json":
		keys := sortedSecretKeys(secrets)
		ordered := make(map[string]string, len(keys))
		for _, k := range keys {
			ordered[k] = secrets[k]
		}
		b, err := json.MarshalIndent(ordered, "", "  ")
		if err != nil {
			return "", fmt.Errorf("json encode failed")
		}
		return string(b) + "\n", nil
	case "yaml":
		keys := sortedSecretKeys(secrets)
		var b strings.Builder
		enc := yaml.NewEncoder(&b)
		enc.SetIndent(2)
		// Encode as a stable ordered map via yaml node for key order
		content := make([]*yaml.Node, 0, len(keys)*2)
		for _, k := range keys {
			content = append(content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: secrets[k]},
			)
		}
		doc := &yaml.Node{Kind: yaml.MappingNode, Content: content}
		if err := enc.Encode(doc); err != nil {
			return "", fmt.Errorf("yaml encode failed")
		}
		_ = enc.Close()
		return b.String(), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func sortedSecretKeys(secrets map[string]string) []string {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
