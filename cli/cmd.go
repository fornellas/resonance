package cli

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/apply"
	"github.com/fornellas/resonance/cli/check"
	"github.com/fornellas/resonance/cli/destroy"
	"github.com/fornellas/resonance/cli/plan"
	"github.com/fornellas/resonance/cli/refresh"
	"github.com/fornellas/resonance/log"
)

var logLevelStr string

var Cmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SetContext(log.SetLoggerValue(cmd.Context(), logLevelStr))
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

	Cmd.AddCommand(apply.Cmd)
	Cmd.AddCommand(check.Cmd)
	Cmd.AddCommand(destroy.Cmd)
	Cmd.AddCommand(plan.Cmd)
	Cmd.AddCommand(refresh.Cmd)
}
