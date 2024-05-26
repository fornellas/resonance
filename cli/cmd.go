package cli

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	// "github.com/fornellas/resonance/cli/apply"
	// "github.com/fornellas/resonance/cli/check"
	// "github.com/fornellas/resonance/cli/graph"
	// "github.com/fornellas/resonance/cli/lib"
	// "github.com/fornellas/resonance/cli/plan"
	// "github.com/fornellas/resonance/cli/refresh"
	// "github.com/fornellas/resonance/cli/rollback"
	// "github.com/fornellas/resonance/cli/show"
	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/cli/validate"
	"github.com/fornellas/resonance/cli/version"
	"github.com/fornellas/resonance/log"
)

var ExitFunc func(int) = func(code int) { os.Exit(code) }

var logLevelStr string
var defaultLogLevelStr = "info"
var forceColor bool
var defaultForceColor = false

var Cmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if forceColor {
			color.NoColor = false
		}
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

var resetFuncs []func()

func Reset() {
	logLevelStr = defaultLogLevelStr
	forceColor = defaultForceColor
	for _, resetFunc := range resetFuncs {
		resetFunc()
	}
	lib.Reset()
}

func init() {
	Cmd.PersistentFlags().StringVarP(
		&logLevelStr, "log-level", "l", defaultLogLevelStr,
		"Logging level",
	)
	Cmd.PersistentFlags().BoolVarP(
		&forceColor, "force-color", "", defaultForceColor,
		"Force colored output",
	)

	// Cmd.AddCommand(apply.Cmd)
	// resetFuncs = append(resetFuncs, apply.Reset)

	// Cmd.AddCommand(check.Cmd)
	// resetFuncs = append(resetFuncs, check.Reset)

	// Cmd.AddCommand(graph.Cmd)
	// resetFuncs = append(resetFuncs, graph.Reset)

	// Cmd.AddCommand(plan.Cmd)
	// resetFuncs = append(resetFuncs, plan.Reset)

	// Cmd.AddCommand(refresh.Cmd)
	// resetFuncs = append(resetFuncs, refresh.Reset)

	// Cmd.AddCommand(rollback.Cmd)
	// resetFuncs = append(resetFuncs, rollback.Reset)

	// Cmd.AddCommand(show.Cmd)
	// resetFuncs = append(resetFuncs, show.Reset)

	Cmd.AddCommand(validate.Cmd)
	resetFuncs = append(resetFuncs, validate.Reset)

	Cmd.AddCommand(version.Cmd)
	resetFuncs = append(resetFuncs, version.Reset)
}
