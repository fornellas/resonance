package resources

// ResourceMap holds references to various Resource for fast query.
type ResourceMap map[string]map[string]Resource

func NewResourceMap(resources Resources) ResourceMap {
	m := ResourceMap{}

	for _, resource := range resources {
		typeName := GetResourceTypeName(resource)
		_, ok := m[typeName]
		if !ok {
			m[typeName] = map[string]Resource{}
		}
		id := GetResourceId(resource)
		m[typeName][id] = resource
	}

	return m
}

// GetResourceWithSameId returns the Resource of the same type and id of the given resource,
// or nil if not present.
func (m ResourceMap) GetResourceWithSameTypeId(resource Resource) Resource {
	idMap, ok := m[GetResourceTypeName(resource)]
	if !ok {
		return nil
	}
	r, ok := idMap[GetResourceId(resource)]
	if !ok {
		return nil
	}
	return r
}

func (m ResourceMap) HasResourceWithSameTypeId(resource Resource) bool {
	return m.GetResourceWithSameTypeId(resource) != nil
}
