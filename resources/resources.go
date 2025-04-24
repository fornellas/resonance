package resources

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Resources []Resource

func (r Resources) Ids() string {
	ids := make([]string, len(r))
	for i, resource := range r {
		ids[i] = GetResourceId(resource)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (r Resources) Validate() error {
	typeNameMap := map[string]bool{}
	for _, resource := range r {
		typeName := fmt.Sprintf("%s:%s", reflect.TypeOf(resource).Name(), GetResourceId(resource))
		if _, ok := typeNameMap[typeName]; ok {
			return fmt.Errorf("duplicated resource %s", typeName)
		}

		if err := ValidateResource(resource); err != nil {
			return fmt.Errorf("resource %s is invalid: %s", typeName, err.Error())
		}
	}

	return nil
}

func (r Resources) MarshalYAML() (any, error) {
	type MarshalSchema []map[string]Resource

	resources := make(MarshalSchema, len(r))

	for i, resource := range r {
		resourceMap := map[string]Resource{}
		typeName := reflect.TypeOf(resource).Elem().Name()
		resourceMap[typeName] = resource
		resources[i] = resourceMap
	}

	return resources, nil
}

func (r *Resources) UnmarshalYAML(node *yaml.Node) error {
	type UnmarshalSchema []map[string]yaml.Node

	resources := UnmarshalSchema{}

	node.KnownFields(true)
	err := node.Decode(&resources)
	if err != nil {
		return fmt.Errorf("line %d: %s", node.Line, err.Error())
	}

	*r = make(Resources, len(resources))

	for i, m := range resources {
		if len(m) != 1 {
			return errors.New("YAML contents does not reflect schema (bug?)")
		}
		for typeName, node := range m {
			resource := GetResourceByTypeName(typeName)
			node.KnownFields(true)
			err := node.Decode(resource)
			if err != nil {
				return fmt.Errorf("line %d: %s", node.Line, err.Error())
			}
			(*r)[i] = resource
		}
	}

	return nil
}

// NewResourcesWithSameIds is analog to NewResourceCopyWithOnlyId
func NewResourcesWithSameIds(resources Resources) Resources {
	nr := make(Resources, len(resources))

	for i, r := range resources {
		nr[i] = NewResourceWithSameId(r)
	}

	return nr
}

// GetResourcesYaml returns a string representing the resource as Yaml.
func GetResourcesYaml(resources Resources) string {
	bs, err := yaml.Marshal(resources)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(bs), "\n")
}
