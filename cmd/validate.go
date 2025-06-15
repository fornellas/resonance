package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	blueprintPkg "github.com/fornellas/resonance/blueprint"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate [flags] [file|dir]",
	Short: "Validates resource files.",
	Long:  "Loads all resoures from yaml files, validating whether they are ok.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "🔍 Validate", "path", path)

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				logger.Error("failed to close host", "error", err)
			}
		}()
		ctx, _ = log.MustWithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		resources, err := resourcesPkg.LoadPath(ctx, path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		blueprint, err := blueprintPkg.NewBlueprintFromResources("validate", resources)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		if err := blueprint.Resolve(ctx, host); err != nil {
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
