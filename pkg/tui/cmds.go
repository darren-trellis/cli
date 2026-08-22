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
	"regexp"
	"sync"

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

type projectDeletedMsg struct {
	projects  []string
	deleted   string
	highlight string
	switched  bool
	configs   []configRow
	secrets   []secretRow
	project   string
	config    string
}

type configLockMsg struct {
	project string
	configs []configRow
	config  string
	locked  bool
}

type secretsLoadedMsg struct {
	secrets       []secretRow
	activeProject string
	activeConfig  string
}

type workplaceSearchMsg struct {
	gen     uint64
	query   string
	hits    []globalHit
	fetched map[string][]secretRow
	configs map[string][]configRow
	err     error
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

func loadCmd(opts, fallback models.ScopedOptions) tea.Cmd {
	return func() tea.Msg {
		msg := loadOnce(opts)
		if _, ok := msg.(errMsg); ok && !sameLoadTarget(opts, fallback) {
			return loadOnce(fallback)
		}
		return msg
	}
}

func sameLoadTarget(a, b models.ScopedOptions) bool {
	return a.EnclaveProject.Value == b.EnclaveProject.Value && a.EnclaveConfig.Value == b.EnclaveConfig.Value
}

func loadOnce(opts models.ScopedOptions) tea.Msg {
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

type quitNowMsg struct{}

func saveAllDirtyCmd(opts models.ScopedOptions, groups []dirtyGroup) tea.Cmd {
	return func() tea.Msg {
		for _, g := range groups {
			if len(g.changes) == 0 {
				continue
			}
			_, err := controllers.SetSecrets(withProjectConfig(opts, g.project, g.config), g.changes)
			if err.Unwrap() != nil {
				return errMsg{err.Unwrap()}
			}
		}
		return quitNowMsg{}
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

func renameConfigCmd(opts models.ScopedOptions, project, from, to string) tea.Cmd {
	return func() tea.Msg {
		info, err := controllers.UpdateConfig(withProjectConfig(opts, project, from), to)
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

func deleteConfigCmd(opts models.ScopedOptions, project, config, stayConfig, env string) tea.Cmd {
	return func() tea.Msg {
		err := controllers.DeleteConfig(withProjectConfig(opts, project, config))
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}

		configInfos, listErr := controllers.GetConfigs(withProject(opts, project))
		if listErr.Unwrap() != nil {
			return errMsg{listErr.Unwrap()}
		}
		configs := buildConfigTree(configInfos)

		next := stayConfig
		if next == config || !configExists(configs, next) {
			next = pickConfigAfterDelete(configs, env)
		}

		if next == "" {
			return projectSelectedMsg{configs: configs, project: project}
		}
		computed, secretsErr := controllers.GetSecrets(withProjectConfig(opts, project, next))
		if secretsErr.Unwrap() != nil {
			return errMsg{secretsErr.Unwrap()}
		}
		return projectSelectedMsg{
			configs: configs,
			secrets: secretsFromComputed(computed),
			project: project,
			config:  next,
		}
	}
}

func pickConfigAfterDelete(configs []configRow, env string) string {
	if env != "" {
		for _, c := range configs {
			if c.environment == env {
				return c.name
			}
		}
	}
	if len(configs) > 0 {
		return configs[0].name
	}
	return ""
}

func deleteProjectCmd(opts models.ScopedOptions, deleted string, oldProjects []string, stayProject string) tea.Cmd {
	return func() tea.Msg {
		err := controllers.DeleteProject(withProject(opts, deleted))
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}
		ids, listErr := controllers.GetProjectIDs(opts)
		if listErr.Unwrap() != nil {
			return errMsg{listErr.Unwrap()}
		}
		highlight := pickProjectAfterDelete(ids, oldProjects, deleted)
		nextProject := stayProject
		if stayProject == deleted || !containsString(ids, stayProject) {
			nextProject = highlight
		}
		if nextProject == "" || nextProject == stayProject {
			return projectDeletedMsg{projects: ids, deleted: deleted, highlight: highlight}
		}

		configInfos, cfgErr := controllers.GetConfigs(withProject(opts, nextProject))
		if cfgErr.Unwrap() != nil {
			return errMsg{cfgErr.Unwrap()}
		}
		configs := buildConfigTree(configInfos)
		configName := ""
		if len(configs) > 0 {
			configName = configs[0].name
		}
		if configName == "" {
			return projectDeletedMsg{
				projects:  ids,
				deleted:   deleted,
				highlight: highlight,
				switched:  true,
				configs:   configs,
				project:   nextProject,
			}
		}
		computed, secretsErr := controllers.GetSecrets(withProjectConfig(opts, nextProject, configName))
		if secretsErr.Unwrap() != nil {
			return errMsg{secretsErr.Unwrap()}
		}
		return projectDeletedMsg{
			projects:  ids,
			deleted:   deleted,
			highlight: highlight,
			switched:  true,
			configs:   configs,
			secrets:   secretsFromComputed(computed),
			project:   nextProject,
			config:    configName,
		}
	}
}

func pickProjectAfterDelete(remaining, old []string, deleted string) string {
	inRemaining := map[string]bool{}
	for _, p := range remaining {
		inRemaining[p] = true
	}
	idx := -1
	for i, p := range old {
		if p == deleted {
			idx = i
			break
		}
	}
	for i := idx + 1; i < len(old); i++ {
		if inRemaining[old[i]] {
			return old[i]
		}
	}
	for i := idx - 1; i >= 0; i-- {
		if inRemaining[old[i]] {
			return old[i]
		}
	}
	if len(remaining) > 0 {
		return remaining[0]
	}
	return ""
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func setConfigLockCmd(opts models.ScopedOptions, project, config string, lock bool) tea.Cmd {
	return func() tea.Msg {
		scoped := withProjectConfig(opts, project, config)
		var info models.ConfigInfo
		var err controllers.Error
		if lock {
			info, err = controllers.LockConfig(scoped)
		} else {
			info, err = controllers.UnlockConfig(scoped)
		}
		if err.Unwrap() != nil {
			return errMsg{err.Unwrap()}
		}

		configInfos, listErr := controllers.GetConfigs(withProject(opts, project))
		if listErr.Unwrap() != nil {
			return errMsg{listErr.Unwrap()}
		}

		return configLockMsg{
			project: project,
			configs: buildConfigTree(configInfos),
			config:  info.Name,
			locked:  info.Locked,
		}
	}
}

func workplaceSearchCmd(
	opts models.ScopedOptions,
	gen uint64,
	query, caseMode string,
	projects []string,
	known map[string][]configRow,
	cached map[string][]secretRow,
) tea.Cmd {
	return func() tea.Msg {
		pattern := query
		if searchShouldIgnoreCase(query, caseMode) {
			pattern = "(?i)" + query
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return workplaceSearchMsg{gen: gen, query: query, err: err}
		}

		g, _ := errgroup.WithContext(context.Background())
		g.SetLimit(8)
		var mu sync.Mutex
		var hits []globalHit
		fetchedSecrets := map[string][]secretRow{}
		fetchedConfigs := map[string][]configRow{}

		for _, project := range projects {
			project := project
			g.Go(func() error {
				cfgs, ok := known[project]
				if !ok {
					infos, cerr := controllers.GetConfigs(withProject(opts, project))
					if cerr.Unwrap() != nil {
						return nil
					}
					cfgs = buildConfigTree(infos)
					mu.Lock()
					fetchedConfigs[project] = cfgs
					mu.Unlock()
				}
				for _, cfg := range cfgs {
					cfg := cfg
					g.Go(func() error {
						key := secretsCacheKey(project, cfg.name)
						secrets, ok := cached[key]
						if !ok {
							computed, serr := controllers.GetSecrets(withProjectConfig(opts, project, cfg.name))
							if serr.Unwrap() != nil {
								return nil
							}
							secrets = secretsFromComputed(computed)
							mu.Lock()
							fetchedSecrets[key] = secrets
							mu.Unlock()
						}
						found := collectSecretHits(re, project, cfg.name, secrets)
						if len(found) == 0 {
							return nil
						}
						mu.Lock()
						hits = append(hits, found...)
						mu.Unlock()
						return nil
					})
				}
				return nil
			})
		}
		if waitErr := g.Wait(); waitErr != nil {
			return workplaceSearchMsg{gen: gen, query: query, err: waitErr}
		}
		return workplaceSearchMsg{
			gen:     gen,
			query:   query,
			hits:    hits,
			fetched: fetchedSecrets,
			configs: fetchedConfigs,
		}
	}
}
