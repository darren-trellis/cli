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
	"sort"

	"github.com/DopplerHQ/cli/pkg/models"
)

type configRow struct {
	name        string
	environment string
	root        bool
	depth       int
	lastSibling bool
}

func buildConfigTree(infos []models.ConfigInfo) []configRow {
	type envGroup struct {
		root *models.ConfigInfo
		kids []models.ConfigInfo
	}

	order := make([]string, 0)
	groups := make(map[string]*envGroup)

	for i := range infos {
		c := infos[i]
		g, ok := groups[c.Environment]
		if !ok {
			g = &envGroup{}
			groups[c.Environment] = g
			order = append(order, c.Environment)
		}
		if c.Root {
			cp := c
			g.root = &cp
		} else {
			g.kids = append(g.kids, c)
		}
	}

	rows := make([]configRow, 0, len(infos))
	for _, env := range order {
		g := groups[env]
		sort.Slice(g.kids, func(i, j int) bool {
			return g.kids[i].Name < g.kids[j].Name
		})

		if g.root != nil {
			rows = append(rows, configRow{
				name:        g.root.Name,
				environment: env,
				root:        true,
				depth:       0,
			})
			for i, kid := range g.kids {
				rows = append(rows, configRow{
					name:        kid.Name,
					environment: env,
					depth:       1,
					lastSibling: i == len(g.kids)-1,
				})
			}
			continue
		}

		for i, kid := range g.kids {
			rows = append(rows, configRow{
				name:        kid.Name,
				environment: env,
				depth:       0,
				lastSibling: i == len(g.kids)-1,
			})
		}
	}
	return rows
}

func indexOfConfig(rows []configRow, name string) int {
	for i, r := range rows {
		if r.name == name {
			return i
		}
	}
	return 0
}
