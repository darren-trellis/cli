/*
Copyright © 2020 Doppler <support@doppler.com>

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
package cmd

import (
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	tuiApp "github.com/DopplerHQ/cli/pkg/tui"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch TUI",
	Args:  cobra.NoArgs,
	Run:   tui,
}

func tui(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)

	cfg := configuration.TUIConfig()
	if cmd.Flags().Changed("theme") {
		theme := cmd.Flag("theme").Value.String()
		if err := tuiApp.CheckTheme(theme); err != nil {
			utils.HandleError(err)
		}
		cfg.Theme = theme
		configuration.TUISetTheme(cfg.Theme)
	}
	if cmd.Flags().Changed("sidebar") {
		cfg.Sidebar, _ = cmd.Flags().GetBool("sidebar")
		configuration.TUISetSidebar(cfg.Sidebar)
	}
	if cmd.Flags().Changed("sidebar-width") {
		cfg.SidebarWidth, _ = cmd.Flags().GetInt("sidebar-width")
		configuration.TUISetSidebarWidth(cfg.SidebarWidth)
	}
	if cmd.Flags().Changed("sidebar-position") {
		cfg.SidebarPosition, _ = cmd.Flags().GetString("sidebar-position")
		configuration.TUISetSidebarPosition(cfg.SidebarPosition)
	}
	if cmd.Flags().Changed("page-lines") {
		cfg.PageLines, _ = cmd.Flags().GetInt("page-lines")
		configuration.TUISetPageLines(cfg.PageLines)
	}
	if cmd.Flags().Changed("scroll-lines") {
		cfg.ScrollLines, _ = cmd.Flags().GetInt("scroll-lines")
		configuration.TUISetScrollLines(cfg.ScrollLines)
	}
	if cmd.Flags().Changed("border") {
		cfg.Border, _ = cmd.Flags().GetBool("border")
		configuration.TUISetBorder(cfg.Border)
	}
	if cmd.Flags().Changed("case-mode") {
		cfg.CaseMode, _ = cmd.Flags().GetString("case-mode")
		configuration.TUISetCaseMode(cfg.CaseMode)
	}
	if cmd.Flags().Changed("name-column-percent") {
		cfg.NameColumnPercent, _ = cmd.Flags().GetInt("name-column-percent")
		configuration.TUISetNameColumnPercent(cfg.NameColumnPercent)
	}
	if cmd.Flags().Changed("list-scrollbar") {
		cfg.ListScrollbarVertical, _ = cmd.Flags().GetBool("list-scrollbar")
		configuration.TUISetListScrollbarVertical(cfg.ListScrollbarVertical)
	}
	if cmd.Flags().Changed("sidebar-scrollbar") {
		cfg.SidebarScrollbarVertical, _ = cmd.Flags().GetBool("sidebar-scrollbar")
		configuration.TUISetSidebarScrollbarVertical(cfg.SidebarScrollbarVertical)
	}
	if cmd.Flags().Changed("autosave") {
		cfg.Autosave, _ = cmd.Flags().GetBool("autosave")
		configuration.TUISetAutosave(cfg.Autosave)
	}
	if cmd.Flags().Changed("autoreload") {
		cfg.Autoreload, _ = cmd.Flags().GetBool("autoreload")
		configuration.TUISetAutoreload(cfg.Autoreload)
	}
	cfg = configuration.TUIConfig()
	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = "default"
	}

	tuiApp.Start(localConfig, cfg)
}

func init() {
	tuiCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	tuiCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	tuiCmd.Flags().String("theme", "default", "TUI theme (default, cool, warm, mono, or a named theme like catppuccin, gruvbox, tokyo-night; an unknown name lists them all)")
	tuiCmd.Flags().Bool("sidebar", true, "show projects sidebar")
	tuiCmd.Flags().Int("sidebar-width", 0, "projects sidebar width in columns (0 = auto)")
	tuiCmd.Flags().String("sidebar-position", "left", "projects sidebar side (left or right)")
	tuiCmd.Flags().Int("page-lines", 0, "lines to jump on PgUp/PgDn (0 = full viewport)")
	tuiCmd.Flags().Int("scroll-lines", 1, "lines to jump on mouse wheel")
	tuiCmd.Flags().Bool("border", true, "draw pane borders")
	tuiCmd.Flags().String("case-mode", "smart", "search/filter case mode (sensitive, insensitive, or smart)")
	tuiCmd.Flags().Int("name-column-percent", 40, "name column width as percent of secrets pane (1-99)")
	tuiCmd.Flags().Bool("list-scrollbar", true, "show vertical scrollbar in secrets list")
	tuiCmd.Flags().Bool("sidebar-scrollbar", true, "show vertical scrollbar in projects sidebar")
	tuiCmd.Flags().Bool("autosave", true, "persist TUI setting changes automatically")
	tuiCmd.Flags().Bool("autoreload", true, "reload TUI settings when config file changes")
	tuiCmd.Flags().BoolVar(&utils.DebugTUI, "debug-tui", utils.DebugTUI, "log TUI messages to file")
	rootCmd.AddCommand(tuiCmd)
}
