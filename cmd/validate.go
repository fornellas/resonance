package main

import (
	"github.com/spf13/cobra"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	iResouresPkg "github.com/fornellas/resonance/internal/resources"
	"github.com/fornellas/resonance/log"
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
		defer hst.Close()

		logger.Info("🔍 Validating", "path", path, hst.Type(), hst)

		resources, err := iResouresPkg.LoadPath(ctx, path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		blueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, resources, hst)
		if err != nil {
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
