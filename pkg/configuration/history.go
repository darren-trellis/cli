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
	"os"
	"path/filepath"

	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

const tuiHistoryFileName = "tui-history.yaml"

type TUIHistory struct {
	Commands []string `yaml:"commands,omitempty"`
	Searches []string `yaml:"searches,omitempty"`
}

func TUIHistoryFile() string {
	return filepath.Join(UserConfigDir, tuiHistoryFileName)
}

func LoadTUIHistory() TUIHistory {
	path := TUIHistoryFile()
	if !utils.Exists(path) {
		return TUIHistory{}
	}
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		utils.LogDebugError(err)
		return TUIHistory{}
	}
	var hist TUIHistory
	if err := yaml.Unmarshal(data, &hist); err != nil {
		utils.LogDebugError(err)
		return TUIHistory{}
	}
	return hist
}

func SaveTUIHistory(hist TUIHistory) {
	data, err := yaml.Marshal(hist)
	if err != nil {
		utils.LogDebugError(err)
		return
	}
	if err := os.MkdirAll(UserConfigDir, 0700); err != nil {
		utils.LogDebugError(err)
		return
	}
	if err := utils.WriteFile(TUIHistoryFile(), data, 0600); err != nil {
		utils.LogDebugError(err)
	}
}
