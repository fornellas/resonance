package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/fornellas/resonance"
	"github.com/fornellas/resonance/log"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

var logLevelValue = NewLogLevelValue()

var logHandlerValue = NewLogHandlerValue()

var defaultLogHandlerAddSource = false
var logHandlerAddSource = defaultLogHandlerAddSource

var defaultLogHandlerConsoleTime = false
var logHandlerConsoleTime = defaultLogHandlerConsoleTime

var RootCmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Inspired by https://github.com/spf13/viper/issues/671#issuecomment-671067523
		v := viper.New()
		v.SetEnvPrefix("RESONANCE")
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && v.IsSet(f.Name) {
				cmd.Flags().Set(f.Name, fmt.Sprintf("%v", v.Get(f.Name)))
			}
		})

		handler := logHandlerValue.GetHandler(
			cmd.OutOrStderr(),
			LogHandlerValueOptions{
				Level:       logLevelValue.Level(),
				AddSource:   logHandlerAddSource,
				ConsoleTime: logHandlerConsoleTime,
			},
		)
		logger := slog.New(handler).With("version", resonance.Version)
		ctx := cmd.Context()
		ctx = log.WithLogger(
			ctx,
			logger,
		)
		cmd.SetContext(ctx)
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.MustLogger(cmd.Context())
		if err := cmd.Help(); err != nil {
			logger.Error("failed to display help", "error", err)
			Exit(1)
		}
	},
}

var resetFlagsFns []func()

func ResetFlags() {
	logLevelValue.Reset()

	logHandlerValue.Reset()

	logHandlerAddSource = defaultLogHandlerAddSource

	logHandlerConsoleTime = defaultLogHandlerConsoleTime

	for _, resetFlagFn := range resetFlagsFns {
		resetFlagFn()
	}
}

func init() {
	RootCmd.PersistentFlags().VarP(logLevelValue, "log-level", "l", "Logging level")

	RootCmd.PersistentFlags().VarP(logHandlerValue, "log-handler", "", "Logging handler")

	RootCmd.PersistentFlags().BoolVarP(
		&logHandlerAddSource, "log-handler-add-source", "", defaultLogHandlerAddSource,
		"Include source code position of the log statement when logging",
	)

	RootCmd.PersistentFlags().BoolVarP(
		&logHandlerConsoleTime, "log-handler-console-time", "", defaultLogHandlerConsoleTime,
		"Enable time for console handler",
	)
}
