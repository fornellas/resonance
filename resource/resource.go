package resource

import (
	"bytes"
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
		return "🗸"
	} else {
		return "❌"
	}
}

// Action to be executed for a resource definition.
type Action int

const (
	// ActionOk means state is as expected
	ActionOk Action = iota
	// ActionSkip denotes no action is required as the resource was merged.
	ActionSkip
	// ActionRefresh means that any in-memory state is to be refreshed (eg: restart a service, reload configuration from files etc).
	ActionRefresh
	// ActionApply means that the state of the resource is not as expected and it that it must be configured on apply.
	// Implies ActionRefresh.
	ActionApply
	// ActionDestroy means that the resource is no longer needed and is to be destroyed.
	ActionDestroy
	// ActionCount has the number of existing actions
	ActionCount
)

var actionEmojiMap = map[Action]string{
	ActionOk:      "🗸",
	ActionSkip:    "💨",
	ActionRefresh: "🔄",
	ActionApply:   "🔧",
	ActionDestroy: "💀",
}

func (a Action) Emoji() string {
	emoji, ok := actionEmojiMap[a]
	if !ok {
		panic(fmt.Errorf("invalid action %d", a))
	}
	return emoji
}

var actionStrMap = map[Action]string{
	ActionOk:      "OK",
	ActionSkip:    "Skip",
	ActionRefresh: "Refresh",
	ActionApply:   "Apply",
	ActionDestroy: "Destroy",
}

func (a Action) String() string {
	str, ok := actionStrMap[a]
	if !ok {
		panic(fmt.Errorf("invalid action %d", a))
	}
	return fmt.Sprintf("%s %s", a.Emoji(), str)
}

func (a Action) GraphvizColor() string {
	var color string
	switch a {
	case ActionOk:
		color = "green4"
	case ActionSkip:
		color = "gray4"
	case ActionRefresh:
		color = "blue4"
	case ActionApply:
		color = "yellow4"
	case ActionDestroy:
		color = "red4"
	default:
		panic(fmt.Sprintf("unexpected Action %q", a))
	}
	return color
}

// Definitions describe a set of resource declarations.
type Definitions map[Name]yaml.Node

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// Check host for the state of instatnce. If changes are required, returns true,
	// otherwise, returns false.
	// No side-effects are to happen when this function is called, which may happen concurrently.
	Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error)

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistant changes are made
	// (eg: change configuration file)
	Refresh(ctx context.Context, hst host.Host, name Name) error
}

// IndividuallyManageableResource is an interface for managing a single resource name.
// This is the most common use case, where resources can be individually managed without one resource
// having dependency on others and changing one resource does not affect the state of another.
type IndividuallyManageableResource interface {
	ManageableResource

	// Apply configures the resource definition at host.
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) error

	// Destroy a configured resource at given host.
	// Must be idempotent.
	Destroy(ctx context.Context, hst host.Host, name Name) error
}

// MergeableManageableResources is an interface for managing multiple resources together.
// The use cases for this are resources where there's inter-dependency between resources, and they
// must be managed "all or nothing". The typical use case is Linux distribution package management,
// where one package may conflict with another, and the transaction of the final state must be
// computed altogether.
type MergeableManageableResources interface {
	ManageableResource

	// ConfigureAll configures all resource definitions at host.
	// Must be idempotent.
	ConfigureAll(ctx context.Context, hst host.Host, actionDefinition map[Action]Definitions) error
}

// Type is the name of the resource type.
type Type string

// IndividuallyManageableResourceTypeMap maps Type to IndividuallyManageableResource.
var IndividuallyManageableResourceTypeMap = map[Type]IndividuallyManageableResource{}

// MergeableManageableResourcesTypeMap maps Type to MergeableManageableResources.
var MergeableManageableResourcesTypeMap = map[Type]MergeableManageableResources{}

