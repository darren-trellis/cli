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
)

type Suggestion struct {
	Text        string
	Label       string
	Help        string
	ReplaceFrom int
}

type CompletionState struct {
	Items     []Suggestion
	Selected  *int
	Browsed   bool
	Scroll    int
	ViewportH int
}

func (c *CompletionState) Clear() {
	c.Items = nil
	c.Selected = nil
	c.Browsed = false
	c.Scroll = 0
	c.ViewportH = 0
}

func (c *CompletionState) SelectedItem() *Suggestion {
	if c.Selected == nil {
		return nil
	}
	i := *c.Selected
	if i < 0 || i >= len(c.Items) {
		return nil
	}
	return &c.Items[i]
}

func (c *CompletionState) DesiredHeight(maxH int) int {
	if len(c.Items) == 0 || maxH < 2 {
		return 0
	}
	n := len(c.Items)
	if n > 8 {
		n = 8
	}
	h := n + 2 // borders
	if h > maxH {
		return maxH
	}
	return h
}

func (c *CompletionState) Step(delta int) {
	if len(c.Items) == 0 {
		c.Selected = nil
		c.Browsed = false
		return
	}
	if !c.Browsed || c.Selected == nil {
		if delta > 0 {
			z := 0
			c.Selected = &z
			c.Browsed = true
			c.centerOnSelection()
		}
		return
	}
	next := *c.Selected + delta
	if next < 0 {
		c.Selected = nil
		c.Browsed = false
		return
	}
	if next >= len(c.Items) {
		next = len(c.Items) - 1
	}
	c.Selected = &next
	c.Browsed = true
	c.centerOnSelection()
}

func (c *CompletionState) SelectNext() {
	if len(c.Items) == 0 {
		return
	}
	if c.Selected == nil {
		z := 0
		c.Selected = &z
	} else {
		n := (*c.Selected + 1) % len(c.Items)
		c.Selected = &n
	}
	c.centerOnSelection()
}

func (c *CompletionState) SelectPrev() {
	if len(c.Items) == 0 {
		return
	}
	if c.Selected == nil || *c.Selected == 0 {
		n := len(c.Items) - 1
		c.Selected = &n
	} else {
		n := *c.Selected - 1
		c.Selected = &n
	}
	c.centerOnSelection()
}

func (c *CompletionState) EnsureVisible() {
	viewport := c.ViewportH
	if viewport < 1 {
		viewport = 1
	}
	if len(c.Items) == 0 {
		c.Scroll = 0
		return
	}
	if c.Selected == nil {
		maxStart := max(0, len(c.Items)-viewport)
		if c.Scroll > maxStart {
			c.Scroll = maxStart
		}
		return
	}
	sel := *c.Selected
	if sel < c.Scroll {
		c.Scroll = sel
	} else if sel >= c.Scroll+viewport {
		c.Scroll = sel + 1 - viewport
	}
	maxStart := max(0, len(c.Items)-viewport)
	if c.Scroll > maxStart {
		c.Scroll = maxStart
	}
}

func (c *CompletionState) centerOnSelection() {
	viewport := c.ViewportH
	if viewport < 1 {
		viewport = 8
	}
	if c.Selected == nil || len(c.Items) == 0 {
		return
	}
	c.Scroll = max(0, *c.Selected-viewport/2)
	maxStart := max(0, len(c.Items)-viewport)
	if c.Scroll > maxStart {
		c.Scroll = maxStart
	}
}

func (m *Model) refreshCompletions() {
	m.completions.Items = suggestionsFor(m.commandInput.Value())
	m.completions.Selected = nil
	m.completions.Browsed = false
	m.completions.Scroll = 0
}

func (m *Model) applySelectedCompletion() {
	sel := m.completions.SelectedItem()
	if sel == nil || sel.Text == "" {
		return
	}
	buf := m.commandInput.Value()
	from := sel.ReplaceFrom
	if from > len(buf) {
		from = len(buf)
	}
	m.commandInput.SetValue(buf[:from] + sel.Text)
	m.commandInput.CursorEnd()
	m.completions.Browsed = false
}

func (m *Model) selectionApplied() bool {
	sel := m.completions.SelectedItem()
	if sel == nil {
		return false
	}
	buf := m.commandInput.Value()
	from := sel.ReplaceFrom
	if from > len(buf) {
		return false
	}
	return buf[from:] == sel.Text
}

func (m *Model) tabComplete(next bool) {
	if len(m.completions.Items) == 0 {
		m.refreshCompletions()
		if len(m.completions.Items) == 0 {
			return
		}
	}
	if m.completions.Browsed {
		m.applySelectedCompletion()
		return
	}
	if next {
		m.completions.SelectNext()
	} else {
		m.completions.SelectPrev()
	}
	m.applySelectedCompletion()
}

