package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
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

// CheckResult is what's returned when a resource is checked. true means resource state is as
// parameterized, false means changes are pending.
type CheckResult bool

func (cr CheckResult) String() string {
	if cr {
		return "✔️ "
	} else {
		return "🚫"
	}
}

// Definitions describe a set of resource declarations.
type Definitions map[Name]yaml.Node

// ManageableResource defines an interface for managing resource state.
type ManageableResource interface {
	// MergeApply informs whether all resources from the same type are to
	// be always merged together when applying.
	// When true, Apply is called only once, with all definitions.
	// When false, Apply is called one time for each definition.
	MergeApply() bool

	// Check host for the state of instatnce. If changes are required, returns true,
	// otherwise, returns false.
	Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error)

	// Apply configures all resource definitions at host.
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, definitions Definitions) error

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistant changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, hst host.Host, name Name) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, hst host.Host, name Name) error
}

// Type is the name of the resource (aka: a ManageableResource).
type Type string

// TypeToManageableResource maps Type to ManageableResource.
var TypeToManageableResource = map[Type]ManageableResource{}

// Validate whether type is known.
func (t Type) Validate() error {
	manageableResource, ok := TypeToManageableResource[t]
	if !ok {
		return fmt.Errorf("unknown resource type '%s'", t)
	}
	rType := reflect.TypeOf(manageableResource)
	if string(t) != rType.Name() {
		panic(fmt.Errorf(
			"ManageableResource %s must be defined with key %s at TypeToManageableResource, not %s",
			rType.Name(), rType.Name(), string(t),
		))
	}
	return nil
}

// ManageableResource returns an instance for the resource type.
func (t Type) ManageableResource() ManageableResource {
	manageableResource, ok := TypeToManageableResource[t]
	if !ok {
		panic(fmt.Errorf("unknown resource type '%s'", t))
	}
	return manageableResource
}

// TypeName is a string that identifies a resource type and name.
// Eg: File[/etc/issue].
type TypeName string

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

func (tn TypeName) Validate() error {
	var tpe Type
	var err error
	if tpe, _, err = tn.typeName(); err != nil {
		return err
	}
	if err := tpe.Validate(); err != nil {
		return err
	}
	return nil
}

func (tn *TypeName) UnmarshalYAML(node *yaml.Node) error {
	var typeNameStr string
	if err := node.Decode(&typeNameStr); err != nil {
		return err
	}
	*tn = TypeName(typeNameStr)
	if err := tn.Validate(); err != nil {
		return err
	}
	return nil
}

// Name returns the Name of the resource.
func (tn TypeName) Name() Name {
	_, name, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return name
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) ManageableResource() ManageableResource {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe.ManageableResource()
}

// ResourceDefinition is the schema used to define a single resource within a Yaml file.
type ResourceDefinitionSchema struct {
	TypeName   TypeName  `yaml:"resource"`
	Parameters yaml.Node `yaml:"parameters"`
}

// ResourceDefinitionKey is a unique identifier for a ResourceDefinition that
// can be used as keys in maps.
type ResourceDefinitionKey string

// ResourceDefinition holds a single resource definition.
type ResourceDefinition struct {
	ManageableResource ManageableResource
	Name               Name
	Parameters         yaml.Node
}

func (rd *ResourceDefinition) UnmarshalYAML(node *yaml.Node) error {
	var resourceDefinitionSchema ResourceDefinitionSchema
	if err := node.Decode(&resourceDefinitionSchema); err != nil {
		return err
	}
	*rd = ResourceDefinition{
		ManageableResource: resourceDefinitionSchema.TypeName.ManageableResource(),
		Name:               resourceDefinitionSchema.TypeName.Name(),
		Parameters:         resourceDefinitionSchema.Parameters,
	}
	return nil
}

func (rd *ResourceDefinition) Type() Type {
	return Type(reflect.TypeOf(rd.ManageableResource).Name())
}

