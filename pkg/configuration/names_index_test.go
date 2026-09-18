/*
Copyright © 2023 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

	    10|Unless required by applicable law or agreed to in writing, software

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

func TestTUINamesIndexRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	names := map[string][]string{
		"api\x00dev": {"TOKEN", "OTHER"},
		"web\x00prd": {"WEB_KEY"},
	}
	SaveTUINamesIndex("tok", "https://api.doppler.com", names)

	got, ok := LoadTUINamesIndex("tok", "https://api.doppler.com")
	require.True(t, ok)
	assert.Equal(t, []string{"TOKEN", "OTHER"}, got["api\x00dev"])
	assert.Equal(t, []string{"WEB_KEY"}, got["web\x00prd"])
}

func TestTUINamesIndexIsTokenScoped(t *testing.T) {
	withTempConfigDir(t)
	SaveTUINamesIndex("tok-a", "https://api.doppler.com", map[string][]string{
		"api\x00dev": {"A"},
	})
	SaveTUINamesIndex("tok-b", "https://api.doppler.com", map[string][]string{
		"api\x00dev": {"B"},
	})

	a, ok := LoadTUINamesIndex("tok-a", "https://api.doppler.com")
	require.True(t, ok)
	assert.Equal(t, []string{"A"}, a["api\x00dev"])

	b, ok := LoadTUINamesIndex("tok-b", "https://api.doppler.com")
	require.True(t, ok)
	assert.Equal(t, []string{"B"}, b["api\x00dev"])
}

func TestLoadTUINamesIndexMissing(t *testing.T) {
	withTempConfigDir(t)
	_, ok := LoadTUINamesIndex("tok", "https://api.doppler.com")
	assert.False(t, ok)
}