// Validate whether type is known.
func (t Type) Validate() error {
	individuallyManageableResource, ok := IndividuallyManageableResourceTypeMap[t]
	if ok {
		rType := reflect.TypeOf(individuallyManageableResource)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at IndividuallyManageableResourceTypeMap, not %s",
				rType.Name(), rType.Name(), string(t),
			))
		}
		return nil
	}

	mergeableManageableResources, ok := MergeableManageableResourcesTypeMap[t]
	if ok {
		rType := reflect.TypeOf(mergeableManageableResources)
		if string(t) != rType.Name() {
			panic(fmt.Errorf(
				"%s must be defined with key %s at MergeableManageableResources, not %s",
				rType.Name(), rType.Name(), string(t),
			))
		}
		return nil
	}

	return fmt.Errorf("unknown resource type '%s'", t)
}

func (t Type) MustValidate() {
	if err := t.Validate(); err != nil {
		panic(err)
	}
}

// ManageableResource returns an instance for the resource type.
func (t Type) ManageableResource() ManageableResource {
	individuallyManageableResource, ok := IndividuallyManageableResourceTypeMap[t]
	if ok {
		return individuallyManageableResource
	}

	mergeableManageableResources, ok := MergeableManageableResourcesTypeMap[t]
	if ok {
		return mergeableManageableResources
	}

	panic(fmt.Errorf("unknown resource type '%s'", t))
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

// ResourceDefinitionSchema is the schema used to define a single resource within a Yaml file.
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

func (rd ResourceDefinition) GraphvizLabel(action Action) string {
	return fmt.Sprintf("< %s[<font color=\"%s\"><b>%s</b></font>] >", rd.Type(), action.GraphvizColor(), rd.Name)
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

func (rd ResourceDefinition) IsIndividuallyManageableResource() bool {
	_, ok := rd.ManageableResource.(IndividuallyManageableResource)
	return ok
}

func (rd ResourceDefinition) IndividuallyManageableResource() IndividuallyManageableResource {
	individuallyManageableResource, ok := rd.ManageableResource.(IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", rd))
	}
	return individuallyManageableResource
}

func (rd ResourceDefinition) IsMergeableManageableResources() bool {
	_, ok := rd.ManageableResource.(MergeableManageableResources)
	return ok
}

func (rd ResourceDefinition) MergeableManageableResources() MergeableManageableResources {
	mergeableManageableResources, ok := rd.ManageableResource.(MergeableManageableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableManageableResources", rd))
	}
	return mergeableManageableResources
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
	logger.Info("📂 Loading resources")

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

type NodeAction interface {
	Execute(ctx context.Context, hst host.Host) error
	String() string
	GraphvizLabel() string
}

type NodeActionIndividual struct {
	ResourceDefinition ResourceDefinition
	Action             Action
}

func (nai NodeActionIndividual) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	logger.Infof("%s", nai)
	nestedCtx := log.IndentLogger(ctx)
	name := nai.ResourceDefinition.Name
	parameters := nai.ResourceDefinition.Parameters
	switch nai.Action {
	case ActionOk, ActionSkip:
	case ActionRefresh:
		return nai.ResourceDefinition.IndividuallyManageableResource().Refresh(nestedCtx, hst, name)
	case ActionApply:
		return nai.ResourceDefinition.IndividuallyManageableResource().Apply(nestedCtx, hst, name, parameters)
	case ActionDestroy:
		return nai.ResourceDefinition.IndividuallyManageableResource().Destroy(nestedCtx, hst, name)
	default:
		panic(fmt.Errorf("unexpected action %v", nai.Action))
	}
	return nil
}

func (nai NodeActionIndividual) String() string {
	return fmt.Sprintf("%s[%s %s]", nai.ResourceDefinition.Type(), nai.Action.Emoji(), nai.ResourceDefinition.Name)
}

func (nai NodeActionIndividual) GraphvizLabel() string {
	return nai.ResourceDefinition.GraphvizLabel(nai.Action)
}

type NodeActionMerged struct {
	ActionResourceDefinitions map[Action][]ResourceDefinition
}

func (nam NodeActionMerged) MergeableManageableResources() MergeableManageableResources {
	var mergeableManageableResources MergeableManageableResources
	for _, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			if mergeableManageableResources != nil && mergeableManageableResources != resourceDefinition.MergeableManageableResources() {
				panic(fmt.Errorf("%s: mixed resource types", nam))
			}
			mergeableManageableResources = resourceDefinition.MergeableManageableResources()
		}
	}
	if mergeableManageableResources == nil {
		panic(fmt.Errorf("%s is empty", nam))
	}
	return mergeableManageableResources
}

