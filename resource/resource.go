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
	"sort"
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
		return "✅"
	} else {
		return "🔧"
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
	ActionOk:      "✅",
	ActionSkip:    "💨",
	ActionRefresh: "🔄",
	ActionApply:   "🔧",
	ActionDestroy: "💀",
}

// Emoji representing the action
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
	return str
}

// Definitions describe a set of resource declarations.
type Definitions map[Name]yaml.Node

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// Validate the name of the resource
	Validate(name Name) error

	// Check host for the state of instatnce. If changes are required, returns true,
	// otherwise, returns false.
	// No side-effects are to happen when this function is called, which may happen concurrently.
	Check(ctx context.Context, hst host.Host, name Name, parameters yaml.Node) (CheckResult, error)
}

// RefreshableManageableResource defines an interface for resources that can be refreshed.
// Refresh means updating in-memory state as a function of file changes (eg: restarting a service,
// loading iptables rules to the kernel etc.)
type RefreshableManageableResource interface {
	ManageableResource

	// Refresh the resource. This is typically used to update the in-memory state of a resource
	// (eg: kerner: sysctl, iptables; process: systemd service) after persistent changes are made
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
	ConfigureAll(ctx context.Context, hst host.Host, actionDefinitions map[Action]Definitions) error
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

// Mustvalidate panics if it validation fails
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

// Validate whether format and type are valid.
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
	node.KnownFields(true)
	if err := node.Decode(&typeNameStr); err != nil {
		return err
	}
	*tn = TypeName(typeNameStr)
	if err := tn.Validate(); err != nil {
		return err
	}
	return nil
}

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

// CheckResults holds results for multiple resources.
type CheckResults map[ResourceDefinitionKey]CheckResult

// ResourceDefinition holds a single resource definition.
type ResourceDefinition struct {
	ManageableResource ManageableResource
	Name               Name
	Parameters         yaml.Node
}

