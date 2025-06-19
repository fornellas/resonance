package main

import (
	"errors"
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

		var retErr error
		defer func() {
			if retErr != nil {
				logger.Error("Failed", "err", retErr)
				Exit(1)
			}
		}()

		host, ctx, err := GetHost(ctx)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get host: %w", err))
			return
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close host: %w", err))
			}
		}()
		ctx, _ = log.MustWithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		resources, err := resourcesPkg.LoadPath(ctx, path)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to load resources: %w", err))
			return
		}

		blueprint, err := blueprintPkg.NewBlueprintFromResources("validate", resources)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to create Blueprint: %w", err))
			return
		}
		if err := blueprint.Resolve(ctx, host); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to resolve Blueprint: %w", err))
			return
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