func (nam NodeActionMerged) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	logger.Infof("%s", nam)
	nestedCtx := log.IndentLogger(ctx)

	configureActionDefinition := map[Action]Definitions{}
	refreshNames := []Name{}
	for action, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resourceDefinition.Name)
			} else {
				configureActionDefinition[action][resourceDefinition.Name] = resourceDefinition.Parameters
			}
		}
	}

	if err := nam.MergeableManageableResources().ConfigureAll(nestedCtx, hst, configureActionDefinition); err != nil {
		return err
	}

	for _, name := range refreshNames {
		if err := nam.MergeableManageableResources().Refresh(nestedCtx, hst, name); err != nil {
			return err
		}
	}

	return nil
}

func (nam NodeActionMerged) String() string {
	var tpe Type
	names := []string{}
	for action, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			tpe = resourceDefinition.Type()
			names = append(names, fmt.Sprintf("%s %s", action.Emoji(), string(resourceDefinition.Name)))
		}
	}
	return fmt.Sprintf("%s[%s]", tpe, strings.Join(names, ", "))
}

func (nam NodeActionMerged) Type() Type {
	tpe := Type(reflect.TypeOf(nam.MergeableManageableResources()).Name())
	tpe.MustValidate()
	return tpe
}

func (nam NodeActionMerged) GraphvizLabel() string {
	var buff bytes.Buffer

	fmt.Fprintf(&buff, "< %s[", nam.Type())

	first := true
	for action, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			if !first {
				fmt.Fprint(&buff, ",")
			}
			fmt.Fprintf(
				&buff, "<font color=\"%s\"><b>%s</b></font>",
				action.GraphvizColor(), resourceDefinition.Name,
			)
			first = false
		}
	}
	fmt.Fprint(&buff, "] >")

	return buff.String()
}

// Node from a Plan
type Node struct {
	NodeAction      NodeAction
	PrerequisiteFor []*Node
}

func (n Node) String() string {
	return n.NodeAction.String()
}

func (n Node) Execute(ctx context.Context, hst host.Host) error {
	return n.NodeAction.Execute(ctx, hst)
}

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan []*Node

// Graphviz returns a DOT directed graph containing the apply plan.
func (p Plan) Graphviz() string {
	var buff bytes.Buffer
	fmt.Fprint(&buff, "digraph resonance {\n")
	for _, node := range p {
		fmt.Fprintf(&buff, "  node [shape=rectanble] \"%s\"\n", node)
	}
	for _, node := range p {
		for _, dependantNode := range node.PrerequisiteFor {
			fmt.Fprintf(&buff, "  \"%s\" -> \"%s\"\n", node.String(), dependantNode.String())
		}
	}
	fmt.Fprint(&buff, "}\n")
	return buff.String()
}

