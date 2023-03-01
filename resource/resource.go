package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
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

type CheckResult bool

func (cr CheckResult) String() string {
	if cr {
		return "✔️ "
	} else {
		return "🚫"
	}
}

// ManageableResource defines an interface for managing resource state.
type ManageableResource interface {
	// MergeApply informs whether all resources from the same type are to
	// be always merged together when applying.
	// When true, Apply is called only once, with all instances.
	// When false, Apply is called one time for each instance.
	MergeApply() bool

	// Check host for the state of instatnce. If changes are required, returns true,
	// otherwise, returns false.
	Check(ctx context.Context, hst host.Host, instance Instance) (CheckResult, error)

	// Apply confiugres the resource at host to given instances state.
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, instances []Instance) error

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistant changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, hst host.Host, name Name) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, hst host.Host, name Name) error
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

func (tn TypeName) typeName() (Type, Name, error) {
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
	tpe, _, err := tn.typeName()
	return tpe, err
}

func (tn TypeName) Name() (Name, error) {
	_, name, err := tn.typeName()
	return name, err
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) ManageableResource() (ManageableResource, error) {
	tpe, err := tn.Type()
	if err != nil {
		return nil, err
	}
	return tpe.ManageableResource()
}

// NewTypeName creates a new TypeName.
func NewTypeName(tpe Type, name Name) TypeName {
	return TypeName(fmt.Sprintf("%s[%s]", tpe, name))
}

// ResourceDefinition is the schema used to declare a single resource within a file.
type ResourceDefinition struct {
	TypeName   TypeName  `yaml:"resource"`
	Parameters yaml.Node `yaml:"parameters"`
}

func (rd ResourceDefinition) String() string {
	return rd.TypeName.String()
}

func (rd ResourceDefinition) ManageableResource() (ManageableResource, error) {
	return rd.TypeName.ManageableResource()
}

func (rd ResourceDefinition) Instance() (Instance, error) {
	name, err := rd.TypeName.Name()
	if err != nil {
		return Instance{}, err
	}
	return Instance{Name: name, Parameters: rd.Parameters}, nil
}

func (rd ResourceDefinition) Check(ctx context.Context, hst host.Host) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	manageableResource, err := rd.ManageableResource()
	if err != nil {
		return false, err
	}
	instance, err := rd.Instance()
	if err != nil {
		return false, err
	}
	logger.Debugf("Checking %s", rd.TypeName)
	check, err := manageableResource.Check(log.IndentLogger(ctx), hst, instance)
	if err != nil {
		return false, err
	}
	return check, nil
}

// ResourceBundle is the schema used to declare multiple resources at a single file.
type ResourceBundle []ResourceDefinition

// Action to be executed for a given Node.
type Action int

var actionNames = []string{
	"❓ Unknown",
	"✔️ OK",
	"⏩ Skip",
	"🔧 Apply",
	"💀 Destroy",
}

func (a Action) String() string {
	return actionNames[a]
}

const (
	ActionUnknown Action = iota
	ActionNone
	ActionSkip
	ActionApply
	ActionDestroy
)

// Node from a Plan
type Node struct {
	ResourceDefinitions []ResourceDefinition
	PrerequisiteFor     []*Node
	Action              Action
	Refresh             bool
}

func (n Node) ManageableResource() (ManageableResource, error) {
	tpe, err := n.ResourceDefinitions[0].TypeName.Type()
	if err != nil {
		return nil, err
	}
	manageableResource, err := tpe.ManageableResource()
	if err != nil {
		return nil, err
	}
	return manageableResource, nil
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
			tpe, err := typeName.Type()
			if err != nil {
				panic(fmt.Sprintf("invalid node: bad TypeName: %s", typeName))
			}
			tpeStr = tpe.String()
			name, err := typeName.Name()
			if err != nil {
				panic(fmt.Sprintf("invalid node: bad TypeName: %s", typeName))
			}
			names = append(names, name.String())
		}
		return fmt.Sprintf("%s[%s]", tpeStr, strings.Join(names, ","))
	} else {
		panic("invalid node: empty ResourceDefinitions")
	}
}

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan []*Node