func (rd ResourceDefinition) String() string {
	return fmt.Sprintf("%s[%s]", rd.Type(), rd.Name)
}

func (rd ResourceDefinition) ResourceDefinitionKey() ResourceDefinitionKey {
	return ResourceDefinitionKey(rd.String())
}

func (rd ResourceDefinition) Check(ctx context.Context, hst host.Host) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	logger.Debugf("Checking %v", rd)
	check, err := rd.ManageableResource.Check(log.IndentLogger(ctx), hst, rd.Name, rd.Parameters)
	if err != nil {
		return false, err
	}
	return check, nil
}

// ResourceBundle is the schema used to declare multiple resources at a single file.
type ResourceBundle []ResourceDefinition

// LoadResourceBundle loads resource definitions from given Yaml file path.
func LoadResourceBundle(ctx context.Context, path string) (ResourceBundle, error) {
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

// LoadSavedState loads a ResourceBundle saved after it was applied to a host.
func LoadSavedState(ctx context.Context, persistantState PersistantState) (ResourceBundle, error) {
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

// ResourceBundles holds all resources definitions for a host.
type ResourceBundles []ResourceBundle

func (rbs ResourceBundles) HasResourceDefinition(resourceDefinition ResourceDefinition) bool {
	for _, resourceBundle := range rbs {
		for _, rd := range resourceBundle {
			if rd.ResourceDefinitionKey() == resourceDefinition.ResourceDefinitionKey() {
				return true
			}
		}
	}
	return false
}

// LoadResourceBundles loads resource definitions from all given Yaml file paths.
// Each file must have the schema defined by ResourceBundle.
func LoadResourceBundles(ctx context.Context, paths []string) ResourceBundles {
	logger := log.GetLogger(ctx)

	resourceBundles := ResourceBundles{}
	for _, path := range paths {
		resourceBundle, err := LoadResourceBundle(ctx, path)
		if err != nil {
			logger.Fatal(err)
		}
		resourceBundles = append(resourceBundles, resourceBundle)
	}
	return resourceBundles
}

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

// Name of the node.
func (n Node) Name() string {
	if len(n.ResourceDefinitions) == 1 {
		return n.ResourceDefinitions[0].String()
	} else if len(n.ResourceDefinitions) > 1 {
		var tpe Type
		names := []string{}
		for _, resourceDefinition := range n.ResourceDefinitions {
			tpe = resourceDefinition.Type()
			names = append(names, string(resourceDefinition.Name))
		}
		return fmt.Sprintf("%s[%s]", tpe, strings.Join(names, ","))
	} else {
		panic("invalid node: empty ResourceDefinitions")
	}
}

func (n Node) ManageableResource() ManageableResource {
	return n.ResourceDefinitions[0].ManageableResource
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

func (p Plan) actionNoneSkip(ctx context.Context, hst host.Host, node *Node) error {
	logger := log.GetLogger(ctx)
	if node.Refresh {
		logger.Infof("🔁 Refresh: %s", node.Name())
		for _, resourceDefinition := range node.ResourceDefinitions {
			if len(node.ResourceDefinitions) > 1 {
				logger.Infof("  Refreshing %s", resourceDefinition)
			}
			if err := node.ManageableResource().Refresh(ctx, hst, resourceDefinition.Name); err != nil {
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
	logger.Infof("%s: %s", node.Action, node.Name())
	definitions := Definitions{}
	for _, resourceDefinition := range node.ResourceDefinitions {
		definitions[resourceDefinition.Name] = resourceDefinition.Parameters
	}
	if err := node.ManageableResource().Apply(ctx, hst, definitions); err != nil {
		return err
	}

	return nil
}

func (p Plan) actionDestroy(ctx context.Context, hst host.Host, node *Node) error {
	logger := log.GetLogger(ctx)
	logger.Infof("%s: %s", node.Action, node.Name())
	for _, resourceDefinition := range node.ResourceDefinitions {
		if len(node.ResourceDefinitions) > 1 {
			logger.Infof("Destroying %s", resourceDefinition)
		}
		if err := node.ManageableResource().Destroy(ctx, hst, resourceDefinition.Name); err != nil {
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

func checkResourcesState(
	ctx context.Context,
	hst host.Host,
	savedResourceDefinition []ResourceDefinition,
	resourceBundles ResourceBundles,
) (map[ResourceDefinitionKey]CheckResult, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	logger.Info("🔎 Checking state")
	checkResults := map[ResourceDefinitionKey]CheckResult{}
	for _, resourceDefinition := range savedResourceDefinition {
		checkResult, err := resourceDefinition.Check(nestedCtx, hst)
		if err != nil {
			return nil, err
		}
		nestedLogger.Infof("%s %s", checkResult, resourceDefinition)
		if !checkResult {
			return nil, fmt.Errorf("resource previously applied now failing check; this usually means that the resource was changed externally")
		}
		checkResults[resourceDefinition.ResourceDefinitionKey()] = checkResult
	}
	for _, resourceBundle := range resourceBundles {
		for _, resourceDefinition := range resourceBundle {
			if _, ok := checkResults[resourceDefinition.ResourceDefinitionKey()]; ok {
				continue
			}
			checkResult, err := resourceDefinition.Check(nestedCtx, hst)
			if err != nil {
				return nil, err
			}
			nestedLogger.Infof("%s %s", checkResult, resourceDefinition)
			checkResults[resourceDefinition.ResourceDefinitionKey()] = checkResult
		}
	}
	return checkResults, nil
}

func buildPlan(
	ctx context.Context,
	resourceBundles ResourceBundles,
	checkResults map[ResourceDefinitionKey]CheckResult,
) (Plan, error) {
	logger := log.GetLogger(ctx)

	// Build unsorted digraph
	logger.Info("👷  Building plan")
	unsortedPlan := Plan{}
	mergedNodes := map[Type]*Node{}
	var lastResourceBundleLastNode *Node
	for _, resourceBundle := range resourceBundles {
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
			resourceDefinition := node.ResourceDefinitions[0]
			checkResult, ok := checkResults[resourceDefinition.ResourceDefinitionKey()]
			if !ok {
				panic("missing check result")
			}
			if node.ManageableResource().MergeApply() {
				node.Action = ActionSkip
				mergedNode, ok := mergedNodes[resourceDefinition.Type()]
				if !ok {
					mergedNode = &Node{}
					mergedNodes[resourceDefinition.Type()] = mergedNode
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

// NewPlan calculates the plan and returns it in the form of a Plan
func NewPlan(
	ctx context.Context,
	hst host.Host,
	savedResourceBundle ResourceBundle,
	resourceBundles ResourceBundles,
) (Plan, error) {
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	// Checking state
	checkResults, err := checkResourcesState(ctx, hst, savedResourceBundle, resourceBundles)
	if err != nil {
		return nil, err
	}

	// Build unsorted digraph
	plan, err := buildPlan(ctx, resourceBundles, checkResults)
	if err != nil {
		return nil, err
	}

	// Append destroy nodes
	for _, resourceDefinition := range savedResourceBundle {
		if !resourceBundles.HasResourceDefinition(resourceDefinition) {
			node := &Node{
				ResourceDefinitions: []ResourceDefinition{resourceDefinition},
				PrerequisiteFor:     []*Node{plan[0]},
				Action:              ActionDestroy,
			}
			plan = append(Plan{node}, plan...)
		}
	}

	nestedLogger.WithFields(logrus.Fields{"Graphviz": plan.Graphviz()}).Debug("Final plan")

	return plan, nil
}

// PersistantState defines an interface for loading and saving HostState
type PersistantState interface {
	Load(ctx context.Context) ([]ResourceDefinition, error)
	Save(ctx context.Context, resourceDefinitions []ResourceDefinition) error
}
