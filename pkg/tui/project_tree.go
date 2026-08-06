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

type treeKind int

const (
	treeProject treeKind = iota
	treeConfig
)

type treeRow struct {
	kind        treeKind
	project     string
	config      string
	configDepth int
	lastSibling bool
	folded      bool
	pinned      bool
}

func buildProjectTree(
	projects []string,
	projectConfigs map[string][]configRow,
	expanded map[string]bool,
	activeProject, activeConfig string,
) []treeRow {
	tree := make([]treeRow, 0, len(projects)*2)
	for _, project := range projects {
		isExpanded := expanded[project]
		tree = append(tree, treeRow{
			kind:    treeProject,
			project: project,
			folded:  !isExpanded,
		})

		configs := projectConfigs[project]
		if isExpanded {
			for _, c := range configs {
				tree = append(tree, treeRow{
					kind:        treeConfig,
					project:     project,
					config:      c.name,
					configDepth: c.depth,
					lastSibling: c.lastSibling,
				})
			}
			continue
		}

		if project == activeProject && activeConfig != "" {
			tree = append(tree, treeRow{
				kind:        treeConfig,
				project:     project,
				config:      activeConfig,
				configDepth: 0,
				lastSibling: true,
				pinned:      true,
			})
		}
	}
	return tree
}

func findTreeIndex(tree []treeRow, kind treeKind, project, config string) int {
	if project == "" {
		if len(tree) == 0 {
			return 0
		}
		return 0
	}

	if kind == treeConfig && config != "" {
		for i, row := range tree {
			if row.kind == treeConfig && row.project == project && row.config == config {
				return i
			}
		}
	}

	for i, row := range tree {
		if row.kind == treeProject && row.project == project {
			return i
		}
	}
	if len(tree) == 0 {
		return 0
	}
	return 0
}

func formatTreeRow(row treeRow, activeProject, activeConfig string) string {
	if row.kind == treeProject {
		marker := "  "
		if row.project == activeProject {
			marker = "* "
		}
		icon := "▾ "
		if row.folded {
			icon = "▸ "
		}
		return marker + icon + row.project
	}

	indent := "  "
	prefix := "  "
	if row.project == activeProject && row.config == activeConfig {
		prefix = "* "
	}
	branch := ""
	if row.configDepth > 0 {
		if row.lastSibling {
			branch = "└─ "
		} else {
			branch = "├─ "
		}
	}
	return indent + prefix + branch + row.config
}
