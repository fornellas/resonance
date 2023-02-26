package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host"
)

// Name is a name that globally uniquely identifies a resource instance of a given type.
// Eg: for File type a Name would be the file absolute path such as /etc/issue.
type Name string

func (n Name) String() string {
	return string(n)
}

// Instance holds parameters for a resource instance.
type Instance struct {
	Name Name `yaml:"name"`
	// Parameters for a resource instance. This is specific for each resource type.
	Parameters yaml.Node `yaml:"parameters"`
}

// State holds information about a resource state. This is specific for each resource type.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type State interface{}

// ManageableResource defines an interface for managing resource state.
type ManageableResource interface {
	// MergeApply informs whether all resources from the same type are to
	// be always merged together when applying.
	// When true, Apply is called only once, with all instances.
	// When false, Apply is called one time for each instance.
	MergeApply() bool

	// GetDesiredState return desired state for given parameters.
	GetDesiredState(parameters yaml.Node) (State, error)

	// GetState returns current resource state from host without any side effects.
	GetState(ctx context.Context, host host.Host, name Name) (State, error)

	// Apply confiugres the resource at host to given instances state.
	// Must be idempotent.
	Apply(ctx context.Context, host host.Host, instances []Instance) error

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistant changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, host host.Host, name Name) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, host host.Host, name Name) error
}

// Type is the name of the resource.
// Must match resource's reflect.Type.Name().
type Type string

func (t Type) String() string {
	return string(t)
}

var TypeToManageableResource = map[Type]ManageableResource{}

// ManageableResource returns an instance for the resource type.
func (t Type) ManageableResource() (ManageableResource, error) {
	manageableResource, ok := TypeToManageableResource[t]
	if !ok {
		types := []string{}
		for tpe := range TypeToManageableResource {
			types = append(types, tpe.String())
		}
		return nil, fmt.Errorf("unknown resource type '%s'; valid types: %s", t, strings.Join(types, ", "))
	}
	return manageableResource, nil
}

// TypeName is a string that identifies a resource type and name.
// Eg: File[/etc/issue].
type TypeName string

func (tn TypeName) String() string {
	return string(tn)
}

var resourceInstanceKeyRegexp = regexp.MustCompile(`^(.+)\[(.+)\]$`)

// GetTypeName returns the Type and Name.
func (tn TypeName) GetTypeName() (Type, Name, error) {
	var tpe Type
	var name Name
	matches := resourceInstanceKeyRegexp.FindStringSubmatch(string(tn))
	if len(matches) != 3 {
		return tpe, name, fmt.Errorf("%s does not match Type[Name] format", tn)
	}
	tpe = Type(matches[1])
	name = Name(matches[2])
	return tpe, name, nil
}

func (tn TypeName) Type() (Type, error) {
	tpe, _, err := tn.GetTypeName()
	return tpe, err
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) ManageableResource() (ManageableResource, error) {
	tpe, _, err := tn.GetTypeName()
	if err != nil {
		return nil, err
	}
	return tpe.ManageableResource()
}

// NewTypeName creates a new TypeName.
func NewTypeName(tpe Type, name Name) TypeName {
	return TypeName(fmt.Sprintf("%s[%s]", tpe, name))
}

// HostState is the schema used to save/load state for all resources for a host.
type HostState map[TypeName]State

// Merge appends received HostState.
func (hs HostState) Merge(stateData HostState) {
	for resourceInstanceKey, resourceState := range stateData {
		if _, ok := hs[resourceInstanceKey]; ok {
			panic(fmt.Sprintf("duplicated resource instance %s", resourceInstanceKey))
		}
		hs[resourceInstanceKey] = resourceState
	}
}

func (hs HostState) String() (string, error) {
	buffer := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buffer)
	if err := encoder.Encode(hs); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// ResourceDefinition is the schema used to declare a single resource within a file.
type ResourceDefinition struct {
	TypeName   TypeName  `yaml:"resource"`
	Parameters yaml.Node `yaml:"parameters"`
}

