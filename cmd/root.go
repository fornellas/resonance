package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	slogxtCobra "github.com/fornellas/slogxt/cobra"
	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

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

		ctx := log.WithLogger(
			cmd.Context(),
			slogxtCobra.GetLogger(cmd.OutOrStderr()).
				With("((o)) Resonance", resonance.Version),
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

var resetFlagsFns = []func(){
	func() { slogxtCobra.Reset() },
}

func ResetFlags() {
	for _, resetFlagFn := range resetFlagsFns {
		resetFlagFn()
	}
}

func init() {
	slogxtCobra.AddLoggerFlags(RootCmd)
}