func suggestionsFor(buffer string) []Suggestion {
	trimmedStart := len(buffer) - len(strings.TrimLeft(buffer, " \t"))
	body := strings.TrimLeft(buffer, " \t")

	var items []Suggestion
	if body == "" {
		items = commandPrefixSuggestions("", trimmedStart)
	} else {
		items = commandPathSuggestions(buffer, body, trimmedStart)
	}
	sortSuggestions(items)
	return items
}

func hasTrailingWS(s string) bool {
	if s == "" {
		return false
	}
	return s[len(s)-1] == ' ' || s[len(s)-1] == '\t'
}

func commandPathSuggestions(buffer, body string, trimmedStart int) []Suggestion {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return commandPrefixSuggestions("", trimmedStart)
	}

	if !hasTrailingWS(body) {
		partial := fields[len(fields)-1]
		complete := fields[:len(fields)-1]
		if len(complete) == 0 {
			if commandPrefixHasChildren([]string{partial}) {
				return nextTokenSuggestions([]string{partial}, "", len(buffer), true)
			}
			return commandPrefixSuggestions(partial, trimmedStart)
		}
		prefix := append(append([]string{}, complete...), partial)
		if commandPrefixHasChildren(prefix) {
			return nextTokenSuggestions(prefix, "", len(buffer), true)
		}
		return nextTokenSuggestions(complete, partial, len(buffer)-len(partial), false)
	}

	return nextTokenSuggestions(fields, "", len(buffer), false)
}

func commandPrefixHasChildren(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	for _, c := range commandCatalog {
		fields := strings.Fields(c.name)
		if len(fields) <= len(tokens) {
			continue
		}
		if commandPathPrefix(fields, tokens) {
			return true
		}
	}
	return false
}

func commandPathPrefix(fields, tokens []string) bool {
	if len(fields) < len(tokens) {
		return false
	}
	for i, t := range tokens {
		if !strings.EqualFold(fields[i], t) {
			return false
		}
	}
	return true
}

func nextTokenSuggestions(complete []string, partial string, replaceFrom int, leadSpace bool) []Suggestion {
	partialL := strings.ToLower(partial)
	seen := map[string]bool{}
	var items []Suggestion
	for _, c := range commandCatalog {
		fields := strings.Fields(c.name)
		if !commandPathPrefix(fields, complete) || len(fields) <= len(complete) {
			continue
		}
		next := fields[len(complete)]
		if partialL != "" && !strings.HasPrefix(strings.ToLower(next), partialL) {
			continue
		}
		if seen[next] {
			continue
		}
		seen[next] = true
		text := next
		if leadSpace {
			text = " " + next
		}
		items = append(items, Suggestion{
			Text:        text,
			Label:       next,
			Help:        nextTokenHelp(complete, next),
			ReplaceFrom: replaceFrom,
		})
	}
	return items
}

func nextTokenHelp(complete []string, next string) string {
	name := strings.TrimSpace(strings.Join(append(append([]string{}, complete...), next), " "))
	for _, c := range commandCatalog {
		if strings.EqualFold(c.name, name) {
			return c.help
		}
	}
	return name + " commands"
}

func commandPrefixSuggestions(prefix string, replaceFrom int) []Suggestion {
	prefixL := strings.ToLower(prefix)
	seen := map[string]bool{}
	var items []Suggestion

	add := func(name, help string) {
		if seen[name] {
			return
		}
		seen[name] = true
		items = append(items, Suggestion{
			Text:        name,
			Label:       name,
			Help:        help,
			ReplaceFrom: replaceFrom,
		})
	}

	for _, parent := range parentCommandNames() {
		if parent == "command" {
			continue
		}
		parentL := strings.ToLower(parent)
		if prefixL == "" || strings.HasPrefix(parentL, prefixL) {
			add(parent, parentCommandHelp(parent))
		}
	}
	if len(items) == 0 && prefixL != "" {
		for _, parent := range parentCommandNames() {
			if parent == "command" {
				continue
			}
			if strings.Contains(strings.ToLower(parent), prefixL) {
				add(parent, parentCommandHelp(parent))
			}
		}
	}
	return items
}

func parentCommandNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, c := range commandCatalog {
		fields := strings.Fields(c.name)
		if len(fields) == 0 || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		names = append(names, fields[0])
	}
	return names
}

func parentCommandHelp(parent string) string {
	for _, c := range commandCatalog {
		if strings.EqualFold(c.name, parent) {
			return c.help
		}
	}
	return parent + " commands"
}

func sortSuggestions(items []Suggestion) {
	sort.Slice(items, func(i, j int) bool {
		a := strings.ToLower(items[i].Label)
		b := strings.ToLower(items[j].Label)
		if a == b {
			return items[i].Label < items[j].Label
		}
		return a < b
	})
}