func (rd ResourceDefinition) String() string {
	return rd.TypeName.String()
}

// ResourceBundle is the schema used to declare multiple resources at a single file.
type ResourceBundle []ResourceDefinition

// ResourceBundles holds all resources definitions for a host.
type ResourceBundles []ResourceBundle

// GetHostState reads and return the state from all resource definitions.
func (rbs ResourceBundles) GetHostState(ctx context.Context, host host.Host) (HostState, error) {
	hostState := HostState{}

	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			tpe, name, err := resourceDefinition.TypeName.GetTypeName()
			if err != nil {
				return hostState, err
			}
			resource, err := tpe.ManageableResource()
			if err != nil {
				return hostState, err
			}
			state, err := resource.GetState(ctx, host, name)
			if err != nil {
				return hostState, fmt.Errorf("%s: failed to read state: %w", resourceDefinition.TypeName, err)
			}

			hostState[resourceDefinition.TypeName] = state
		}
	}

	return hostState, nil
}

// GetDesiredHostState returns the desired HostState for all resources.
func (rbs ResourceBundles) GetDesiredHostState() (HostState, error) {
	hostState := HostState{}

	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			tpe, _, err := resourceDefinition.TypeName.GetTypeName()
			if err != nil {
				return hostState, err
			}
			resource, err := tpe.ManageableResource()
			if err != nil {
				return hostState, err
			}
			state, err := resource.GetDesiredState(resourceDefinition.Parameters)
			if err != nil {
				return hostState, fmt.Errorf("%s: failed get desired state: %w", resourceDefinition.TypeName, err)
			}

			hostState[resourceDefinition.TypeName] = state
		}
	}

	return hostState, nil
}

// Action to be executed for a given Node.
type Action int

const (
	ActionUnknown Action = iota
	ActionNone
	ActionSkip
	ActionApply
	ActionDestroy
)

// Node from a Digraph
type Node struct {
	ResourceDefinitions []ResourceDefinition
	PrerequisiteFor     []*Node
	Action              Action
	Refresh             bool
	// visited             bool
}

// Name of the node.
func (n Node) Name() string {
	if len(n.ResourceDefinitions) == 1 {
		return string(n.ResourceDefinitions[0].TypeName)
	} else if len(n.ResourceDefinitions) > 1 {
		var tpeStr string
		names := []string{}
		for _, resourceDefinition := range n.ResourceDefinitions {
			typeName := resourceDefinition.TypeName
			tpe, name, err := typeName.GetTypeName()
			if err != nil {
				panic(fmt.Sprintf("invalid node: bad TypeName: %s", typeName))
			}
			tpeStr = tpe.String()
			names = append(names, name.String())
		}
		return fmt.Sprintf("%s[%s]", tpeStr, strings.Join(names, ","))
	} else {
		panic("invalid node: empty ResourceDefinitions")
	}
}

// Digraph is a directed graph which contains the plan for applying resources to a host.
type Digraph []*Node

// Graphviz returns a DOT directed graph containing the apply plan.
func (dg Digraph) Graphviz() string {
	str := "digraph resonance {\n"
	str += "  subgraph cluster_Action {\n"
	str += "    label=Action\n"
	str += "    node [color=gray4] None\n"
	str += "    node [color=yellow4] Skip\n"
	str += "    node [color=green4] Apply\n"
	str += "    node [color=red4] Destroy\n"
	str += "    node [color=default style=dashed] Refresh\n"
	str += "    None -> Skip  [style=invis]\n"
	str += "    Skip -> Apply  [style=invis]\n"
	str += "    Apply -> Destroy  [style=invis]\n"
	str += "    Destroy -> Refresh  [style=invis]\n"
	str += "  }\n"
	for _, node := range dg {
		var style, color string
		if node.Refresh {
			style = "dashed"
		} else {
			style = "solid"
		}
		switch node.Action {
		case ActionNone:
			color = "gray4"
		case ActionSkip:
			color = "yellow4"
		case ActionApply:
			color = "green4"
		case ActionDestroy:
			color = "red4"
		default:
			panic(fmt.Sprintf("unexpected Node.Action %q", node.Action))
		}
		str += fmt.Sprintf("  node [style=%s color=%s] \"%s\";\n", style, color, node.Name())
	}
	for _, node := range dg {
		for _, dependantNode := range node.PrerequisiteFor {
			str += fmt.Sprintf("  \"%s\" -> \"%s\";\n", node.Name(), dependantNode.Name())
		}
	}
	str += "}\n"
	return str
}

