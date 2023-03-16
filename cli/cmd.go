package cli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/apply"
	"github.com/fornellas/resonance/cli/version"
	"github.com/fornellas/resonance/log"
)

var ExitFunc func(int)

var forceColor bool
var logLevelStr string

var Cmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		color.NoColor = !forceColor
		cmd.SetContext(log.SetLoggerValue(
			cmd.Context(), cmd.OutOrStderr(), logLevelStr, ExitFunc,
		))
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.GetLogger(cmd.Context())
		if err := cmd.Help(); err != nil {
			logger.Fatal(err)
		}
	},
}

func init() {
	Cmd.PersistentFlags().StringVarP(
		&logLevelStr, "log-level", "l", "info",
		"Logging level",
	)
	Cmd.PersistentFlags().BoolVarP(
		&forceColor, "force-color", "", false,
		"Force colored output",
	)

	Cmd.AddCommand(apply.Cmd)
	Cmd.AddCommand(version.Cmd)
}