func (rd *ResourceDefinition) UnmarshalYAML(node *yaml.Node) error {
	var resourceDefinitionSchema ResourceDefinitionSchema
	node.KnownFields(true)
	if err := node.Decode(&resourceDefinitionSchema); err != nil {
		return err
	}
	manageableResource := resourceDefinitionSchema.TypeName.ManageableResource()
	name := resourceDefinitionSchema.TypeName.Name()
	if err := manageableResource.Validate(name); err != nil {
		return err
	}
	*rd = ResourceDefinition{
		ManageableResource: manageableResource,
		Name:               name,
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

// Check runs ManageableResource.Check.
func (rd ResourceDefinition) Check(ctx context.Context, hst host.Host) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	logger.Debugf("Checking %v", rd)
	return rd.ManageableResource.Check(log.IndentLogger(ctx), hst, rd.Name, rd.Parameters)
}

// Refreshable returns whether the resource definition is refreshable or not.
func (rd ResourceDefinition) Refreshable() bool {
	_, ok := rd.ManageableResource.(RefreshableManageableResource)
	return ok
}

// IsIndividuallyManageableResource returns true only if ManageableResource is of type IndividuallyManageableResource.
func (rd ResourceDefinition) IsIndividuallyManageableResource() bool {
	_, ok := rd.ManageableResource.(IndividuallyManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (rd ResourceDefinition) MustIndividuallyManageableResource() IndividuallyManageableResource {
	individuallyManageableResource, ok := rd.ManageableResource.(IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", rd))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource is of type MergeableManageableResources.
func (rd ResourceDefinition) IsMergeableManageableResources() bool {
	_, ok := rd.ManageableResource.(MergeableManageableResources)
	return ok
}

// MustMergeableManageableResources returns MergeableManageableResources from ManageableResource or
// panics if it isn't of the required type.
func (rd ResourceDefinition) MustMergeableManageableResources() MergeableManageableResources {
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

// HasResourceDefinition returns true if ResourceDefinition is contained at ResourceBundles.
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

// NodeAction defines an interface for an action that can be executed from a Node.
type NodeAction interface {
	// Execute actions for all bundled resource definitions.
	Execute(ctx context.Context, hst host.Host) error
	String() string
	// Actionable returns whether any action is different from ActionOk or ActionSkip
	Actionable() bool
}

// NodeActionIndividual is a NodeAction which can execute a single ResourceDefinition.
type NodeActionIndividual struct {
	ResourceDefinition ResourceDefinition
	Action             Action
}

// Execute Action for the ResourceDefinition.
func (nai NodeActionIndividual) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	if nai.Action == ActionOk || nai.Action == ActionSkip {
		logger.Debugf("%s", nai)
		return nil
	}
	logger.Infof("%s", nai)
	nestedCtx := log.IndentLogger(ctx)
	individuallyManageableResource := nai.ResourceDefinition.MustIndividuallyManageableResource()
	name := nai.ResourceDefinition.Name
	parameters := nai.ResourceDefinition.Parameters
	switch nai.Action {
	case ActionRefresh:
		refreshableManageableResource, ok := individuallyManageableResource.(RefreshableManageableResource)
		if ok {
			return refreshableManageableResource.Refresh(nestedCtx, hst, name)
		}
	case ActionApply:
		if err := individuallyManageableResource.Apply(nestedCtx, hst, name, parameters); err != nil {
			return err
		}
		checkResult, err := individuallyManageableResource.Check(nestedCtx, hst, name, parameters)
		if err != nil {
			return err
		}
		if !checkResult {
			return fmt.Errorf("%s: check failed immediately after apply: this often means there's a bug 🪲 with the resource implementation", nai.ResourceDefinition)
		}
	case ActionDestroy:
		return individuallyManageableResource.Destroy(nestedCtx, hst, name)
	default:
		panic(fmt.Errorf("unexpected action %v", nai.Action))
	}
	return nil
}

func (nai NodeActionIndividual) String() string {
	return fmt.Sprintf("%s[%s %s]", nai.ResourceDefinition.Type(), nai.Action.Emoji(), nai.ResourceDefinition.Name)
}

func (nai NodeActionIndividual) Actionable() bool {
	return !(nai.Action == ActionOk || nai.Action == ActionSkip)
}

// NodeActionMerged is a NodeAction which contains multiple merged ResourceDefinition.
type NodeActionMerged struct {
	ActionResourceDefinitions map[Action][]ResourceDefinition
}

// MustMergeableManageableResources returns MergeableManageableResources common to all
// ResourceDefinition or panics.
func (nam NodeActionMerged) MustMergeableManageableResources() MergeableManageableResources {
	var mergeableManageableResources MergeableManageableResources
	for _, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			if mergeableManageableResources != nil &&
				mergeableManageableResources != resourceDefinition.MustMergeableManageableResources() {
				panic(fmt.Errorf("%s: mixed resource types", nam))
			}
			mergeableManageableResources = resourceDefinition.MustMergeableManageableResources()
		}
	}
	if mergeableManageableResources == nil {
		panic(fmt.Errorf("%s is empty", nam))
	}
	return mergeableManageableResources
}

func (nam NodeActionMerged) Actionable() bool {
	hasAction := false
	for action := range nam.ActionResourceDefinitions {
		if action == ActionOk || action == ActionSkip {
			continue
		}
		hasAction = true
		break
	}
	return hasAction
}

// Execute the required Action for each ResourceDefinition.
func (nam NodeActionMerged) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)

	if !nam.Actionable() {
		logger.Debugf("%s", nam)
		return nil
	}
	logger.Infof("%s", nam)

	checkResourceDefinitions := []ResourceDefinition{}
	configureActionDefinition := map[Action]Definitions{}
	refreshNames := []Name{}
	for action, resourceDefinitions := range nam.ActionResourceDefinitions {
		for _, resourceDefinition := range resourceDefinitions {
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resourceDefinition.Name)
			} else {
				if configureActionDefinition[action] == nil {
					configureActionDefinition[action] = Definitions{}
				}
				configureActionDefinition[action][resourceDefinition.Name] = resourceDefinition.Parameters
			}
			if action != ActionDestroy {
				checkResourceDefinitions = append(checkResourceDefinitions, resourceDefinition)
			}
		}
	}

	if err := nam.MustMergeableManageableResources().ConfigureAll(
		nestedCtx, hst, configureActionDefinition,
	); err != nil {
		return err
	}

	for _, resourceDefinition := range checkResourceDefinitions {
		checkResult, err := nam.MustMergeableManageableResources().Check(
			nestedCtx, hst, resourceDefinition.Name, resourceDefinition.Parameters,
		)
		if err != nil {
			return err
		}
		if !checkResult {
			return fmt.Errorf("%s: check failed immediately after apply: this often means there's a bug 🪲 with the resource implementation", resourceDefinition)
		}
	}

	for _, name := range refreshNames {
		refreshableManageableResource, ok := nam.MustMergeableManageableResources().(RefreshableManageableResource)
		if ok {
			return refreshableManageableResource.Refresh(nestedCtx, hst, name)
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
	sort.Strings(names)
	return fmt.Sprintf("%s[%s]", tpe, strings.Join(names, ", "))
}

// Node that's used at a Plan
type Node struct {
	NodeAction      NodeAction
	PrerequisiteFor []*Node
}

func (n Node) String() string {
	return n.NodeAction.String()
}

// Execute NodeAction
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

// Execute all Node at the Plan.
func (p Plan) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	logger.Info("🛠️  Executing changes")
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

func (p Plan) Actionable() bool {
	for _, node := range p {
		if node.NodeAction.Actionable() {
			return true
		}
	}
	return false
}

// Print the whole plan
func (p Plan) Print(ctx context.Context) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Plan")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	var legendBuff bytes.Buffer
	first := true
	for action := Action(0); action < ActionCount; action++ {
		if !first {
			fmt.Fprint(&legendBuff, ", ")
		}
		fmt.Fprintf(&legendBuff, "%s = %s", action.Emoji(), action.String())
		first = false
	}
	nestedLogger.Infof("%s", legendBuff.String())

	for _, node := range p {
		if node.NodeAction.Actionable() {
			nestedLogger.Infof("%s", node)
		} else {
			nestedLogger.Debugf("%s", node)
		}
	}

	nestedLogger.WithFields(logrus.Fields{"Digraph": p.Graphviz()}).Debug("Graphviz")
}

