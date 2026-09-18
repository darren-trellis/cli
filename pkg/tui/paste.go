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
	"regexp"
	"strconv"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

var envLineRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

func (m Model) pasteSecrets() (Model, Cmd) {
	if m.focus != focusSecrets && m.focus != focusSecretInsert {
		m.errMsg = "Paste is only available in Secrets"
		return m, nil
	}
	if m.inSecretInsert() {
		m.applyCellToSelection()
		m.setFocus(focusSecrets)
	}

	raw, err := utils.ReadClipboard()
	if err != nil {
		m.errMsg = "Clipboard unavailable"
		return m, nil
	}
	parsed, format, err := parseSecretsClipboard(raw)
	if err != nil {
		m.errMsg = err.Error()
		m.statusMsg = ""
		return m, nil
	}

	added, updated := m.applyPastedSecrets(parsed)
	m.statusMsg = fmt.Sprintf("Pasted %d secrets (%s) · +%d ~%d", added+updated, format, added, updated)
	m.errMsg = ""
	return m, nil
}

func (m *Model) applyPastedSecrets(parsed map[string]string) (added, updated int) {
	byName := map[string]int{}
	for i, s := range m.secrets {
		if s.name == "" {
			continue
		}
		byName[s.name] = i
	}

	for _, name := range sortedSecretKeys(parsed) {
		val := parsed[name]
		if name == "" {
			continue
		}
		if idx, ok := byName[name]; ok {
			s := &m.secrets[idx]
			if s.originalVisibility == "restricted" {
				s.isTouched = true
			}
			s.value = val
			s.shouldDelete = false
			m.noteSecretEdit(idx)
			updated++
			continue
		}
		row := secretRow{name: name, value: val}
		m.secrets = append(m.secrets, row)
		m.noteSecretEdit(len(m.secrets) - 1)
		byName[name] = len(m.secrets) - 1
		added++
	}
	return added, updated
}

func parseSecretsClipboard(text string) (map[string]string, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", fmt.Errorf("clipboard empty")
	}

	if strings.HasPrefix(text, "{") {
		if m, err := parseSecretsJSON(text); err == nil && len(m) > 0 {
			return m, "json", nil
		}
	}

	if looksLikeEnv(text) {
		if m, ok := parseSecretsEnv(text); ok && len(m) > 0 {
			return m, "env", nil
		}
	}

	if m, err := parseSecretsYAML(text); err == nil && len(m) > 0 {
		return m, "yaml", nil
	}

	if m, ok := parseSecretsEnv(text); ok && len(m) > 0 {
		return m, "env", nil
	}

	return nil, "", fmt.Errorf("unrecognized clipboard format (want yaml/json/env)")
}

func parseSecretsJSON(text string) (map[string]string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, err
	}
	return stringifySecretMap(raw), nil
}

func parseSecretsYAML(text string) (map[string]string, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("empty yaml")
	}
	return stringifySecretMap(raw), nil
}

func parseSecretsEnv(text string) (map[string]string, bool) {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := envLineRE.FindStringSubmatch(line)
		if m == nil {
			return nil, false
		}
		out[m[1]] = unquoteEnvValue(m[2])
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func looksLikeEnv(text string) bool {
	kv, other := 0, 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if envLineRE.MatchString(line) {
			kv++
		} else {
			other++
		}
	}
	return kv > 0 && kv >= other
}

func unquoteEnvValue(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			if u, err := strconv.Unquote(v); err == nil {
				return u
			}
			return v[1 : len(v)-1]
		}
	}
	return v
}

func stringifySecretMap(raw map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range raw {
		if k == "" || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		case bool:
			out[k] = strconv.FormatBool(t)
		case float64:
			out[k] = strconv.FormatFloat(t, 'f', -1, 64)
		case int:
			out[k] = strconv.Itoa(t)
		default:
			b, err := json.Marshal(t)
			if err != nil {
				out[k] = fmt.Sprint(t)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}
