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
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/utils"
)

const tuiNamesIndexDirName = "tui-index"

type tuiNamesIndexFile struct {
	Version int                            `json:"version"`
	Names   map[string]map[string][]string `json:"names"`
}

func tuiNamesIndexPath(token, apiHost string) string {
	sum := crypto.Hash(apiHost + "\x00" + token)
	return filepath.Join(UserConfigDir, tuiNamesIndexDirName, sum+".json")
}

func flattenTUINamesIndex(nested map[string]map[string][]string) map[string][]string {
	out := map[string][]string{}
	for project, configs := range nested {
		if project == "" {
			continue
		}
		for config, names := range configs {
			if config == "" {
				continue
			}
			out[project+"\x00"+config] = append([]string(nil), names...)
		}
	}
	return out
}

func nestTUINamesIndex(flat map[string][]string) map[string]map[string][]string {
	out := map[string]map[string][]string{}
	for key, names := range flat {
		project, config, ok := splitTUINamesIndexKey(key)
		if !ok {
			continue
		}
		if out[project] == nil {
			out[project] = map[string][]string{}
		}
		out[project][config] = append([]string(nil), names...)
	}
	return out
}

func splitTUINamesIndexKey(key string) (string, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			if i == 0 || i+1 >= len(key) {
				return "", "", false
			}
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}

// LoadTUINamesIndex returns cached secret names keyed by project\x00config.
func LoadTUINamesIndex(token, apiHost string) (map[string][]string, bool) {
	if token == "" {
		return nil, false
	}
	path := tuiNamesIndexPath(token, apiHost)
	if !utils.Exists(path) {
		return nil, false
	}
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		utils.LogDebugError(err)
		return nil, false
	}
	var file tuiNamesIndexFile
	if err := json.Unmarshal(data, &file); err != nil {
		utils.LogDebugError(err)
		return nil, false
	}
	if len(file.Names) == 0 {
		return nil, false
	}
	return flattenTUINamesIndex(file.Names), true
}

// SaveTUINamesIndex writes secret names keyed by project\x00config.
func SaveTUINamesIndex(token, apiHost string, names map[string][]string) {
	if token == "" {
		return
	}
	file := tuiNamesIndexFile{
		Version: 1,
		Names:   nestTUINamesIndex(names),
	}
	data, err := json.Marshal(file)
	if err != nil {
		utils.LogDebugError(err)
		return
	}
	dir := filepath.Join(UserConfigDir, tuiNamesIndexDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		utils.LogDebugError(err)
		return
	}
	if err := utils.WriteFile(tuiNamesIndexPath(token, apiHost), data, 0600); err != nil {
		utils.LogDebugError(err)
	}
}
