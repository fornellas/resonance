package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/fornellas/slogxt/log"

	resourcesPkg "github.com/fornellas/resonance/resources"
)

type ResourceTypeValue struct {
	name string
}

func (r *ResourceTypeValue) String() string {
	return r.name
}

func (r *ResourceTypeValue) Set(name string) error {
	resource := resourcesPkg.GetResourceByTypeName(name)
	if resource == nil {
		return fmt.Errorf("invalid resource type: %#v", name)
	}
	r.name = name
	return nil
}
func (r *ResourceTypeValue) Type() string {
	names := resourcesPkg.GetResourceTypeNames()
	sort.Strings(names)
	return fmt.Sprintf("[%s]", strings.Join(names, "|"))
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

		resourceType := resourceTypeValue.String()
		resources := resourcesPkg.Resources{}
		for _, id := range resourceIds {
			resources = append(resources, resourcesPkg.NewResourceFromTypeNameAndId(resourceType, id))
		}

		if resourcesPkg.IsGroupResource(resourceType) {
			groupResource := resourcesPkg.GetGroupResourceByTypeName(resourceType)
			if err := groupResource.Load(ctx, host, resources); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed load resources: %w", err))
				return
			}
		} else {
			for _, resource := range resources {
				singleResource := resource.(resourcesPkg.SingleResource)
				if err := singleResource.Load(ctx, host); err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("failed load resource: %w", err))
					return
				}
			}
		}

		yamlBytes, err := yaml.Marshal(resources)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed marshal yaml: %w", err))
			return
		}

		if _, err := os.Stdout.Write(yamlBytes); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed write: %w", err))
			return
		}
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
