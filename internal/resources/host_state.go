package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"

	"github.com/fornellas/resonance/host"
	resourcesPkg "github.com/fornellas/resonance/resources"
)

type Node struct {
	SingleResource resourcesPkg.SingleResource `yaml:"single_resource,omitempty"`
	GroupResources resourcesPkg.Resources      `yaml:"resources,omitempty"`
	GroupResource  resourcesPkg.GroupResource  `yaml:"group_resource,omitempty"`
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
			"%s:%s",
			reflect.TypeOf(n.SingleResource).Name(), n.SingleResource.Name(),
		)
	}

	if n.GroupResource != nil {
		return fmt.Sprintf(
			"%s:%s",
			reflect.TypeOf(n.GroupResource).Name(), n.GroupResources.Names(),
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

type Nodes []*Node

func NewNodes(resources resourcesPkg.Resources) Nodes {
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

func (ns Nodes) TopologicalSsort() (Nodes, error) {
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

// Returns a Graphviz graph of all nodes.
func (ns Nodes) Graphviz() (string, error) {
	g := graphviz.New()
	defer g.Close()

	graph, err := g.Graph()
	if err != nil {
		return "", err
	}
	defer graph.Close()

	gNodeMap := map[string]*cgraph.Node{}
	for _, node := range ns {
		name := node.String()
		gNode, err := graph.CreateNode(name)
		if err != nil {
			return "", err
		}
		gNodeMap[name] = gNode
	}

	for _, node := range ns {
		gNode := gNodeMap[node.String()]
		for _, toNode := range node.requiredBy {
			toGNode := gNodeMap[toNode.String()]
			_, err := graph.CreateEdge("required_by", gNode, toGNode)
			if err != nil {
				return "", err
			}
		}
	}

	var buf bytes.Buffer
	if err := g.Render(graph, "dot", &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type HostState Nodes

func NewHostState(resources resourcesPkg.Resources) (HostState, error) {
	nodes := NewNodes(resources)
	nodes, err := nodes.TopologicalSsort()
	if err != nil {
		return nil, err
	}

	return (HostState)(nodes), nil
}

func (h HostState) Update(ctx context.Context, hst host.Host) error {
	for _, node := range h {
		if err := node.Update(ctx, hst); err != nil {
			return err
		}
	}
	return nil
}
