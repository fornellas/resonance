package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/internal/resource"
	"github.com/fornellas/resonance/log"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate [flags] path",
	Short: "Validates resource files.",
	Long:  "Loads all resoures from .yaml files at path validating whether they are ok.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()

		logger := log.MustLogger(ctx)

		hst, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer hst.Close()

		logger.Info("⚙️ Validating", "path", path, "host", hst)

		resourceDefs, err := resource.LoadDir(ctx, hst, path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		logger.Info("🎆 Validation successful", "resources", len(resourceDefs))
	},
}

func init() {
	AddHostFlags(ValidateCmd)

	RootCmd.AddCommand(ValidateCmd)
}
