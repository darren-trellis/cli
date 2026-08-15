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
	"strconv"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

func (m *Model) beginYank() {
	m.pendingYank = true
	m.yankFormat = "yaml"
	m.yankFormatLocked = false
	m.statusMsg = m.yankPrompt()
	m.errMsg = ""
}

func (m *Model) cancelYank() {
	if !m.pendingYank {
		return
	}
	m.clearYankOp()
	m.statusMsg = ""
}

func (m *Model) clearYankOp() {
	m.pendingYank = false
	m.yankFormat = ""
	m.yankFormatLocked = false
}

func (m Model) yankPrompt() string {
	s := "y"
	if m.yankFormatLocked {
		switch m.yankFormat {
		case "json":
			s += "j"
		case "env":
			s += "e"
		}
	}
	if m.motionCount > 0 {
		s += strconv.Itoa(m.motionCount)
	}
	return s
}

func (m Model) handleYankMotion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusSecrets {
		return m.handleSecretsYankMotion(msg)
	}
	return m.handleProjectsYankMotion(msg)
}

func (m Model) handleProjectsYankMotion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord, ok := encodeKey(msg)
	if !ok {
		return m, nil
	}
	m.clearYankOp()
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
		m.errMsg = "yank: n name · y yaml · j json · e env"
		return m, nil
	}
}

func (m Model) handleSecretsYankMotion(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord, ok := encodeKey(msg)
	if !ok {
		return m, nil
	}
	if d, isDigit := countDigit(chord); isDigit {
		if d == 0 && m.motionCount == 0 {
			return m, nil
		}
		m.motionCount = m.motionCount*10 + d
		if m.motionCount > 9999 {
			m.motionCount = 9999
		}
		m.statusMsg = m.yankPrompt()
		return m, nil
	}
	if !m.yankFormatLocked && m.motionCount == 0 {
		switch chord {
		case "j":
			m.yankFormat = "json"
			m.yankFormatLocked = true
			m.statusMsg = m.yankPrompt()
			return m, nil
		case "e":
			m.yankFormat = "env"
			m.yankFormatLocked = true
			m.statusMsg = m.yankPrompt()
			return m, nil
		}
	}
	switch chord {
	case "y", "j", "down":
		return m.finishSecretYankRange(m.takeMotionCount(), 1)
	case "k", "up":
		return m.finishSecretYankRange(m.takeMotionCount(), -1)
	case "n":
		m.clearYankOp()
		m.motionCount = 0
		return m.yankSecret()
	case "G", "end":
		return m.finishSecretYankToIndex(m.takeMotionCountExplicit())
	case "g", "home":
		m.motionCount = 0
		return m.finishSecretYankIndexRange(m.secretIdx, 0)
	case "pagedown":
		return m.finishSecretYankRange(m.pageSize()*m.takeMotionCount(), 1)
	case "pageup":
		return m.finishSecretYankRange(m.pageSize()*m.takeMotionCount(), -1)
	case "esc":
		m.cancelYank()
		return m, nil
	default:
		m.cancelYank()
		m.errMsg = "yank: y line · n cell · [j|e] format · count+motion"
		return m, nil
	}
}

func (m Model) execYank(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		if m.focus != focusProjects && m.focus != focusSecrets {
			m.errMsg = "yank operator is only available in Projects or Secrets"
			return m, nil
		}
		m.beginYank()
		return m, nil
	}
	switch args[0] {
	case "name":
		return m.yankName()
	case "yaml", "json", "env":
		return m.yankSecretsFormat(args[0])
	case "cell":
		return m.yankSecret()
	default:
		m.errMsg = "usage: yank name|yaml|json|env|cell"
		return m, nil
	}
}