func checkResourcesState(
	ctx context.Context,
	hst host.Host,
	savedResourceDefinition []ResourceDefinition,
	resourceBundles ResourceBundles,
) (CheckResults, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	logger.Info("🔎 Checking state")
	checkResults := CheckResults{}
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
	checkResults CheckResults,
) (Plan, error) {
	logger := log.GetLogger(ctx)

	// Build unsorted digraph
	logger.Info("👷 Building apply/refresh plan")
	unsortedPlan := Plan{}

	mergedNodes := map[Type]*Node{}
	var lastBundleLastNode *Node
	for _, resourceBundle := range resourceBundles {
		resourceBundleNodes := []*Node{}
		refresh := false
		var node *Node
		for i, resourceDefinition := range resourceBundle {
			node = &Node{}
			unsortedPlan = append(unsortedPlan, node)

			// Dependant on previous bundle
			if i == 0 && lastBundleLastNode != nil {
				lastBundleLastNode.PrerequisiteFor = append(lastBundleLastNode.PrerequisiteFor, node)
			}

			// Result
			checkResult, ok := checkResults[resourceDefinition.ResourceDefinitionKey()]
			if !ok {
				panic(fmt.Errorf("%v missing check result", resourceDefinition))
			}

			// Action
			var action Action
			if checkResult {
				if refresh && resourceDefinition.Refreshable() {
					action = ActionRefresh
				} else {
					action = ActionOk
				}
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

			// Refresh
			if action == ActionApply {
				refresh = true
			}
		}
		lastBundleLastNode = node
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
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, resourceDefinition := range savedResourceBundle {
		if resourceBundles.HasResourceDefinition(resourceDefinition) ||
			resourceDefinition.MustIndividuallyManageableResource() == nil {
			continue
		}
		node := &Node{
			NodeAction: NodeActionIndividual{
				ResourceDefinition: resourceDefinition,
				Action:             ActionDestroy,
			},
			PrerequisiteFor: []*Node{plan[0]},
		}
		nestedLogger.Infof("%s", node)
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

	return plan, nil
}

// PersistantState defines an interface for loading and saving HostState
type PersistantState interface {
	Load(ctx context.Context) ([]ResourceDefinition, error)
	Save(ctx context.Context, resourceDefinitions []ResourceDefinition) error
}
