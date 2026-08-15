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
	"context"

	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/models"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"
)

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type loadedMsg struct {
	projects      []string
	configs       []configRow
	secrets       []secretRow
	activeProject string
	activeConfig  string
}

type projectSelectedMsg struct {
	configs []configRow
	secrets []secretRow
	project string
	config  string
}

type configsLoadedMsg struct {
	project string
	configs []configRow
	expand  bool
}

type secretsLoadedMsg struct {
	secrets       []secretRow
	activeProject string
	activeConfig  string
}

func withProject(opts models.ScopedOptions, project string) models.ScopedOptions {
	opts.EnclaveProject = models.ScopedOption{Scope: "", Source: "tui", Value: project}
	return opts
}

func withProjectConfig(opts models.ScopedOptions, project, config string) models.ScopedOptions {
	opts.EnclaveProject = models.ScopedOption{Scope: "", Source: "tui", Value: project}
	opts.EnclaveConfig = models.ScopedOption{Scope: "", Source: "tui", Value: config}
	return opts
}

func loadCmd(opts models.ScopedOptions) tea.Cmd {
	return func() tea.Msg {
		var projectIDs []string
		var configInfos []models.ConfigInfo
		var computed map[string]models.ComputedSecret

		g, _ := errgroup.WithContext(context.Background())
		g.Go(func() error {
			var err controllers.Error
			projectIDs, err = controllers.GetProjectIDs(opts)
			return err.Unwrap()
		})
		g.Go(func() error {
			var err controllers.Error
			configInfos, err = controllers.GetConfigs(withProject(opts, opts.EnclaveProject.Value))
			return err.Unwrap()
		})
		g.Go(func() error {
			var err controllers.Error
			computed, err = controllers.GetSecrets(withProjectConfig(opts, opts.EnclaveProject.Value, opts.EnclaveConfig.Value))
			return err.Unwrap()
		})
		if err := g.Wait(); err != nil {
			return errMsg{err}
		}

		projects := make([]string, len(projectIDs))
		copy(projects, projectIDs)

		return loadedMsg{
			projects:      projects,
			configs:       buildConfigTree(configInfos),
			secrets:       secretsFromComputed(computed),
			activeProject: opts.EnclaveProject.Value,
			activeConfig:  opts.EnclaveConfig.Value,
		}
	}
}

func selectProjectCmd(opts models.ScopedOptions, project string, preferredConfig string) tea.Cmd {
	return func() tea.Msg {
		configInfos, err := controllers.GetConfigs(withProject(opts, project))
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}

		configs := buildConfigTree(configInfos)
		if len(configs) == 0 {
			return projectSelectedMsg{configs: configs, project: project}
		}

		configIdx := indexOfConfig(configs, preferredConfig)
		configName := configs[configIdx].name

		computed, secretsErr := controllers.GetSecrets(withProjectConfig(opts, project, configName))
		if secretsErr.Unwrap() != nil {
			return errMsg{secretsErr.Unwrap()}
		}

		return projectSelectedMsg{
			configs: configs,
			secrets: secretsFromComputed(computed),
			project: project,
			config:  configName,
		}
	}
}

func fetchProjectConfigsCmd(opts models.ScopedOptions, project string) tea.Cmd {
	return func() tea.Msg {
		configInfos, err := controllers.GetConfigs(withProject(opts, project))
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}
		return configsLoadedMsg{
			project: project,
			configs: buildConfigTree(configInfos),
			expand:  true,
		}
	}
}

func selectConfigCmd(opts models.ScopedOptions, project, config string) tea.Cmd {
	return func() tea.Msg {
		computed, err := controllers.GetSecrets(withProjectConfig(opts, project, config))
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}
		return secretsLoadedMsg{
			secrets:       secretsFromComputed(computed),
			activeProject: project,
			activeConfig:  config,
		}
	}
}

func saveSecretsCmd(opts models.ScopedOptions, project, config string, changes []models.ChangeRequest) tea.Cmd {
	return func() tea.Msg {
		computed, err := controllers.SetSecrets(withProjectConfig(opts, project, config), changes)
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}
		return secretsLoadedMsg{
			secrets:       secretsFromComputed(computed),
			activeProject: project,
			activeConfig:  config,
		}
	}
}

func createConfigCmd(opts models.ScopedOptions, project, name, environment string) tea.Cmd {
	return func() tea.Msg {
		info, err := controllers.CreateConfig(withProject(opts, project), name, environment)
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}

		configInfos, listErr := controllers.GetConfigs(withProject(opts, project))
		if listErr.Unwrap() != nil {
			return errMsg{listErr.Unwrap()}
		}

		computed, secretsErr := controllers.GetSecrets(withProjectConfig(opts, project, info.Name))
		if secretsErr.Unwrap() != nil {
			return errMsg{secretsErr.Unwrap()}
		}

		return projectSelectedMsg{
			configs: buildConfigTree(configInfos),
			secrets: secretsFromComputed(computed),
			project: project,
			config:  info.Name,
		}
	}
}