func (m Model) yankName() (tea.Model, tea.Cmd) {
	var text string
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

func (m Model) finishSecretYankRange(count, dir int) (tea.Model, tea.Cmd) {
	idxs := m.filteredIndexes()
	lo, hi := secretYankWindow(m.secretIdx, count, dir, len(idxs))
	return m.finishSecretYankIndexRange(lo, hi)
}

func (m Model) finishSecretYankToIndex(n int, explicit bool) (tea.Model, tea.Cmd) {
	idxs := m.filteredIndexes()
	end := len(idxs) - 1
	if explicit {
		end = n - 1
	}
	return m.finishSecretYankIndexRange(m.secretIdx, end)
}

func (m Model) finishSecretYankIndexRange(from, to int) (tea.Model, tea.Cmd) {
	format := m.yankFormat
	if format == "" {
		format = "yaml"
	}
	m.clearYankOp()
	m.motionCount = 0

	idxs := m.filteredIndexes()
	if len(idxs) == 0 {
		m.errMsg = "No secrets to copy"
		m.statusMsg = ""
		return m, nil
	}
	lo, hi := from, to
	if lo > hi {
		lo, hi = hi, lo
	}
	lo = clamp(lo, 0, len(idxs)-1)
	hi = clamp(hi, 0, len(idxs)-1)

	var rows []secretRow
	skipped := 0
	for i := lo; i <= hi; i++ {
		s := m.secrets[idxs[i]]
		if !secretRowCopyable(s) {
			if s.originalVisibility == "restricted" && !s.isTouched {
				skipped++
			}
			continue
		}
		rows = append(rows, s)
	}
	if len(rows) == 0 {
		m.errMsg = "No secrets to copy"
		if skipped > 0 {
			m.errMsg = fmt.Sprintf("No secrets to copy (%d restricted)", skipped)
		}
		m.statusMsg = ""
		return m, nil
	}

	text, err := formatSecretRows(rows, format)
	if err != nil {
		m.errMsg = err.Error()
		m.statusMsg = ""
		return m, nil
	}
	if err := utils.CopyToClipboard(text); err != nil {
		m.errMsg = "Clipboard unavailable"
		m.statusMsg = ""
		return m, nil
	}
	m.statusMsg = fmt.Sprintf("Copied %d secrets (%s)", len(rows), format)
	if skipped > 0 {
		m.statusMsg += fmt.Sprintf(" · skipped %d restricted", skipped)
	}
	m.errMsg = ""
	return m, nil
}

func secretYankWindow(start, count, dir, n int) (int, int) {
	if n <= 0 {
		return 0, -1
	}
	if count < 1 {
		count = 1
	}
	if dir == 0 {
		dir = 1
	}
	start = clamp(start, 0, n-1)
	end := start + dir*(count-1)
	end = clamp(end, 0, n-1)
	if start <= end {
		return start, end
	}
	return end, start
}

func secretRowCopyable(s secretRow) bool {
	if s.shouldDelete || strings.TrimSpace(s.name) == "" {
		return false
	}
	if s.originalVisibility == "restricted" && !s.isTouched {
		return false
	}
	return true
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
	keys := sortedSecretKeys(secrets)
	rows := make([]secretRow, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, secretRow{name: k, value: secrets[k]})
	}
	return formatSecretRows(rows, format)
}

func formatSecretRows(rows []secretRow, format string) (string, error) {
	switch format {
	case "env":
		lines := make([]string, 0, len(rows))
		for _, s := range rows {
			lines = append(lines, envCopyLine(s.name, s.value))
		}
		return strings.Join(lines, "\n") + "\n", nil
	case "json":
		var b strings.Builder
		b.WriteString("{\n")
		for i, s := range rows {
			key, err := json.Marshal(s.name)
			if err != nil {
				return "", fmt.Errorf("json encode failed")
			}
			val, err := json.Marshal(s.value)
			if err != nil {
				return "", fmt.Errorf("json encode failed")
			}
			b.WriteString("  ")
			b.Write(key)
			b.WriteString(": ")
			b.Write(val)
			if i < len(rows)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString("}\n")
		return b.String(), nil
	case "yaml":
		var b strings.Builder
		enc := yaml.NewEncoder(&b)
		enc.SetIndent(2)
		content := make([]*yaml.Node, 0, len(rows)*2)
		for _, s := range rows {
			content = append(content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.name},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s.value},
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

func envCopyLine(key, value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return fmt.Sprintf("%s=\"%s\"", key, value)
}

func sortedSecretKeys(secrets map[string]string) []string {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