// GetDigraph calculates the plan and returns it in the form of a Digraph
func (rbs ResourceBundles) GetDigraph(savedHostState, desiredHostState, currentHostState HostState) (Digraph, error) {
	digraph := Digraph{}
	mergedNodes := map[Type]*Node{}
	for _, resourceBundle := range rbs {
		resourceBundleNodes := []*Node{}
		for _, resourceDefinition := range resourceBundle {
			resourceBundleNodes = append(resourceBundleNodes, &Node{
				ResourceDefinitions: []ResourceDefinition{resourceDefinition},
			})
		}
		var lastNode *Node
		refresh := false
		for _, node := range resourceBundleNodes {
			if lastNode != nil {
				lastNode.PrerequisiteFor = append(lastNode.PrerequisiteFor, node)
			}
			typeName := node.ResourceDefinitions[0].TypeName
			manageableResource, err := typeName.ManageableResource()
			if err != nil {
				return nil, err
			}
			stateEqual := reflect.DeepEqual(desiredHostState[typeName], currentHostState[typeName])
			if manageableResource.MergeApply() {
				node.Action = ActionSkip
				tpe, err := typeName.Type()
				if err != nil {
					return nil, err
				}
				mergedNode, ok := mergedNodes[tpe]
				if !ok {
					mergedNode = &Node{}
					mergedNodes[tpe] = mergedNode
					digraph = append(digraph, mergedNode)
				}
				mergedNode.ResourceDefinitions = append(mergedNode.ResourceDefinitions, node.ResourceDefinitions...)
				mergedNode.PrerequisiteFor = append(mergedNode.PrerequisiteFor, node)
				if !stateEqual {
					mergedNode.Action = ActionApply
				}
			} else {
				if stateEqual {
					node.Action = ActionNone
				} else {
					node.Action = ActionApply
					refresh = true
				}
			}
			node.Refresh = refresh
			lastNode = node
		}
		digraph = append(digraph, resourceBundleNodes...)
	}

	// TODO topological sort

	// TODO append nodes to destroy
	// toDestroyTypeNames := []TypeName{}
	// for savedTypeName := range savedHostState {
	// 	if _, ok := desiredHostState[savedTypeName]; !ok {
	// 		toDestroyTypeNames = append(toDestroyTypeNames, savedTypeName)
	// 	}
	// }

	return digraph, nil
}

func loadResourceBundle(ctx context.Context, path string) (ResourceBundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return ResourceBundle{}, fmt.Errorf("failed to load resource definitions: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	resourceBundle := ResourceBundle{}

	for {
		docResourceBundle := ResourceBundle{}
		if err := decoder.Decode(&docResourceBundle); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ResourceBundle{}, fmt.Errorf("failed to load resource definitions: %s: %w", path, err)
		}
		resourceBundle = append(resourceBundle, docResourceBundle...)
	}

	return resourceBundle, nil
}

// LoadResourceBundles loads resource definitions from all given Yaml file paths.
// Each file must have the schema defined by ResourceBundle.
func LoadResourceBundles(ctx context.Context, paths []string) ResourceBundles {
	resourceBundles := ResourceBundles{}
	for _, path := range paths {
		resourceBundle, err := loadResourceBundle(ctx, path)
		if err != nil {
			logrus.Fatal(err)
		}
		resourceBundles = append(resourceBundles, resourceBundle)
	}
	return resourceBundles
}
