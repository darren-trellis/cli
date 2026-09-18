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
	"sort"
	"strings"
	"unicode"
)

func (m *Model) refreshSecretRefCompletions() {
	if m.secretCol != colValue {
		m.completions.Clear()
		return
	}
	m.completions.Items = m.secretRefSuggestions(m.cellInput.Value(), m.cellInput.Cursor())
	m.completions.Selected = nil
	m.completions.Browsed = false
	m.completions.Scroll = 0
}

func incompleteSecretRef(buf string, cursor int) (replaceFrom, stage int, parts []string, partial string, ok bool) {
	runes := []rune(buf)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	prefix := runes[:cursor]
	brace := -1
	for i, r := range prefix {
		if r == '{' {
			brace = i
		} else if r == '}' {
			brace = -1
		}
	}
	if brace < 0 {
		return 0, 0, nil, "", false
	}
	rest := string(prefix[brace+1:])
	if strings.Count(rest, ".") > 2 {
		return 0, 0, nil, "", false
	}
	for _, r := range rest {
		if r != '.' && !isSecretRefIdentRune(r) {
			return 0, 0, nil, "", false
		}
	}
	segs := strings.Split(rest, ".")
	partial = segs[len(segs)-1]
	parts = segs[:len(segs)-1]
	replaceFrom = brace + 1 + len([]rune(rest)) - len([]rune(partial))
	return replaceFrom, len(parts), parts, partial, true
}

func isSecretRefIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func (m Model) secretRefSuggestions(buf string, cursor int) []Suggestion {
	from, stage, parts, partial, ok := incompleteSecretRef(buf, cursor)
	if !ok {
		return nil
	}
	var labels []string
	help, suffix := "", "."
	prefer := ""
	switch stage {
	case 0:
		labels = m.refProjectNames()
		help = "project"
		prefer = m.activeProject
	case 1:
		labels = m.refConfigNames(parts[0])
		help = "config"
		if strings.EqualFold(parts[0], m.activeProject) {
			prefer = m.activeConfig
		}
	case 2:
		labels = m.namesFor(parts[0], parts[1])
		help = "secret"
		suffix = "}"
	default:
		return nil
	}
	labels = filterRefLabels(labels, partial, prefer)
	items := make([]Suggestion, 0, len(labels))
	for _, label := range labels {
		items = append(items, Suggestion{
			Text:        label + suffix,
			Label:       label,
			Help:        help,
			ReplaceFrom: from,
		})
	}
	return items
}

func filterRefLabels(labels []string, partial, prefer string) []string {
	partialL := strings.ToLower(partial)
	var matched []string
	seen := map[string]bool{}
	for _, label := range labels {
		if label == "" || seen[label] {
			continue
		}
		if partialL != "" && !strings.HasPrefix(strings.ToLower(label), partialL) {
			continue
		}
		seen[label] = true
		matched = append(matched, label)
	}
	if len(matched) == 0 && partialL != "" {
		seen = map[string]bool{}
		for _, label := range labels {
			if label == "" || seen[label] {
				continue
			}
			if !strings.Contains(strings.ToLower(label), partialL) {
				continue
			}
			seen[label] = true
			matched = append(matched, label)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return strings.ToLower(matched[i]) < strings.ToLower(matched[j])
	})
	if prefer == "" {
		return matched
	}
	var first, rest []string
	for _, label := range matched {
		if strings.EqualFold(label, prefer) {
			first = append(first, label)
		} else {
			rest = append(rest, label)
		}
	}
	return append(first, rest...)
}

func (m Model) refProjectNames() []string {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, p := range m.projects {
		add(p)
	}
	for key := range m.secretNames {
		project, _, ok := splitSecretsCacheKey(key)
		if ok {
			add(project)
		}
	}
	return names
}

func (m Model) refConfigNames(project string) []string {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, c := range m.projectConfigs[project] {
		add(c.name)
	}
	for key := range m.secretNames {
		p, cfg, ok := splitSecretsCacheKey(key)
		if ok && p == project {
			add(cfg)
		}
	}
	return names
}
