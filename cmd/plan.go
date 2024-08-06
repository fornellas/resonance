package main

import (
	"github.com/spf13/cobra"

	blueprintPkg "github.com/fornellas/resonance/internal/blueprint"
	"github.com/fornellas/resonance/internal/diff"
	iResouresPkg "github.com/fornellas/resonance/internal/resources"
	storePkg "github.com/fornellas/resonance/internal/store"
	"github.com/fornellas/resonance/log"
)

var PlanCmd = &cobra.Command{
	Use:   "plan [flags] [file|dir]",
	Short: "Plan actions.",
	Long:  "Load resources from file/dir and plan which actions are required to apply them.",
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

		logger.Info("✏️ Planning", "path", path, "host", hst)

		resources, err := iResouresPkg.LoadPath(ctx, path)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		targetBlueprint, err := blueprintPkg.NewBlueprintFromResources(ctx, resources, hst)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		logger.Info(
			"🧩 Blueprint",
			"resources", targetBlueprint.String(),
		)

		store := storePkg.NewHostStore(hst)

		lastBlueprint, err := store.GetLastBlueprint(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		if lastBlueprint == nil {
			logger.Info("🚫 No previous Blueprint")
			lastBlueprint, err := targetBlueprint.Load(ctx, hst)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			if err := store.Save(ctx, lastBlueprint); err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
		} else {
			currentBlueprint, err := lastBlueprint.Load(ctx, hst)
			if err != nil {
				logger.Error(err.Error())
				Exit(1)
			}
			chunks := diff.DiffAsYaml(lastBlueprint, currentBlueprint)

			if chunks.HaveChanges() {
				logger.Error(
					"host state has changed since last time, can't proceed",
					"diff", chunks.String(),
				)
				Exit(1)
			}
		}

		planBlueprint, err := blueprintPkg.NewPlanBlueprint(ctx, lastBlueprint, targetBlueprint)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		logger.Info(
			"🧩 Blueprint",
			"resources", planBlueprint.String(),
		)

		logger.Info("🎆 Planning successful")
	},
}

func init() {
	AddHostFlags(PlanCmd)

	RootCmd.AddCommand(PlanCmd)
}
