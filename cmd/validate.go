package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/recipe"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate PATH [PATH ...]",
	Short: "Validate recipes.",
	Long: "Validate recipes.\n\n" +
		"Each PATH is either a recipe file, or a directory, which is recursed " +
		"into for files ending in .yaml. All files found are parsed and merged " +
		"into a single recipe.",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "🔎 Validate")

		if _, err := recipe.LoadPaths(ctx, args...); err != nil {
			logger.Error("Failed", "err", err)
			Exit(1)
		}

		logger.Info("Recipe is valid", "paths", args)
	},
}

func init() {
	RootCmd.AddCommand(ValidateCmd)
}
