package resource

import (
	"fmt"
	"regexp"
)

// ResourceState holds information about a resource.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type ResourceState interface{}

type ResourceInstanceKey string

var resourceInstanceKeyRegexp = regexp.MustCompile(`^(.+)\[(.+)\]$`)

func (rik ResourceInstanceKey) GetNames() (ResourceName, InstanceName, error) {
	var resourceName ResourceName
	var instanceName InstanceName
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(rik))
	if len(matches) != 3 {
		return resourceName, instanceName, fmt.Errorf("%s does not match Type[Name] format", rik)
	}
	resourceName = ResourceName(matches[1])
	instanceName = InstanceName(matches[2])
	return resourceName, instanceName, nil
}

func GetResourceInstanceKey(resourceName ResourceName, instanceName InstanceName) ResourceInstanceKey {
	return ResourceInstanceKey(fmt.Sprintf("%s[%s]", resourceName, instanceName))
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
