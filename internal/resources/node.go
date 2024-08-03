package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/fornellas/resonance/host"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type Node struct {
	SingleResource resourcesPkg.SingleResource
	GroupResource  resourcesPkg.GroupResource
	GroupResources resourcesPkg.Resources
	requiredBy     []*Node
}

func NewSingleResourceNode(singleResource resourcesPkg.SingleResource) *Node {
	return &Node{
		SingleResource: singleResource,
	}
}

func NewGroupResourceNode(groupResource resourcesPkg.GroupResource) *Node {
	return &Node{
		GroupResource: groupResource,
	}
}

func (n *Node) String() string {
	if n.SingleResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			reflect.TypeOf(n.SingleResource).Elem().Name(), n.SingleResource.Name(),
		)
	}

	if n.GroupResource != nil {
		return fmt.Sprintf(
			"%s: %s",
			reflect.TypeOf(n.GroupResource).Elem().Name(), n.GroupResources.Names(),
		)
	}

	panic("bug: invalid state")
}

func (n *Node) AppendGroupResource(resource resourcesPkg.Resource) {
	if n.GroupResource == nil {
		panic("bug: can't add Resource to non GroupResource Node")
	}
	n.GroupResources = append(n.GroupResources, resource)
}

func (n *Node) AppendRequiredByNode(node *Node) {
	n.requiredBy = append(n.requiredBy, node)
}

func (n *Node) Update(ctx context.Context, hst host.Host) error {
	if n.SingleResource != nil {
		return n.SingleResource.Update(ctx, hst)
	}

	if n.GroupResource != nil {
		return n.GroupResource.Update(ctx, hst, n.GroupResources)
	}

	panic("bug: invalid state")
}

func (n *Node) MarshalYAML() (interface{}, error) {
	type marshalSchema struct {
		SingleResourceType string                      `yaml:"single_resource_type,omitempty"`
		SingleResource     resourcesPkg.SingleResource `yaml:"single_resource,omitempty"`
		GroupResourceType  string                      `yaml:"group_resource_type,omitempty"`
		GroupResourcesType string                      `yaml:"group_resources_type,omitempty"`
		GroupResources     resourcesPkg.Resources      `yaml:"group_resources,omitempty"`
	}

	var singleResourceType string
	if n.SingleResource != nil {
		singleResourceType = reflect.TypeOf(n.SingleResource).Elem().Name()
	}

	var groupResourceType string
	var groupResourcesType string
	if n.GroupResource != nil {
		groupResourceType = reflect.TypeOf(n.GroupResource).Elem().Name()
		groupResourcesType = reflect.TypeOf(n.GroupResources[0]).Elem().Name()
	}

	return &marshalSchema{
		SingleResourceType: singleResourceType,
		SingleResource:     n.SingleResource,
		GroupResourceType:  groupResourceType,
		GroupResourcesType: groupResourcesType,
		GroupResources:     n.GroupResources,
	}, nil
}

// func (n *Node) UnmarshalYAML(node *yaml.Node) error {
// 	type unmarshalSchema struct {
// 		SingleResourceType string                      `yaml:"single_resource_type,omitempty"`
// 		SingleResource     yaml.Node `yaml:"single_resource,omitempty"`
// 		GroupResourceType  string                      `yaml:"group_resource_type,omitempty"`
// 		GroupResources     yaml.Node      `yaml:"group_resources,omitempty"`
// 	}

// 	node.KnownFields(true)
// 	node.Decode()

// }

type Nodes []*Node

func (ns Nodes) topologicalSsort() (Nodes, error) {
	dependantCount := map[*Node]int{}
	for _, node := range ns {
		if _, ok := dependantCount[node]; !ok {
			dependantCount[node] = 0
		}
		for _, requiredNode := range node.requiredBy {
			dependantCount[requiredNode]++
		}
	}

	noDependantsNodes := Nodes{}
	for _, node := range ns {
		if dependantCount[node] == 0 {
			noDependantsNodes = append(noDependantsNodes, node)
		}
	}

	sortedNodes := Nodes{}
	for len(noDependantsNodes) > 0 {
		node := noDependantsNodes[0]
		noDependantsNodes = noDependantsNodes[1:]
		sortedNodes = append(sortedNodes, node)
		for _, dependantNode := range node.requiredBy {
			dependantCount[dependantNode]--
			if dependantCount[dependantNode] == 0 {
				noDependantsNodes = append(noDependantsNodes, dependantNode)
			}
		}
	}

	if len(sortedNodes) != len(ns) {
		return nil, errors.New("unable to topological sort, cycle detected")
	}

	return sortedNodes, nil
}

func (ns Nodes) String() string {
	var buff bytes.Buffer
	for i, node := range ns {
		i++
		fmt.Fprintf(&buff, "%d. %s\n", i, node)
	}
	return buff.String()
}

func (h Nodes) Update(ctx context.Context, hst host.Host) error {
	for _, node := range h {
		if err := node.Update(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}

func newNodesFromRessources(resources resourcesPkg.Resources) Nodes {
	nodes := Nodes{}

	typeToNodeMap := map[string]*Node{}

	requiredNodes := Nodes{}
	pastRequiredNodes := map[*Node]bool{}

	for _, resource := range resources {
		var node *Node = nil

		typeName := reflect.TypeOf(resource).Elem().Name()

		if resourcesPkg.IsGroupResource(typeName) {
			var ok bool
			node, ok = typeToNodeMap[typeName]
			if !ok {
				node = NewGroupResourceNode(
					resourcesPkg.GetGroupResourceByName(typeName),
				)
				typeToNodeMap[typeName] = node
				nodes = append(nodes, node)
			}
			node.AppendGroupResource(resource)
		} else {
			singleResource, ok := resource.(resourcesPkg.SingleResource)
			if !ok {
				panic("bug: Resource is not SingleResource")
			}
			node = NewSingleResourceNode(singleResource)
			nodes = append(nodes, node)
		}

		var extraRequiredNode *Node = nil
		for _, requiredNode := range requiredNodes {
			if _, ok := pastRequiredNodes[node]; !ok {
				requiredNode.AppendRequiredByNode(node)
				pastRequiredNodes[requiredNode] = true
			} else {
				extraRequiredNode = requiredNode
			}
		}

		requiredNodes = Nodes{node}
		if extraRequiredNode != nil {
			requiredNodes = append(requiredNodes, extraRequiredNode)
		}
	}

	return nodes
}

func NewNodes(resources resourcesPkg.Resources) (Nodes, error) {
	nodes := newNodesFromRessources(resources)
	nodes, err := nodes.topologicalSsort()
	if err != nil {
		return nil, err
	}

	return (nodes), nil
}
