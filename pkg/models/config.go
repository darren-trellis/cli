/*
Copyright © 2019 Doppler <support@doppler.com>

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
package models

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigFile structure of the config file
type ConfigFile struct {
	Scoped       map[string]FileScopedOptions `yaml:"scoped"`
	VersionCheck VersionCheck                 `yaml:"version-check"`
	Analytics    AnalyticsOptions             `yaml:"analytics,omitempty"`
	TUI          TUIOptions                   `yaml:"tui"`
	Flags        Flags                        `yaml:"flags,omitempty"`
}

// FileScopedOptions config options
type FileScopedOptions struct {
	Token          string `json:"token,omitempty" yaml:"token,omitempty"`
	APIHost        string `json:"api-host,omitempty" yaml:"api-host,omitempty"`
	DashboardHost  string `json:"dashboard-host,omitempty" yaml:"dashboard-host,omitempty"`
	VerifyTLS      string `json:"verify-tls,omitempty" yaml:"verify-tls,omitempty"`
	EnclaveProject string `json:"enclave.project,omitempty" yaml:"enclave.project,omitempty"`
	EnclaveConfig  string `json:"enclave.config,omitempty" yaml:"enclave.config,omitempty"`
}

// VersionCheck info about the last check for the latest cli version
type VersionCheck struct {
	LatestVersion string    `yaml:"latest-version,omitempty"`
	CheckedAt     time.Time `yaml:"checked-at,omitempty"`
}

type AnalyticsOptions struct {
	// we use the key 'disable' rather than 'enable' because blank value are automatically parsed as 'false',
	// and we want this feature to be enabled by default
	Disable bool `yaml:"disable"`
}

type TUIOptions struct {
	IntroVersionSeen         int             `yaml:"introVersionSeen"`
	Theme                    string          `yaml:"theme,omitempty"`
	Sidebar                  *bool           `yaml:"sidebar,omitempty"`
	SidebarWidth             int             `yaml:"sidebarWidth,omitempty"`
	SidebarPosition          string          `yaml:"sidebarPosition,omitempty"`
	PageLines                int             `yaml:"pageLines,omitempty"`
	ScrollLines              int             `yaml:"scrollLines,omitempty"`
	Border                   *bool           `yaml:"border,omitempty"`
	CaseMode                 string          `yaml:"caseMode,omitempty"`
	NameColumnPercent        int             `yaml:"nameColumnPercent,omitempty"`
	ListScrollbarVertical    *bool           `yaml:"listScrollbarVertical,omitempty"`
	SidebarScrollbarVertical *bool           `yaml:"sidebarScrollbarVertical,omitempty"`
	Autosave                 *bool           `yaml:"autosave,omitempty"`
	Autoreload               *bool           `yaml:"autoreload,omitempty"`
	Keys                     *TUIKeysOptions `yaml:"keys,omitempty"`
}

// TUIKeysOptions stores user keybindings under tui.keys.
// Flat entries are base bindings; projects/secrets are focus overlays.
type TUIKeysOptions struct {
	Bindings map[string]string `yaml:"-"`
	Projects map[string]string `yaml:"projects,omitempty"`
	Secrets  map[string]string `yaml:"secrets,omitempty"`
}

func (k *TUIKeysOptions) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("tui.keys must be a mapping")
	}
	k.Bindings = map[string]string{}
	k.Projects = nil
	k.Secrets = nil
	for i := 0; i+1 < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		key := keyNode.Value
		switch key {
		case "projects", "secrets":
			m := map[string]string{}
			if err := valNode.Decode(&m); err != nil {
				return fmt.Errorf("tui.keys.%s: %w", key, err)
			}
			if key == "projects" {
				k.Projects = m
			} else {
				k.Secrets = m
			}
		default:
			var s string
			if err := valNode.Decode(&s); err != nil {
				return fmt.Errorf("tui.keys.%s: %w", key, err)
			}
			k.Bindings[key] = s
		}
	}
	return nil
}

func (k TUIKeysOptions) MarshalYAML() (interface{}, error) {
	out := yaml.Node{Kind: yaml.MappingNode}
	add := func(key, val string) {
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val},
		)
	}
	for key, val := range k.Bindings {
		add(key, val)
	}
	encodeMap := func(name string, m map[string]string) error {
		if len(m) == 0 {
			return nil
		}
		var node yaml.Node
		if err := node.Encode(m); err != nil {
			return err
		}
		out.Content = append(out.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
			&node,
		)
		return nil
	}
	if err := encodeMap("projects", k.Projects); err != nil {
		return nil, err
	}
	if err := encodeMap("secrets", k.Secrets); err != nil {
		return nil, err
	}
	return &out, nil
}

func (k *TUIKeysOptions) IsZero() bool {
	return k == nil || (len(k.Bindings) == 0 && len(k.Projects) == 0 && len(k.Secrets) == 0)
}

// ScopedOptions options with their scope
type ScopedOptions struct {
	Token          ScopedOption `json:"token,omitempty" yaml:"token,omitempty"`
	APIHost        ScopedOption `json:"api-host,omitempty" yaml:"api-host,omitempty"`
	DashboardHost  ScopedOption `json:"dashboard-host,omitempty" yaml:"dashboard-host,omitempty"`
	VerifyTLS      ScopedOption `json:"verify-tls,omitempty" yaml:"verify-tls,omitempty"`
	EnclaveProject ScopedOption `json:"enclave.project,omitempty" yaml:"enclave.project,omitempty"`
	EnclaveConfig  ScopedOption `json:"enclave.config,omitempty" yaml:"enclave.config,omitempty"`
}

// ScopedOption value and its scope
type ScopedOption struct {
	Value  string `json:"value"`
	Scope  string `json:"scope"`
	Source string `json:"source"`
}

type source int

// the source of the value
const (
	FlagSource source = iota
	ConfigFileSource
	EnvironmentSource
	DefaultValueSource
)

func (s source) String() string {
	return [...]string{"Flag", "Config File", "Environment", "Default Value"}[s]
}

var allConfigOptions = []string{
	"token",
	"api-host",
	"dashboard-host",
	"verify-tls",
	"enclave.project",
	"enclave.config",
}

type configOption int

// valid config options
const (
	ConfigToken configOption = iota
	ConfigAPIHost
	ConfigDashboardHost
	ConfigVerifyTLS
	ConfigEnclaveProject
	ConfigEnclaveConfig
)

func (s configOption) String() string {
	return allConfigOptions[s]
}

// AllConfigOptions all supported options
func AllConfigOptions() []string {
	return allConfigOptions
}

// OptionsMap get the options for the given config
func OptionsMap(conf FileScopedOptions) map[string]string {
	return map[string]string{
		ConfigToken.String():          conf.Token,
		ConfigAPIHost.String():        conf.APIHost,
		ConfigDashboardHost.String():  conf.DashboardHost,
		ConfigVerifyTLS.String():      conf.VerifyTLS,
		ConfigEnclaveProject.String(): conf.EnclaveProject,
		ConfigEnclaveConfig.String():  conf.EnclaveConfig,
	}
}

// ScopedOptionsMap get the options for the given scoped config
func ScopedOptionsMap(conf *ScopedOptions) map[string]*ScopedOption {
	return map[string]*ScopedOption{
		ConfigToken.String():          &conf.Token,
		ConfigAPIHost.String():        &conf.APIHost,
		ConfigDashboardHost.String():  &conf.DashboardHost,
		ConfigVerifyTLS.String():      &conf.VerifyTLS,
		ConfigEnclaveProject.String(): &conf.EnclaveProject,
		ConfigEnclaveConfig.String():  &conf.EnclaveConfig,
	}
}

// ScopedOptions get the options for the given scoped config
func ScopedOptionsStringMap(conf *ScopedOptions) map[string]string {
	return map[string]string{
		ConfigToken.String():          conf.Token.Value,
		ConfigAPIHost.String():        conf.APIHost.Value,
		ConfigDashboardHost.String():  conf.DashboardHost.Value,
		ConfigVerifyTLS.String():      conf.VerifyTLS.Value,
		ConfigEnclaveProject.String(): conf.EnclaveProject.Value,
		ConfigEnclaveConfig.String():  conf.EnclaveConfig.Value,
	}
}

// EnvOptions get the scoped config options for each environment variable
func EnvOptions(conf *ScopedOptions) map[string]*ScopedOption {
	return map[string]*ScopedOption{
		"DOPPLER_TOKEN":          &conf.Token,
		"DOPPLER_API_HOST":       &conf.APIHost,
		"DOPPLER_DASHBOARD_HOST": &conf.DashboardHost,
		"DOPPLER_VERIFY_TLS":     &conf.VerifyTLS,
		"DOPPLER_PROJECT":        &conf.EnclaveProject,
		"DOPPLER_CONFIG":         &conf.EnclaveConfig,
		"ENCLAVE_PROJECT":        &conf.EnclaveProject, // deprecated, remove in v4
		"ENCLAVE_CONFIG":         &conf.EnclaveConfig,  // deprecated, remove in v4
	}
}
