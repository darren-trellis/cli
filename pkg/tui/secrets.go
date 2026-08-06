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
)

type secretRow struct {
	originalName       *string
	originalValue      string
	originalVisibility string
	name               string
	value              string
	isTouched          bool
	shouldDelete       bool
}

func newSecretRow(name, value, visibility string) secretRow {
	n := name
	return secretRow{
		originalName:       &n,
		originalValue:      value,
		originalVisibility: visibility,
		name:               name,
		value:              value,
	}
}

func newEmptySecretRow() secretRow {
	return secretRow{}
}

func (s secretRow) originalNameString() string {
	if s.originalName == nil {
		return ""
	}
	return *s.originalName
}

func (s secretRow) isDirty() bool {
	nameChanged := s.name != s.originalNameString()
	valueChanged := (s.originalVisibility != "restricted" && s.value != s.originalValue) ||
		(s.originalVisibility == "restricted" && s.isTouched)
	return s.originalName == nil || nameChanged || valueChanged || s.shouldDelete
}

func (s secretRow) shouldSubmit() bool {
	return s.isDirty() && (len(s.name) > 0 || s.originalNameString() != "")
}

func (s secretRow) toChangeRequest() models.ChangeRequest {
	shouldDelete := s.shouldDelete
	cr := models.ChangeRequest{
		OriginalName: nil,
		Name:         s.name,
		Value:        s.value,
		ShouldDelete: &shouldDelete,
	}
	if s.originalName != nil {
		cr.OriginalName = *s.originalName
	}
	if s.originalVisibility != "restricted" {
		cr.OriginalValue = s.originalValue
	}
	return cr
}

func (s secretRow) displayValue() string {
	if s.originalVisibility == "restricted" && !s.isTouched {
		return "[RESTRICTED]"
	}
	return s.value
}

func (s secretRow) previewValue() string {
	v := s.displayValue()
	v = strings.ReplaceAll(v, "\n", " ")
	if len(v) > 40 {
		return v[:40] + "…"
	}
	return v
}

func (s *secretRow) undo() {
	s.shouldDelete = false
	s.isTouched = false
	s.name = s.originalNameString()
	if s.originalVisibility == "restricted" {
		s.value = ""
	} else {
		s.value = s.originalValue
	}
}

func normalizeSecretName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(unicode.ToUpper(r))
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func secretsFromComputed(computed map[string]models.ComputedSecret) []secretRow {
	var secrets []secretRow
	for _, cs := range computed {
		if cs.Name == "DOPPLER_CONFIG" || cs.Name == "DOPPLER_ENVIRONMENT" || cs.Name == "DOPPLER_PROJECT" {
			continue
		}
		value := ""
		if cs.RawValue != nil {
			value = *cs.RawValue
		}
		secrets = append(secrets, newSecretRow(cs.Name, value, cs.RawVisibility))
	}
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].name < secrets[j].name
	})
	return secrets
}

func collectChanges(secrets []secretRow) []models.ChangeRequest {
	var changes []models.ChangeRequest
	for _, s := range secrets {
		if s.shouldSubmit() {
			changes = append(changes, s.toChangeRequest())
		}
	}
	return changes
}

func filterSecretIndexes(secrets []secretRow, filter string) []int {
	filter = strings.ToUpper(filter)
	var idxs []int
	for i, s := range secrets {
		if s.isDirty() || filter == "" || strings.Contains(strings.ToUpper(s.name), filter) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
