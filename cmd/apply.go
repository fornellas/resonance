package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance"
	blueprintPkg "github.com/fornellas/resonance/blueprint"
	planPkg "github.com/fornellas/resonance/plan"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

var ApplyCmd = &cobra.Command{
	Use:   "apply [flags] [file|dir]",
	Short: "Apply resources.",
	Long:  "Load resources from file/dir and apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "✏️ Apply", "path", path)

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

		store, storeConfig, err := GetStore(host)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get store: %w", err))
			return
		}
		ctx, logger = log.MustWithAttrs(ctx, "store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))

		storelogWriterCloser, err := store.GetLogWriterCloser(ctx, "apply")
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get store log writer: %w", err))
			return
		}
		defer func() {
			if err := storelogWriterCloser.Close(); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close store log: %w", err))
			}
		}()

		logHandler := logger.Handler()
		storeHandler := log.NewTerminalLineHandler(storelogWriterCloser, &log.TerminalHandlerOptions{
			HandlerOptions: slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
			TimeLayout: time.RFC3339,
			NoColor:    true,
		}).
			WithAttrs([]slog.Attr{slog.String("version", resonance.Version)}).
			WithGroup("✏️ Apply").WithAttrs([]slog.Attr{slog.String("path", path)}).
			WithAttrs([]slog.Attr{slog.String("host", fmt.Sprintf("%s => %s", host.Type(), host.String()))}).
			WithAttrs([]slog.Attr{slog.String("store", fmt.Sprintf("%s %s", storeValue.String(), storeConfig))})
		logger = slog.New(log.NewMultiHandler(logHandler, storeHandler))
		ctx = log.WithLogger(ctx, logger)
		cmd.SetContext(ctx)

		var targetResources resourcesPkg.Resources
		{
			var err error
			targetResources, err = resourcesPkg.LoadPath(ctx, path)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to load resources: %w", err))
				return
			}
			_, logger := log.MustWithGroup(ctx, "📚 Target resources")
			for _, resource := range targetResources {
				logger.Debug(resourcesPkg.GetResourceTypeName(resource), "yaml", resourcesPkg.GetResourceYaml(resource))
			}
		}

		var plan planPkg.Plan
		var targetBlueprint *blueprintPkg.Blueprint
		var lastBlueprint *blueprintPkg.Blueprint
		plan, targetBlueprint, lastBlueprint, err = planPkg.CraftPlan(ctx, host, store, targetResources)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to plan: %w", err))
			return
		}

		{
			ctx, _ := log.MustWithGroup(ctx, "💾 Saving target Blueprint")
			hasTargetBlueprint, err := store.HasTargetBlueprint(ctx)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed save blueprint: %w", err))
				return
			}
			if hasTargetBlueprint {
				retErr = errors.Join(retErr, fmt.Errorf("a previous apply was interrupted"))
				return
			} else {
				if err := store.SaveTargetBlueprint(ctx, targetBlueprint); err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("failed save target blueprint: %w", err))
					return
				}
			}
		}

		if err := plan.Apply(ctx, host); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to apply: %w", err))
			return
		}

		{
			ctx, _ := log.MustWithGroup(ctx, "🧹 State cleanup")

			targetResourcesMap := resourcesPkg.NewResourceMap(targetResources)
			for _, lastResource := range lastBlueprint.Resources() {
				if !targetResourcesMap.HasResourceWithSameTypeId(lastResource) {
					if err := store.DeleteOriginalResource(ctx, lastResource); err != nil {
						retErr = errors.Join(retErr, fmt.Errorf("failed to delete original resource: %w", err))
						return
					}
				}
			}

			if err := store.SaveLastBlueprint(ctx, targetBlueprint); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to save last blueprint: %w", err))
				return
			}

			if err := store.DeleteTargetBlueprint(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to delete target blueprint: %w", err))
				return
			}
		}

		logger.Info("🎆 Apply successful")
	},
}

func init() {
	AddHostFlags(ApplyCmd)

	AddStoreFlags(ApplyCmd)

	RootCmd.AddCommand(ApplyCmd)
}
