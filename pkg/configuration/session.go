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
	"strings"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

const tuiSessionFileName = "tui-sessions.yaml"

// TUISession is ephemeral TUI navigation state for a working-directory scope.
type TUISession struct {
	Project      string          `yaml:"project,omitempty"`
	Config       string          `yaml:"config,omitempty"`
	ExpandedEnvs map[string]bool `yaml:"expandedEnvs,omitempty"`
}

type tuiSessionFile struct {
	Sessions map[string]TUISession `yaml:"sessions"`
}

func TUISessionFile() string {
	return filepath.Join(UserConfigDir, tuiSessionFileName)
}

func sessionScopeKey(scope string) string {
	normalized, err := NormalizeScope(scope)
	if err != nil {
		return ""
	}
	if !strings.HasSuffix(normalized, string(filepath.Separator)) {
		normalized += string(filepath.Separator)
	}
	return normalized
}

func lookupTUISession(sessions map[string]TUISession, scope string) (TUISession, bool) {
	key := sessionScopeKey(scope)
	if key == "" || len(sessions) == 0 {
		return TUISession{}, false
	}

	var match TUISession
	matchLen := -1
	for saved, session := range sessions {
		savedKey := saved
		if !strings.HasSuffix(savedKey, string(filepath.Separator)) {
			savedKey += string(filepath.Separator)
		}
		if strings.HasPrefix(key, savedKey) && len(savedKey) > matchLen {
			match = session
			matchLen = len(savedKey)
		}
	}
	if matchLen < 0 {
		return TUISession{}, false
	}
	return match, true
}

func readTUISessionFile() tuiSessionFile {
	path := TUISessionFile()
	if !utils.Exists(path) {
		return tuiSessionFile{Sessions: map[string]TUISession{}}
	}
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		utils.LogDebugError(err)
		return tuiSessionFile{Sessions: map[string]TUISession{}}
	}
	var file tuiSessionFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		utils.LogDebugError(err)
		return tuiSessionFile{Sessions: map[string]TUISession{}}
	}
	if file.Sessions == nil {
		file.Sessions = map[string]TUISession{}
	}
	return file
}

func writeTUISessionFile(file tuiSessionFile) {
	if file.Sessions == nil {
		file.Sessions = map[string]TUISession{}
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		utils.LogDebugError(err)
		return
	}
	if err := os.MkdirAll(UserConfigDir, 0700); err != nil {
		utils.LogDebugError(err)
		return
	}
	if err := utils.WriteFile(TUISessionFile(), data, 0600); err != nil {
		utils.LogDebugError(err)
	}
}

// LoadTUISession returns the session for the current CLI scope, if any.
func LoadTUISession() (TUISession, bool) {
	return tuiSessionFor(Scope)
}

func tuiSessionFor(scope string) (TUISession, bool) {
	return lookupTUISession(readTUISessionFile().Sessions, scope)
}

// SaveTUISession writes session state for the current CLI scope.
func SaveTUISession(session TUISession) {
	saveTUISessionFor(Scope, session)
}

func saveTUISessionFor(scope string, session TUISession) {
	key := sessionScopeKey(scope)
	if key == "" {
		return
	}
	file := readTUISessionFile()
	file.Sessions[key] = session
	writeTUISessionFile(file)
}

// ShouldRestoreTUISession is false when project/config were set by flag or env.
func ShouldRestoreTUISession(opts models.ScopedOptions) bool {
	return !isExplicitSource(opts.EnclaveProject.Source) && !isExplicitSource(opts.EnclaveConfig.Source)
}

func isExplicitSource(source string) bool {
	return source == models.FlagSource.String() || source == models.EnvironmentSource.String()
}

// ApplyTUISession overlays saved project/config onto setup-scoped options.
func ApplyTUISession(opts models.ScopedOptions, session TUISession) models.ScopedOptions {
	if session.Project != "" {
		opts.EnclaveProject.Value = session.Project
		opts.EnclaveProject.Source = "TUI Session"
	}
	if session.Config != "" {
		opts.EnclaveConfig.Value = session.Config
		opts.EnclaveConfig.Source = "TUI Session"
	}
	return opts
}
