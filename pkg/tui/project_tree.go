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
	kind            treeKind
	project         string
	config          string
	depth           int // 1 = child of project, 2 = grandchild
	lastSibling     bool
	parentContinues bool // for depth 2: draw │ under non-final parent
	folded          bool
	pinned          bool
	hasChildren     bool
	locked          bool
	foldRoot        string // root config name for env fold target
}

func envKey(project, rootConfig string) string {
	return project + "\x00" + rootConfig
}

func isEnvExpanded(expandedEnvs map[string]bool, project, rootConfig string) bool {
	if expandedEnvs == nil {
		return true
	}
	expanded, ok := expandedEnvs[envKey(project, rootConfig)]
	if !ok {
		return true
	}
	return expanded
}

func buildProjectTree(
	projects []string,
	projectConfigs map[string][]configRow,
	expanded map[string]bool,
	expandedEnvs map[string]bool,
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
			tree = append(tree, configTreeRows(project, configs, expandedEnvs, activeProject, activeConfig)...)
			continue
		}

		if project == activeProject && activeConfig != "" {
			tree = append(tree, treeRow{
				kind:        treeConfig,
				project:     project,
				config:      activeConfig,
				depth:       1,
				lastSibling: true,
				pinned:      true,
				locked:      configIsLocked(configs, activeConfig),
			})
		}
	}
	return tree
}

type envGroup struct {
	root configRow
	kids []configRow
}

func groupConfigs(configs []configRow) []envGroup {
	groups := make([]envGroup, 0)
	for _, c := range configs {
		if c.depth == 0 {
			groups = append(groups, envGroup{root: c})
			continue
		}
		if len(groups) == 0 {
			groups = append(groups, envGroup{root: c})
			continue
		}
		groups[len(groups)-1].kids = append(groups[len(groups)-1].kids, c)
	}
	return groups
}

func configTreeRows(
	project string,
	configs []configRow,
	expandedEnvs map[string]bool,
	activeProject, activeConfig string,
) []treeRow {
	groups := groupConfigs(configs)
	rows := make([]treeRow, 0, len(configs))

	for gi, g := range groups {
		parentIsLast := gi == len(groups)-1
		hasChildren := len(g.kids) > 0
		envExpanded := !hasChildren || isEnvExpanded(expandedEnvs, project, g.root.name)

		rows = append(rows, treeRow{
			kind:        treeConfig,
			project:     project,
			config:      g.root.name,
			depth:       1,
			lastSibling: parentIsLast,
			folded:      hasChildren && !envExpanded,
			hasChildren: hasChildren,
			foldRoot:    g.root.name,
			locked:      g.root.locked,
		})

		if !hasChildren {
			continue
		}

		if envExpanded {
			for i, kid := range g.kids {
				rows = append(rows, treeRow{
					kind:            treeConfig,
					project:         project,
					config:          kid.name,
					depth:           2,
					lastSibling:     i == len(g.kids)-1,
					parentContinues: !parentIsLast,
					foldRoot:        g.root.name,
					hasChildren:     false,
					locked:          kid.locked,
				})
			}
			continue
		}

		if project == activeProject && activeConfig != "" && activeConfig != g.root.name {
			for _, kid := range g.kids {
				if kid.name == activeConfig {
					rows = append(rows, treeRow{
						kind:            treeConfig,
						project:         project,
						config:          kid.name,
						depth:           2,
						lastSibling:     true,
						parentContinues: !parentIsLast,
						pinned:          true,
						foldRoot:        g.root.name,
						locked:          kid.locked,
					})
					break
				}
			}
		}
	}
	return rows
}

func findTreeIndex(tree []treeRow, kind treeKind, project, config string) int {
	if kind == treeConfig && project != "" && config != "" {
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

func formatTreeRow(row treeRow, activeProject, activeConfig string, secretsDirty bool) string {
	if row.kind == treeProject {
		icon := "▾ "
		if row.folded {
			icon = "▸ "
		}
		return icon + row.project
	}

	active := row.project == activeProject && row.config == activeConfig
	label := row.config
	if row.locked {
		label = label + " 🔒"
	}
	if active && secretsDirty {
		label = label + " +"
	}
	name := label
	if active {
		name = activeEnvStyle.Render("*" + label)
	}
	if row.hasChildren {
		icon := "▾ "
		if row.folded {
			icon = "▸ "
		}
		name = icon + name
	}

	branch := "├─ "
	if row.lastSibling {
		branch = "└─ "
	}

	switch row.depth {
	case 2:
		guide := "│  "
		if !row.parentContinues {
			guide = "   "
		}
		return "  " + guide + branch + name
	default:
		return "  " + branch + name
	}
}

func configIsLocked(configs []configRow, name string) bool {
	for _, c := range configs {
		if c.name == name {
			return c.locked
		}
	}
	return false
}
