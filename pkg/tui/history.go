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
	"github.com/charmbracelet/bubbles/textinput"
)

const maxInputHistory = 200

type inputHistory struct {
	items    []string
	draft    string
	matchIdx int
	browsing bool
}

func newInputHistory(items []string) inputHistory {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		cleaned = append(cleaned, item)
	}
	if len(cleaned) > maxInputHistory {
		cleaned = cleaned[len(cleaned)-maxInputHistory:]
	}
	return inputHistory{items: cleaned, matchIdx: -1}
}

func (h inputHistory) Items() []string {
	return append([]string(nil), h.items...)
}

func (h *inputHistory) Reset() {
	h.draft = ""
	h.matchIdx = -1
	h.browsing = false
}

func (h *inputHistory) Push(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		h.Reset()
		return
	}
	for i, item := range h.items {
		if item == s {
			h.items = append(h.items[:i], h.items[i+1:]...)
			break
		}
	}
	h.items = append(h.items, s)
	if len(h.items) > maxInputHistory {
		h.items = h.items[len(h.items)-maxInputHistory:]
	}
	h.Reset()
}

func (h inputHistory) matching(prefix string) []string {
	if prefix == "" {
		return h.items
	}
	var out []string
	for _, item := range h.items {
		if strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return out
}

func (h *inputHistory) Prev(current string) (string, bool) {
	if !h.browsing {
		h.draft = current
		h.browsing = true
		h.matchIdx = len(h.matching(h.draft))
	}
	matches := h.matching(h.draft)
	if len(matches) == 0 || h.matchIdx <= 0 {
		return current, false
	}
	h.matchIdx--
	return matches[h.matchIdx], true
}

func (h *inputHistory) Next(current string) (string, bool) {
	if !h.browsing {
		return current, false
	}
	matches := h.matching(h.draft)
	h.matchIdx++
	if h.matchIdx >= len(matches) {
		draft := h.draft
		h.Reset()
		return draft, true
	}
	if h.matchIdx < 0 {
		return current, false
	}
	return matches[h.matchIdx], true
}

func historyStep(msg string) int {
	switch msg {
	case "up", "ctrl+p":
		return -1
	case "down", "ctrl+n":
		return 1
	default:
		return 0
	}
}

func applyHistoryValue(input *textinput.Model, value string) {
	input.SetValue(value)
	input.CursorEnd()
}

func (m *Model) persistInputHistory() {
	if !m.sessionEnabled {
		return
	}
	configuration.SaveTUIHistory(configuration.TUIHistory{
		Commands: m.commandHistory.Items(),
		Searches: m.searchHistory.Items(),
	})
}

func (m *Model) recordCommandHistory(line string) {
	m.commandHistory.Push(line)
	m.persistInputHistory()
}

func (m *Model) recordSearchHistory(query string) {
	m.searchHistory.Push(query)
	m.persistInputHistory()
}
