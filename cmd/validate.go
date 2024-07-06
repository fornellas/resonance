package main

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/internal/resource"
	"github.com/fornellas/resonance/log"
)

var ValidateCmd = &cobra.Command{
	Use:   "validate [flags] resources_root",
	Short: "Validates resource files.",
	Long:  "Loads all resoures from .yaml files at resources_root validating whether they are ok.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		logger := log.GetLogger(ctx)

		root := args[0]

		hst, err := GetHost(ctx)
		if err != nil {
			logger.Fatal(err)
		}
		defer hst.Close()

		_, err = resource.LoadDir(ctx, hst, root)
		if err != nil {
			logger.Fatal(err)
		}

		logger.Info("🎆 Validation successful")
	},
}

func init() {
	AddHostFlags(ValidateCmd)

	RootCmd.AddCommand(ValidateCmd)
}