// Execute required changes to host
func (p Plan) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	logger.Info("▶️  Executing changes")
	nestedCtx := log.IndentLogger(ctx)
	for _, node := range p {
		err := node.Execute(nestedCtx, hst)
		if err != nil {
			return err
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

func (p Plan) Print(ctx context.Context) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Plan")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	for _, node := range p {
		nestedLogger.Infof("%s", node)
	}

	nestedLogger.WithFields(logrus.Fields{"Digraph": p.Graphviz()}).Debug("Graphviz")
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

func buildApplyRefreshPlan(
	ctx context.Context,
	resourceBundles ResourceBundles,
	checkResults map[ResourceDefinitionKey]CheckResult,
) (Plan, error) {
	logger := log.GetLogger(ctx)

	// Build unsorted digraph
	logger.Info("👷 Building apply/refresh plan")
	unsortedPlan := Plan{}

	// TODO refresh
	// TODO link last node from one bundle to the first node of the next bundle
	mergedNodes := map[Type]*Node{}
	for _, resourceBundle := range resourceBundles {
		resourceBundleNodes := []*Node{}
		for i, resourceDefinition := range resourceBundle {
			node := &Node{}
			unsortedPlan = append(unsortedPlan, node)

			// Result
			checkResult, ok := checkResults[resourceDefinition.ResourceDefinitionKey()]
			if !ok {
				panic(fmt.Errorf("%v missing check result", resourceDefinition))
			}

			// Action
			var action Action
			if checkResult {
				action = ActionOk
			} else {
				action = ActionApply
			}

			// Prerequisites
			resourceBundleNodes = append(resourceBundleNodes, node)
			if i > 0 {
				dependantNode := resourceBundleNodes[i-1]
				dependantNode.PrerequisiteFor = append(dependantNode.PrerequisiteFor, node)
			}

			// NodeAction
			nodeAction := NodeActionIndividual{
				ResourceDefinition: resourceDefinition,
			}
			if resourceDefinition.IsIndividuallyManageableResource() {
				nodeAction.Action = action
			} else if resourceDefinition.IsMergeableManageableResources() {
				nodeAction.Action = ActionSkip

				// Merged node
				mergedNode, ok := mergedNodes[resourceDefinition.Type()]
				if !ok {
					mergedNode = &Node{NodeAction: NodeActionMerged{
						ActionResourceDefinitions: map[Action][]ResourceDefinition{},
					}}
					unsortedPlan = append(unsortedPlan, mergedNode)
					mergedNodes[resourceDefinition.Type()] = mergedNode
				}
				nodeActionMerged, ok := mergedNode.NodeAction.(NodeActionMerged)
				if !ok {
					panic(fmt.Errorf("%v NodeAction not NodeActionMerged", mergedNode))
				}
				nodeActionMerged.ActionResourceDefinitions[action] = append(
					nodeActionMerged.ActionResourceDefinitions[action], resourceDefinition,
				)
				mergedNode.PrerequisiteFor = append(mergedNode.PrerequisiteFor, node)
			} else {
				panic(fmt.Errorf("%s: unknown managed resource", resourceDefinition))
			}
			node.NodeAction = nodeAction
		}
	}

	// Sort
	plan, err := unsortedPlan.topologicalSort()
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func appendDestroyNodes(
	ctx context.Context,
	savedResourceBundle ResourceBundle,
	resourceBundles ResourceBundles,
	plan Plan,
) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Prepending resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, resourceDefinition := range savedResourceBundle {
		if resourceBundles.HasResourceDefinition(resourceDefinition) ||
			resourceDefinition.IndividuallyManageableResource() == nil {
			continue
		}
		node := &Node{
			NodeAction: NodeActionIndividual{
				ResourceDefinition: resourceDefinition,
				Action:             ActionDestroy,
			},
			PrerequisiteFor: []*Node{plan[0]},
		}
		nestedLogger.Infof("💀 %s", node)
		plan = append(Plan{node}, plan...)
	}
	return plan
}

// NewPlan calculates the plan and returns it in the form of a Plan
func NewPlan(
	ctx context.Context,
	hst host.Host,
	savedResourceBundle ResourceBundle,
	resourceBundles ResourceBundles,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning changes")
	nestedCtx := log.IndentLogger(ctx)

	// Checking state
	checkResults, err := checkResourcesState(nestedCtx, hst, savedResourceBundle, resourceBundles)
	if err != nil {
		return nil, err
	}

	// Build unsorted digraph
	plan, err := buildApplyRefreshPlan(nestedCtx, resourceBundles, checkResults)
	if err != nil {
		return nil, err
	}

	// Append destroy nodes
	plan = appendDestroyNodes(nestedCtx, savedResourceBundle, resourceBundles, plan)

	// Print
	plan.Print(ctx)

	return plan, nil
}

// PersistantState defines an interface for loading and saving HostState
type PersistantState interface {
	Load(ctx context.Context) ([]ResourceDefinition, error)
	Save(ctx context.Context, resourceDefinitions []ResourceDefinition) error
}
