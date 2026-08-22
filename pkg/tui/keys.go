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

	"github.com/DopplerHQ/cli/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
)

// KeysConfig is the runtime keymap (defaults merged with user overrides).
type KeysConfig struct {
	Bindings map[string]string
	Projects map[string]string
	Secrets  map[string]string
}

func defaultBaseKeys() map[string]string {
	return map[string]string{
		"q":        "quit",
		"C-c":      "quit",
		"j":        "nav down",
		"down":     "nav down",
		"k":        "nav up",
		"up":       "nav up",
		"g":        "nav top",
		"home":     "nav top",
		"G":        "nav bottom",
		"end":      "nav bottom",
		"pagedown": "nav page down",
		"pageup":   "nav page up",
		"h":        "nav left",
		"left":     "nav left",
		"l":        "nav right",
		"right":    "nav right",
		"tab":      "focus cycle",
		"backtab":  "focus prev",
		"B":        "sidebar toggle",
		"/":        "search",
		"C-f":      "search global",
		"n":        "search next",
		"N":        "search prev",
		"esc":      "command clear",
		"f":        "filter",
		"?":        "help",
		":":        "command",
		"d":        "delete",
		"u":        "secret undo",
		"s":        "secret save",
	}
}

func defaultProjectsKeys() map[string]string {
	return map[string]string{
		"space":     "fold toggle",
		"enter":     "select",
		"o":         "config create",
		"r":         "config rename",
		"L":         "config lock toggle",
		"backspace": "unload",
		"y":         "yank",
	}
}

func defaultSecretsKeys() map[string]string {
	return map[string]string{
		"enter": "edit",
		"i":     "edit",
		"a":     "edit",
		"o":     "secret add",
		"y":     "yank",
		"p":     "paste",
	}
}

func MergeKeys(user *models.TUIKeysOptions) KeysConfig {
	if user == nil {
		return KeysConfig{
			Bindings: mergeBindings(defaultBaseKeys(), nil),
			Projects: mergeOverlay(defaultProjectsKeys(), nil),
			Secrets:  mergeOverlay(defaultSecretsKeys(), nil),
		}
	}
	return KeysConfig{
		Bindings: mergeBindings(defaultBaseKeys(), user.Bindings),
		Projects: mergeOverlay(defaultProjectsKeys(), user.Projects),
		Secrets:  mergeOverlay(defaultSecretsKeys(), user.Secrets),
	}
}

func mergeBindings(base, overrides map[string]string) map[string]string {
	out := copyStringMap(base)
	for key, cmd := range overrides {
		if strings.TrimSpace(cmd) == "" {
			delete(out, key)
			continue
		}
		out[key] = cmd
	}
	return out
}

func mergeOverlay(base, overrides map[string]string) map[string]string {
	out := copyStringMap(base)
	for key, cmd := range overrides {
		out[key] = cmd // empty string blocks base fallback
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (k KeysConfig) Resolve(focus focusArea, chord string) (string, bool) {
	var overlay map[string]string
	switch focus {
	case focusProjects:
		overlay = k.Projects
	case focusSecrets:
		overlay = k.Secrets
	}
	if overlay != nil {
		if cmd, ok := overlay[chord]; ok {
			if strings.TrimSpace(cmd) == "" {
				return "", false
			}
			return cmd, true
		}
	}
	cmd, ok := k.Bindings[chord]
	if !ok || strings.TrimSpace(cmd) == "" {
		return "", false
	}
	return cmd, true
}

func (k KeysConfig) OverlayFor(focus focusArea) map[string]string {
	switch focus {
	case focusProjects:
		return k.Projects
	case focusSecrets:
		return k.Secrets
	default:
		return nil
	}
}

func (k KeysConfig) BindingsForCommand(focus focusArea, command string) []string {
	overlay := k.OverlayFor(focus)
	var keys []string
	seen := map[string]struct{}{}
	if overlay != nil {
		for key, bound := range overlay {
			if bound == command {
				keys = append(keys, key)
				seen[key] = struct{}{}
			}
		}
	}
	for key, bound := range k.Bindings {
		if _, shadowed := seen[key]; shadowed {
			continue
		}
		if overlay != nil {
			if _, ok := overlay[key]; ok {
				continue
			}
		}
		if bound == command {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) < len(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}

func encodeKey(msg tea.KeyMsg) (string, bool) {
	s := msg.String()
	if s == "" {
		return "", false
	}

	switch s {
	case "ctrl+c":
		return "C-c", true
	case " ":
		return "space", true
	case "enter":
		return "enter", true
	case "esc", "escape":
		return "esc", true
	case "up", "down", "left", "right", "home", "end", "tab":
		return s, true
	case "pgup":
		return "pageup", true
	case "pgdown":
		return "pagedown", true
	case "shift+tab":
		return "backtab", true
	case "backspace":
		return "backspace", true
	case "delete":
		return "delete-key", true
	}

	if strings.HasPrefix(s, "alt+") {
		rest := strings.TrimPrefix(s, "alt+")
		if rest == " " {
			return "A-space", true
		}
		if len([]rune(rest)) == 1 {
			r := []rune(rest)[0]
			if unicode.IsLetter(r) {
				return "A-" + string(unicode.ToLower(r)), true
			}
			return "A-" + rest, true
		}
	}

	if strings.HasPrefix(s, "ctrl+") && len(s) == 6 {
		return "C-" + strings.ToLower(string(s[5])), true
	}

	runes := []rune(s)
	if len(runes) == 1 {
		return s, true
	}
	return "", false
}

func displayKey(spec string) string {
	rest := spec
	var prefixes []string
	for {
		switch {
		case strings.HasPrefix(rest, "C-"):
			prefixes = append(prefixes, "C")
			rest = rest[2:]
		case strings.HasPrefix(rest, "A-"):
			prefixes = append(prefixes, "A")
			rest = rest[2:]
		case strings.HasPrefix(rest, "S-"):
			prefixes = append(prefixes, "S")
			rest = rest[2:]
		case strings.HasPrefix(rest, "D-"):
			prefixes = append(prefixes, "D")
			rest = rest[2:]
		default:
			goto done
		}
	}
done:
	pretty := rest
	switch rest {
	case "up":
		pretty = "↑"
	case "down":
		pretty = "↓"
	case "left":
		pretty = "←"
	case "right":
		pretty = "→"
	case "pagedown":
		pretty = "PgDn"
	case "pageup":
		pretty = "PgUp"
	case "enter":
		pretty = "Enter"
	case "esc":
		pretty = "Esc"
	case "space":
		pretty = "Space"
	case "backspace":
		pretty = "Backspace"
	case "tab":
		pretty = "Tab"
	case "backtab":
		pretty = "S-Tab"
	case "home":
		pretty = "Home"
	case "end":
		pretty = "End"
	case "delete-key":
		pretty = "Delete"
	}
	if len(prefixes) == 0 {
		return pretty
	}
	return strings.Join(prefixes, "-") + "-" + pretty
}
