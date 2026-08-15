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
package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTUIHistoryRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	SaveTUIHistory(TUIHistory{
		Commands: []string{"help", "config delete"},
		Searches: []string{"API", "TOKEN"},
	})
	got := LoadTUIHistory()
	assert.Equal(t, []string{"help", "config delete"}, got.Commands)
	assert.Equal(t, []string{"API", "TOKEN"}, got.Searches)
}

func TestLoadTUIHistoryMissingFile(t *testing.T) {
	withTempConfigDir(t)
	got := LoadTUIHistory()
	require.Empty(t, got.Commands)
	require.Empty(t, got.Searches)
}
