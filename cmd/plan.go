package main

import (
	"github.com/spf13/cobra"
)

var PlanCmd = &cobra.Command{
	Use:   "plan [flags] path",
	Short: "Plan changes required to apply resource files.",
	Long:  "Loads all resoures from .yaml files at path and craft a plan required to apply them.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// path := args[0]

		// ctx := cmd.Context()
		// logger := log.MustLogger(ctx)

		// hst, err := GetHost(ctx)
		// if err != nil {
		// 	logger.Error(err.Error())
		// 	Exit(1)
		// }
		// defer hst.Close()

		// resourceDefs, err := resource.LoadDir(ctx, hst, path)
		// if err != nil {
		// 	logger.Error(err.Error())
		// 	Exit(1)
		// }

		// latestSnapshot, err := hostState.GetLatestSnapshot(ctx)
		// if err != nil {
		// 	logger.Error(err.Error())
		// 	Exit(1)
		// }
		// if latestSnapshot != nil {
		// 	if err := latestSnapshot.Check(ctx, hst); err != nil {
		// 		logger.Error(err.Error())
		// 		Exit(1)
		// 	}
		// } else {
		// 	latestSnapshot, err = hostState.SaveSnapshot(resourceDefs.TypeNames())
		// 	if err := latestSnapshot.Check(ctx, hst); err != nil {
		// 		logger.Error(err.Error())
		// 		Exit(1)
		// 	}
		// }

		// TODO add delete resourecs to plan
		// for _, typeName := range latestSnapshot.TypeNames() {
		// 	if !resourceDefs.HasTypeName(typeName) {
		// 		resourceDefs.Prepend(resource.NewDestroyResourceDef(typeName))
		// 	}
		// }

		// TODO create graph & calculate action required for each node, as a function of
		// diff between latestSnapshot and resourceDefs
		// graph, err := resource.NewGraph(resourceDefs)
		// if err != nil {
		// 	logger.Error(err.Error())
		// 	Exit(1)
		// }
	},
}

func init() {
	AddHostFlags(PlanCmd)

	RootCmd.AddCommand(PlanCmd)
}
