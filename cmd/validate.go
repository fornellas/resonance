package main

import (
	"github.com/spf13/cobra"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	"github.com/fornellas/resonance/log"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate [flags] [file|dir]",
	Short: "Validates resource files.",
	Long:  "Loads all resoures from yaml files, validating whether they are ok.",
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
		defer hst.Close(ctx)

		logger.Info("🔍 Validating", "path", path, "host", hst)

		resources, err := resourcesPkg.LoadPath(ctx, path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		blueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, "validate", resources)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		if err := blueprint.Resolve(ctx, hst); err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		logger.Info(
			"🧩 Blueprint",
			"resources", blueprint.String(),
		)

		logger.Info("🎆 Validation successful")
	},
}

func init() {
	AddHostFlags(ValidateCmd)

	RootCmd.AddCommand(ValidateCmd)
}