// Graphviz returns a DOT directed graph containing the apply plan.
func (p Plan) Graphviz() string {
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
	for _, node := range p {
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
	for _, node := range p {
		for _, dependantNode := range node.PrerequisiteFor {
			str += fmt.Sprintf("  \"%s\" -> \"%s\";\n", node.Name(), dependantNode.Name())
		}
	}
	str += "}\n"
	return str
}

// topologicalSort sorts the nodes based on their prerequisites. If the graph has cycles, it returns
// error.
func (p Plan) topologicalSort() (Plan, error) {
	// Thanks ChatGPT :-D

	// 1. Create a map to store the in-degree of each node
	inDegree := make(map[*Node]int)
	for _, node := range p {
		inDegree[node] = 0
	}
	// 2. Calculate the in-degree of each node
	for _, node := range p {
		for _, prereq := range node.PrerequisiteFor {
			inDegree[prereq]++
		}
	}
	// 3. Create a queue to store nodes with in-degree 0
	queue := make([]*Node, 0)
	for _, node := range p {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	// 4. Initialize the result slice
	result := make(Plan, 0)
	// 5. Process nodes in the queue
	for len(queue) > 0 {
		// 5.1. Dequeue a node from the queue
		node := queue[0]
		queue = queue[1:]
		// 5.2. Add the node to the result
		result = append(result, node)
		// 5.3. Decrease the in-degree of each of its neighbors
		for _, neighbor := range node.PrerequisiteFor {
			inDegree[neighbor]--
			// 5.4. If the neighbor's in-degree is 0, add it to the queue
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
	// 6. Check if all nodes were visited
	if len(result) != len(p) {
		return nil, errors.New("the graph has cycles")
	}
	return result, nil
}

func (p Plan) actionNoneSkip(ctx context.Context, hst host.Host, node *Node) error {
	logger := log.GetLogger(ctx)

	manageableResource, err := node.ManageableResource()
	if err != nil {
		return err
	}

	if node.Refresh {
		logger.Infof("🔁 Refresh: %s", node.Name())
		for _, resourceDefinition := range node.ResourceDefinitions {
			if len(node.ResourceDefinitions) > 1 {
				logger.Infof("  Refreshing %s", resourceDefinition)
			}
			name, err := resourceDefinition.TypeName.Name()
			if err != nil {
				return err
			}
			if err := manageableResource.Refresh(ctx, hst, name); err != nil {
				return err
			}
		}
	} else {
		logger.Infof("%s: %s", node.Action, node.Name())
	}
	return nil
}

func (p Plan) actionApply(ctx context.Context, hst host.Host, node *Node) error {
	logger := log.GetLogger(ctx)

	manageableResource, err := node.ManageableResource()
	if err != nil {
		return err
	}

	logger.Infof("%s: %s", node.Action, node.Name())
	instances := []Instance{}
	for _, resourceDefinition := range node.ResourceDefinitions {
		name, err := resourceDefinition.TypeName.Name()
		if err != nil {
			return err
		}
		instances = append(instances, Instance{
			Name:       name,
			Parameters: resourceDefinition.Parameters,
		})
	}
	if err := manageableResource.Apply(ctx, hst, instances); err != nil {
		return err
	}

	return nil
}

func (p Plan) actionDestroy(ctx context.Context, hst host.Host, node *Node) error {
	logger := log.GetLogger(ctx)

	manageableResource, err := node.ManageableResource()
	if err != nil {
		return err
	}

	logger.Infof("%s: %s", node.Action, node.Name())
	for _, resourceDefinition := range node.ResourceDefinitions {
		if len(node.ResourceDefinitions) > 1 {
			logger.Infof("Destroying %s", resourceDefinition)
		}
		name, err := resourceDefinition.TypeName.Name()
		if err != nil {
			return err
		}
		if err := manageableResource.Destroy(ctx, hst, name); err != nil {
			return err
		}
	}
	return nil
}

// Apply required changes to host
func (p Plan) Apply(ctx context.Context, hst host.Host) error {
	for _, node := range p {
		switch node.Action {
		case ActionNone, ActionSkip:
			return p.actionNoneSkip(ctx, hst, node)
		case ActionApply:
			return p.actionApply(ctx, hst, node)
		case ActionDestroy:
			return p.actionDestroy(ctx, hst, node)
		default:
			panic(fmt.Sprintf("unexpected action %q", node.Action))
		}
	}
	return nil
}

// ResourceBundles holds all resources definitions for a host.
type ResourceBundles []ResourceBundle

func (rbs ResourceBundles) HasTypeName(typeName TypeName) bool {
	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			if resourceDefinition.TypeName == typeName {
				return true
			}
		}
	}
	return false
}

func (rbs ResourceBundles) getSavedResourceDefinition(ctx context.Context, persistantState PersistantState) ([]ResourceDefinition, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	logger.Info("📂 Loading saved state")
	savedResourceDefinition, err := persistantState.Load(nestedCtx)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	savedResourceBundlesYamlBytes, err := yaml.Marshal(&savedResourceDefinition)
	if err != nil {
		return nil, err
	}
	nestedLogger.WithFields(logrus.Fields{"ResourceBundles": string(savedResourceBundlesYamlBytes)}).Debug("Loaded saved state")

	return savedResourceDefinition, nil
}

func (rbs ResourceBundles) check(
	ctx context.Context, hst host.Host, savedResourceDefinition []ResourceDefinition,
) (map[TypeName]CheckResult, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	logger.Info("🔎 Checking state")
	checkResults := map[TypeName]CheckResult{}
	for _, resourceDefinition := range savedResourceDefinition {
		checkResult, err := resourceDefinition.Check(nestedCtx, hst)
		if err != nil {
			return nil, err
		}
		nestedLogger.Infof("%s %v", checkResult, resourceDefinition.TypeName)
		if !checkResult {
			return nil, fmt.Errorf("resource previously applied now failing check; this usually means that the resource was changed externally")
		}
		checkResults[resourceDefinition.TypeName] = checkResult
	}
	for _, resourceBundle := range rbs {
		for _, resourceDefinition := range resourceBundle {
			if _, ok := checkResults[resourceDefinition.TypeName]; ok {
				continue
			}
			checkResult, err := resourceDefinition.Check(nestedCtx, hst)
			if err != nil {
				return nil, err
			}
			nestedLogger.Infof("%s %v", checkResult, resourceDefinition.TypeName)
			checkResults[resourceDefinition.TypeName] = checkResult
		}
	}
	return checkResults, nil
}

func (rbs ResourceBundles) buildPlan(ctx context.Context, checkResults map[TypeName]CheckResult) (Plan, error) {
	logger := log.GetLogger(ctx)

	// Build unsorted digraph
	logger.Info("👷  Building plan")
	unsortedPlan := Plan{}
	mergedNodes := map[Type]*Node{}
	var lastResourceBundleLastNode *Node
	for _, resourceBundle := range rbs {
		// Create nodes with only resource definitions...
		resourceBundleNodes := []*Node{}
		var node *Node
		for i, resourceDefinition := range resourceBundle {
			node = &Node{
				ResourceDefinitions: []ResourceDefinition{resourceDefinition},
			}
			resourceBundleNodes = append(resourceBundleNodes, node)
			if i == 0 && lastResourceBundleLastNode != nil {
				lastResourceBundleLastNode.PrerequisiteFor = append(lastResourceBundleLastNode.PrerequisiteFor, node)
			}
		}
		lastResourceBundleLastNode = node
		// ...and populate other Node attributes
		var previousNode *Node
		refresh := false
		for _, node := range resourceBundleNodes {
			if previousNode != nil {
				previousNode.PrerequisiteFor = append(previousNode.PrerequisiteFor, node)
			}
			typeName := node.ResourceDefinitions[0].TypeName
			manageableResource, err := typeName.ManageableResource()
			if err != nil {
				return nil, err
			}
			checkResult, ok := checkResults[typeName]
			if !ok {
				panic("missing check result")
			}
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
					unsortedPlan = append(unsortedPlan, mergedNode)
				}
				mergedNode.ResourceDefinitions = append(mergedNode.ResourceDefinitions, node.ResourceDefinitions...)
				mergedNode.PrerequisiteFor = append(mergedNode.PrerequisiteFor, node)
				if !checkResult {
					mergedNode.Action = ActionApply
				}
			} else {
				if checkResult {
					node.Action = ActionNone
				} else {
					node.Action = ActionApply
					refresh = true
				}
			}
			node.Refresh = refresh
			previousNode = node
		}
		unsortedPlan = append(unsortedPlan, resourceBundleNodes...)
	}

	// Sort
	plan, err := unsortedPlan.topologicalSort()
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// GetPlan calculates the plan and returns it in the form of a Plan
func (rbs ResourceBundles) GetPlan(ctx context.Context, hst host.Host, persistantState PersistantState) (Plan, error) {
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	// Load saved state
	savedResourceDefinition, err := rbs.getSavedResourceDefinition(ctx, persistantState)
	if err != nil {
		return nil, err
	}

	// Checking state
	checkResults, err := rbs.check(ctx, hst, savedResourceDefinition)
	if err != nil {
		return nil, err
	}

	// Build unsorted digraph
	plan, err := rbs.buildPlan(ctx, checkResults)
	if err != nil {
		return nil, err
	}

	// Append destroy nodes
	for _, resourceDefinition := range savedResourceDefinition {
		if !rbs.HasTypeName(resourceDefinition.TypeName) {
			node := &Node{
				ResourceDefinitions: []ResourceDefinition{ResourceDefinition{
					TypeName: resourceDefinition.TypeName,
				}},
				PrerequisiteFor: []*Node{plan[0]},
				Action:          ActionDestroy,
			}
			plan = append(Plan{node}, plan...)
		}
	}

	nestedLogger.WithFields(logrus.Fields{"Graphviz": plan.Graphviz()}).Debug("Final plan")

	return plan, nil
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
	logger := log.GetLogger(ctx)

	resourceBundles := ResourceBundles{}
	for _, path := range paths {
		resourceBundle, err := loadResourceBundle(ctx, path)
		if err != nil {
			logger.Fatal(err)
		}
		resourceBundles = append(resourceBundles, resourceBundle)
	}
	return resourceBundles
}

// PersistantState defines an interface for loading and saving HostState
type PersistantState interface {
	Load(ctx context.Context) ([]ResourceDefinition, error)
	Save(ctx context.Context, resourceDefinitions []ResourceDefinition) error
}
