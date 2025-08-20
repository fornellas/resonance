package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"
)

type ResourceTypeValue struct {
	name string
}

func (r *ResourceTypeValue) String() string {
	return r.name
}

func (r *ResourceTypeValue) Set(name string) error {
	panic("TODO")
}
func (r *ResourceTypeValue) Type() string {
	panic("TODO")
}

func (r *ResourceTypeValue) Reset() {
	r.name = ""
}

var resourceTypeValue = &ResourceTypeValue{}

var resourceIds = []string{}
var defaultResourceNames = []string{}

var InspectCmd = &cobra.Command{
	Use:   "inspect [flags]",
	Short: "Inspect resources from host",
	Long:  "Load resources from host and print them to stdout.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "🔎 Inspect")

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

		panic("TODO")
	},
}

func init() {
	AddHostFlags(InspectCmd)

	InspectCmd.Flags().Var(resourceTypeValue, "resource-type", "The resource type")
	InspectCmd.MarkFlagRequired("resource-type")

	InspectCmd.Flags().StringSliceVar(&resourceIds, "resource-ids", []string{}, "Ids of resources")
	InspectCmd.MarkFlagRequired("resource-ids")

	RootCmd.AddCommand(InspectCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		resourceTypeValue.Reset()
		resourceIds = defaultResourceNames
	})
}
