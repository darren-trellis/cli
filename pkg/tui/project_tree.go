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
	cached          bool
	dirty           bool
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

func keepConfig(project, config, activeProject, activeConfig string, keep map[string]bool) bool {
	if config == "" {
		return false
	}
	if project == activeProject && config == activeConfig {
		return true
	}
	return keep[secretsCacheKey(project, config)]
}

func buildProjectTree(
	projects []string,
	projectConfigs map[string][]configRow,
	expanded map[string]bool,
	expandedEnvs map[string]bool,
	activeProject, activeConfig string,
	keep map[string]bool,
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
			tree = append(tree, configTreeRows(project, configs, expandedEnvs, activeProject, activeConfig, keep)...)
			continue
		}

		tree = append(tree, pinnedProjectRows(project, configs, activeProject, activeConfig, keep)...)
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

func pinnedKids(project string, kids []configRow, activeProject, activeConfig string, keep map[string]bool) []configRow {
	var out []configRow
	for _, kid := range kids {
		if keepConfig(project, kid.name, activeProject, activeConfig, keep) {
			out = append(out, kid)
		}
	}
	return out
}

func pinnedProjectRows(
	project string,
	configs []configRow,
	activeProject, activeConfig string,
	keep map[string]bool,
) []treeRow {
	groups := groupConfigs(configs)
	type part struct {
		root     configRow
		showRoot bool
		kids     []configRow
	}
	var parts []part
	for _, g := range groups {
		showRoot := keepConfig(project, g.root.name, activeProject, activeConfig, keep)
		kids := pinnedKids(project, g.kids, activeProject, activeConfig, keep)
		if !showRoot && len(kids) == 0 {
			continue
		}
		parts = append(parts, part{root: g.root, showRoot: showRoot, kids: kids})
	}

	rows := make([]treeRow, 0, len(parts)*2)
	for pi, p := range parts {
		parentIsLast := pi == len(parts)-1
		if p.showRoot {
			showKids := len(p.kids) > 0
			rows = append(rows, treeRow{
				kind:        treeConfig,
				project:     project,
				config:      p.root.name,
				depth:       1,
				lastSibling: parentIsLast,
				folded:      showKids,
				hasChildren: showKids,
				foldRoot:    p.root.name,
				locked:      p.root.locked,
				pinned:      true,
			})
			for i, kid := range p.kids {
				rows = append(rows, treeRow{
					kind:            treeConfig,
					project:         project,
					config:          kid.name,
					depth:           2,
					lastSibling:     i == len(p.kids)-1,
					parentContinues: !parentIsLast,
					pinned:          true,
					foldRoot:        p.root.name,
					locked:          kid.locked,
				})
			}
			continue
		}
		for i, kid := range p.kids {
			rows = append(rows, treeRow{
				kind:        treeConfig,
				project:     project,
				config:      kid.name,
				depth:       1,
				lastSibling: parentIsLast && i == len(p.kids)-1,
				pinned:      true,
				locked:      kid.locked,
			})
		}
	}
	return rows
}

func configTreeRows(
	project string,
	configs []configRow,
	expandedEnvs map[string]bool,
	activeProject, activeConfig string,
	keep map[string]bool,
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

		pinned := pinnedKids(project, g.kids, activeProject, activeConfig, keep)
		for i, kid := range pinned {
			rows = append(rows, treeRow{
				kind:            treeConfig,
				project:         project,
				config:          kid.name,
				depth:           2,
				lastSibling:     i == len(pinned)-1,
				parentContinues: !parentIsLast,
				pinned:          true,
				foldRoot:        g.root.name,
				locked:          kid.locked,
			})
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

const (
	treeExpanded  = "◇"
	treeCollapsed = "◆"
	treeLocked    = "◉"
)

func treeFoldMark(folded bool) string {
	if folded {
		return treeCollapsed
	}
	return treeExpanded
}

func formatTreeRow(row treeRow) string {
	if row.kind == treeProject {
		return treeFoldMark(row.folded) + " " + row.project
	}

	label := row.config
	if row.dirty {
		label += " +"
	}
	if row.locked {
		label += " " + treeLocked
	}

	node := "─"
	if row.hasChildren {
		node = treeFoldMark(row.folded)
	}
	branch := "├──"
	if row.lastSibling {
		branch = "└──"
	}
	line := branch + node + " " + label
	if row.depth == 2 {
		guide := "│  "
		if !row.parentContinues {
			guide = "   "
		}
		return guide + line
	}
	return line
}

func configIsLocked(configs []configRow, name string) bool {
	for _, c := range configs {
		if c.name == name {
			return c.locked
		}
	}
	return false
}
