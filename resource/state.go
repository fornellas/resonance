package resource

import "fmt"

// ResourceState holds information about a resource.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type ResourceState interface{}

type ResourceInstanceKey string

func GetResourceInstanceKey(ResourceName ResourceName, InstanceName string) ResourceInstanceKey {
	return ResourceInstanceKey(fmt.Sprintf("%s[%s]", ResourceName, InstanceName))
}

type StateData map[ResourceInstanceKey]ResourceState

func (sd StateData) Merge(stateData StateData) {
	for resourceInstanceKey, resourceState := range stateData {
		if _, ok := sd[resourceInstanceKey]; ok {
			panic(fmt.Sprintf("duplicated resource instance %s", resourceInstanceKey))
		}
		sd[resourceInstanceKey] = resourceState
	}
}
