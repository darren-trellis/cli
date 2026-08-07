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
		cfg.Theme = cmd.Flag("theme").Value.String()
		configuration.TUISetTheme(cfg.Theme)
	}
	if cmd.Flags().Changed("sidebar-width") {
		cfg.SidebarWidth, _ = cmd.Flags().GetInt("sidebar-width")
		configuration.TUISetSidebarWidth(cfg.SidebarWidth)
	}
	if cmd.Flags().Changed("page-lines") {
		cfg.PageLines, _ = cmd.Flags().GetInt("page-lines")
		configuration.TUISetPageLines(cfg.PageLines)
	}
	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = "default"
	}

	tuiApp.Start(localConfig, cfg)
}

func init() {
	tuiCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	tuiCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	tuiCmd.Flags().String("theme", "default", "TUI theme (default, cool, warm, mono, or a teleminator theme like catppuccin)")
	tuiCmd.Flags().Int("sidebar-width", 0, "projects sidebar width in columns (0 = auto)")
	tuiCmd.Flags().Int("page-lines", 0, "lines to jump on PgUp/PgDn (0 = full viewport)")
	tuiCmd.Flags().BoolVar(&utils.DebugTUI, "debug-tui", utils.DebugTUI, "log TUI messages to file")
	rootCmd.AddCommand(tuiCmd)
}
