package main

import (
	"os"

	"github.com/spf13/cobra"

	iResouresPkg "github.com/fornellas/resonance/internal/resources"
	"github.com/fornellas/resonance/log"
	resouresPkg "github.com/fornellas/resonance/resources"
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

		logger.Info("🔍 Validating", "path", path, "host", hst)

		var resources resouresPkg.Resources

		fileInfo, err := os.Stat(path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		if fileInfo.IsDir() {
			resources, err = iResouresPkg.LoadDir(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		} else {
			resources, err = iResouresPkg.LoadFile(ctx, path)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		}

		nodes, err := iResouresPkg.NewNodes(resources)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		if err := nodes.Update(ctx, hst); err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		logger.Info(
			"📦 Resources",
			"resources", iResouresPkg.Nodes(nodes).String(),
		)

		logger.Info("🎆 Validation successful")
	},
}

func init() {
	AddHostFlags(ValidateCmd)

	RootCmd.AddCommand(ValidateCmd)
}
