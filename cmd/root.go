package main

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/log"
)

var logLevelStr string
var defaultLogLevelStr = "info"
var forceColor bool
var defaultForceColor = false

var RootCmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if forceColor {
			color.NoColor = false
		}
		cmd.SetContext(log.SetLoggerValue(
			cmd.Context(), cmd.OutOrStderr(), logLevelStr, Exit,
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
}

func init() {
	RootCmd.PersistentFlags().StringVarP(
		&logLevelStr, "log-level", "l", defaultLogLevelStr,
		"Logging level",
	)

	RootCmd.PersistentFlags().BoolVarP(
		&forceColor, "force-color", "", defaultForceColor,
		"Force colored output",
	)

	RootCmd.AddCommand(ValidateCmd)

	RootCmd.AddCommand(VersionCmd)
}
