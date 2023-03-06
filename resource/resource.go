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
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/version"
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

// SaveState tells whether a Step with this action should have its state saved.
func (a Action) SaveState() bool {
	if a == ActionSkip || a == ActionDestroy {
		return false
	}
	return true
}

// Parameters is a Type specific interface for defining resource parameters.
type Parameters interface{}

// Definitions describe a set of resource declarations.
type Definitions map[Name]Parameters

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// Validate the name of the resource
	Validate(name Name) error

	// Check host for the state of instatnce. If changes are required, returns true,
	// otherwise, returns false.
	// No side-effects are to happen when this function is called, which may happen concurrently.
	Check(ctx context.Context, hst host.Host, name Name, parameters Parameters) (CheckResult, error)
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
	Apply(ctx context.Context, hst host.Host, name Name, parameters Parameters) error

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

// ManageableResourcesParametersMap maps Type to its Parameters interface
var ManageableResourcesParametersMap = map[Type]Parameters{}

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
		return tpe, name, fmt.Errorf("%#v does not match Type[Name] format", tn)
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
	typeName := TypeName(typeNameStr)
	if err := typeName.Validate(); err != nil {
		return err
	}
	*tn = typeName
	return nil
}

func (tn TypeName) Type() Type {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe
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

// ResourceDefinitionKey is a unique identifier for a ResourceDefinition that
// can be used as keys in maps.
type ResourceDefinitionKey string

// CheckResults holds results for multiple resources.
type CheckResults map[ResourceDefinitionKey]CheckResult

// ResourceDefinition holds a single resource definition.
type ResourceDefinition struct {
	TypeName           TypeName           `yaml:"resource"`
	Parameters         Parameters         `yaml:"parameters"`
	ManageableResource ManageableResource `yaml:"-"`
}

type ResourceDefinitionUnmarshalSchema struct {
	TypeName   TypeName  `yaml:"resource"`
	Parameters yaml.Node `yaml:"parameters"`
}

func (rd *ResourceDefinition) UnmarshalYAML(node *yaml.Node) error {
	var resourceDefinitionUnmarshalSchema ResourceDefinitionUnmarshalSchema
	node.KnownFields(true)
	if err := node.Decode(&resourceDefinitionUnmarshalSchema); err != nil {
		return err
	}

	manageableResource := resourceDefinitionUnmarshalSchema.TypeName.ManageableResource()
	tpe := resourceDefinitionUnmarshalSchema.TypeName.Type()
	name := resourceDefinitionUnmarshalSchema.TypeName.Name()
	if err := manageableResource.Validate(name); err != nil {
		return err
	}

	parametersInterface, ok := ManageableResourcesParametersMap[tpe]
	if !ok {
		panic(fmt.Errorf("Type %s missing from ManageableResourcesParametersMap", tpe))
	}
	parametersType := reflect.ValueOf(parametersInterface).Type()
	parametersValue := reflect.New(parametersType)
	err := resourceDefinitionUnmarshalSchema.Parameters.Decode(parametersValue.Interface())
	if err != nil {
		return err
	}

	*rd = ResourceDefinition{
		TypeName:           resourceDefinitionUnmarshalSchema.TypeName,
		ManageableResource: manageableResource,
		Parameters:         parametersValue.Interface(),
	}
	return nil
}

func (rd *ResourceDefinition) Type() Type {
	return rd.TypeName.Type()
}

func (rd *ResourceDefinition) Name() Name {
	return rd.TypeName.Name()
}

func (rd ResourceDefinition) String() string {
	return fmt.Sprintf("%s[%s]", rd.Type(), rd.Name())
}

func (rd ResourceDefinition) ResourceDefinitionKey() ResourceDefinitionKey {
	return ResourceDefinitionKey(rd.String())
}

// Check runs ManageableResource.Check.
func (rd ResourceDefinition) Check(ctx context.Context, hst host.Host) (CheckResult, error) {
	logger := log.GetLogger(ctx)

	logger.Debugf("Checking %v", rd)
	return rd.ManageableResource.Check(log.IndentLogger(ctx), hst, rd.Name(), rd.Parameters)
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

// HostState holds the state
type HostState struct {
	// Version of the binary used to put the host in this state.
	Version version.Version `yaml:"version"`
	// ResourceDefinitions used to put the host in the state.
	ResourceDefinitions []ResourceDefinition
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

// StepAction defines an interface for an action that can be executed from a Step.
type StepAction interface {
	// Execute actions for all bundled resource definitions.
	Execute(ctx context.Context, hst host.Host) error
	String() string
	// Actionable returns whether any action is different from ActionOk or ActionSkip
	Actionable() bool
	// ActionResourceDefinitions returns a map from Action to a slice of ResourceDefinition.
	ActionResourceDefinitions() map[Action][]ResourceDefinition
}

// StepActionIndividual is a StepAction which can execute a single ResourceDefinition.
type StepActionIndividual struct {
	ResourceDefinition ResourceDefinition
	Action             Action
}

// Execute Action for the ResourceDefinition.
func (sai StepActionIndividual) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	if sai.Action == ActionOk || sai.Action == ActionSkip {
		logger.Debugf("%s", sai)
		return nil
	}
	logger.Infof("%s", sai)
	nestedCtx := log.IndentLogger(ctx)
	individuallyManageableResource := sai.ResourceDefinition.MustIndividuallyManageableResource()
	name := sai.ResourceDefinition.Name()
	parameters := sai.ResourceDefinition.Parameters
	switch sai.Action {
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
			return fmt.Errorf("%s: check failed immediately after apply: this often means there's a bug 🪲 with the resource implementation", sai.ResourceDefinition)
		}
	case ActionDestroy:
		return individuallyManageableResource.Destroy(nestedCtx, hst, name)
	default:
		panic(fmt.Errorf("unexpected action %v", sai.Action))
	}
	return nil
}

func (sai StepActionIndividual) String() string {
	return fmt.Sprintf("%s[%s %s]", sai.ResourceDefinition.Type(), sai.Action.Emoji(), sai.ResourceDefinition.Name())
}

func (sai StepActionIndividual) Actionable() bool {
	return !(sai.Action == ActionOk || sai.Action == ActionSkip)
}

func (sai StepActionIndividual) ActionResourceDefinitions() map[Action][]ResourceDefinition {
	return map[Action][]ResourceDefinition{sai.Action: []ResourceDefinition{sai.ResourceDefinition}}
}

// StepActionMerged is a StepAction which contains multiple merged ResourceDefinition.
type StepActionMerged map[Action][]ResourceDefinition

// MustMergeableManageableResources returns MergeableManageableResources common to all
// ResourceDefinition or panics.
func (sam StepActionMerged) MustMergeableManageableResources() MergeableManageableResources {
	var mergeableManageableResources MergeableManageableResources
	for _, resourceDefinitions := range sam {
		for _, resourceDefinition := range resourceDefinitions {
			if mergeableManageableResources != nil &&
				mergeableManageableResources != resourceDefinition.MustMergeableManageableResources() {
				panic(fmt.Errorf("%s: mixed resource types", sam))
			}
			mergeableManageableResources = resourceDefinition.MustMergeableManageableResources()
		}
	}
	if mergeableManageableResources == nil {
		panic(fmt.Errorf("%s is empty", sam))
	}
	return mergeableManageableResources
}

// Execute the required Action for each ResourceDefinition.
func (sam StepActionMerged) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)

	if !sam.Actionable() {
		logger.Debugf("%s", sam)
		return nil
	}
	logger.Infof("%s", sam)

	checkResourceDefinitions := []ResourceDefinition{}
	configureActionDefinition := map[Action]Definitions{}
	refreshNames := []Name{}
	for action, resourceDefinitions := range sam {
		for _, resourceDefinition := range resourceDefinitions {
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resourceDefinition.Name())
			} else {
				if configureActionDefinition[action] == nil {
					configureActionDefinition[action] = Definitions{}
				}
				configureActionDefinition[action][resourceDefinition.Name()] = resourceDefinition.Parameters
			}
			if action != ActionDestroy {
				checkResourceDefinitions = append(checkResourceDefinitions, resourceDefinition)
			}
		}
	}

	if err := sam.MustMergeableManageableResources().ConfigureAll(
		nestedCtx, hst, configureActionDefinition,
	); err != nil {
		return err
	}

	for _, resourceDefinition := range checkResourceDefinitions {
		checkResult, err := sam.MustMergeableManageableResources().Check(
			nestedCtx, hst, resourceDefinition.Name(), resourceDefinition.Parameters,
		)
		if err != nil {
			return err
		}
		if !checkResult {
			return fmt.Errorf("%s: check failed immediately after apply: this often means there's a bug 🪲 with the resource implementation", resourceDefinition)
		}
	}

	for _, name := range refreshNames {
		refreshableManageableResource, ok := sam.MustMergeableManageableResources().(RefreshableManageableResource)
		if ok {
			return refreshableManageableResource.Refresh(nestedCtx, hst, name)
		}
	}

	return nil
}

func (sam StepActionMerged) String() string {
	var tpe Type
	names := []string{}
	for action, resourceDefinitions := range sam {
		for _, resourceDefinition := range resourceDefinitions {
			tpe = resourceDefinition.Type()
			names = append(names, fmt.Sprintf("%s %s", action.Emoji(), string(resourceDefinition.Name())))
		}
	}
	sort.Strings(names)
	return fmt.Sprintf("%s[%s]", tpe, strings.Join(names, ", "))
}

func (sam StepActionMerged) Actionable() bool {
	hasAction := false
	for action := range sam {
		if action == ActionOk || action == ActionSkip {
			continue
		}
		hasAction = true
		break
	}
	return hasAction
}

func (sam StepActionMerged) ActionResourceDefinitions() map[Action][]ResourceDefinition {
	return sam
}

// Step that's used at a Plan
type Step struct {
	StepAction      StepAction
	prerequisiteFor []*Step
}

func (s Step) Actionable() bool {
	return s.StepAction.Actionable()
}

func (s Step) String() string {
	return s.StepAction.String()
}

// Execute StepAction
func (s Step) Execute(ctx context.Context, hst host.Host) error {
	return s.StepAction.Execute(ctx, hst)
}

func (s Step) ActionResourceDefinitions() map[Action][]ResourceDefinition {
	return s.StepAction.ActionResourceDefinitions()
}

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan []*Step

// Graphviz returns a DOT directed graph containing the apply plan.
func (p Plan) Graphviz() string {
	var buff bytes.Buffer
	fmt.Fprint(&buff, "digraph resonance {\n")
	for _, step := range p {
		fmt.Fprintf(&buff, "  node [shape=rectanble] \"%s\"\n", step)
	}
	for _, step := range p {
		for _, dependantStep := range step.prerequisiteFor {
			fmt.Fprintf(&buff, "  \"%s\" -> \"%s\"\n", step.String(), dependantStep.String())
		}
	}
	fmt.Fprint(&buff, "}\n")
	return buff.String()
}

func (p Plan) executeSteps(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	logger.Info("🛠️  Applying changes")
	if p.Actionable() {
		for _, node := range p {
			err := node.Execute(nestedCtx, hst)
			if err != nil {
				return err
			}
		}
	} else {
		nestedLogger.Infof("👌 Nothing to do")
	}
	return nil
}

func (p Plan) check(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	logger.Info("🔎 Checking state")

	var retErr error
	for _, step := range p {
		for action, resourceDefinitions := range step.ActionResourceDefinitions() {
			if action == ActionSkip || action == ActionDestroy {
				continue
			}
			for _, resourceDefinition := range resourceDefinitions {
				checkResult, err := resourceDefinition.Check(nestedCtx, hst)
				if err != nil {
					return err
				}
				if checkResult {
					nestedLogger.Infof("%s %s", checkResult, resourceDefinition)
				} else {
					nestedLogger.Errorf("%s %s", checkResult, resourceDefinition)
					retErr = errors.New("some resources failed to check immediately after apply, this means either something external changed the state or some resource implementation is broken")
				}
			}
		}
	}
	return retErr
}

// Execute every Step from the Plan.
func (p Plan) Execute(ctx context.Context, hst host.Host) (HostState, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	logger.Info("⚙️  Executing plan")

	hostState := HostState{
		Version: version.GetVersion(),
	}

	// Execute all steps
	if err := p.executeSteps(nestedCtx, hst); err != nil {
		return hostState, err
	}

	// HostState
	for _, step := range p {
		for action, resourceDefinitions := range step.ActionResourceDefinitions() {
			if !action.SaveState() {
				continue
			}
			hostState.ResourceDefinitions = append(
				hostState.ResourceDefinitions, resourceDefinitions...,
			)
		}
	}

	// Check
	if err := p.check(nestedCtx, hst); err != nil {
		return hostState, err
	}

	return hostState, nil
}

func (p Plan) Actionable() bool {
	for _, step := range p {
		if step.StepAction.Actionable() {
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

	if !p.Actionable() {
		nestedLogger.Infof("👌 Nothing to do")
		return
	}

	for _, step := range p {
		if step.Actionable() {
			nestedLogger.Infof("%s", step)
		} else {
			nestedLogger.Debugf("%s", step)
		}
	}

	nestedLogger.WithFields(logrus.Fields{"Digraph": p.Graphviz()}).Debug("Graphviz")
}

func checkResourcesState(
	ctx context.Context,
	hst host.Host,
	savedHostState *HostState,
	resourceBundles ResourceBundles,
) (CheckResults, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	logger.Info("🔎 Checking state")
	checkResults := CheckResults{}
	if savedHostState != nil {
		for _, resourceDefinition := range savedHostState.ResourceDefinitions {
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

	logger.Info("👷 Building apply/refresh plan")
	plan := Plan{}

	var lastBundleLastStep *Step
	for _, resourceBundle := range resourceBundles {
		resourceBundleSteps := []*Step{}
		refresh := false
		var step *Step
		for i, resourceDefinition := range resourceBundle {
			step = &Step{}
			plan = append(plan, step)

			// Dependant on previous bundle
			if i == 0 && lastBundleLastStep != nil {
				lastBundleLastStep.prerequisiteFor = append(lastBundleLastStep.prerequisiteFor, step)
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
			resourceBundleSteps = append(resourceBundleSteps, step)
			if i > 0 {
				dependantStep := resourceBundleSteps[i-1]
				dependantStep.prerequisiteFor = append(dependantStep.prerequisiteFor, step)
			}

			// StepAction
			step.StepAction = StepActionIndividual{
				ResourceDefinition: resourceDefinition,
				Action:             action,
			}

			// Refresh
			if action == ActionApply {
				refresh = true
			}
		}
		lastBundleLastStep = step
	}

	return plan, nil
}

func appendDestroySteps(
	ctx context.Context,
	savedHostState *HostState,
	resourceBundles ResourceBundles,
	plan Plan,
) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, resourceDefinition := range savedHostState.ResourceDefinitions {
		if resourceBundles.HasResourceDefinition(resourceDefinition) {
			continue
		}
		step := &Step{
			StepAction: StepActionIndividual{
				ResourceDefinition: resourceDefinition,
				Action:             ActionDestroy,
			},
			prerequisiteFor: []*Step{plan[0]},
		}
		nestedLogger.Infof("%s", step)
		plan = append(Plan{step}, plan...)
	}

	return plan
}

func mergeSteps(ctx context.Context, plan Plan) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("📦 Merging resources")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	newPlan := Plan{}

	mergedSteps := map[Type]*Step{}
	for _, step := range plan {
		stepActionIndividual := step.StepAction.(StepActionIndividual)
		if !stepActionIndividual.ResourceDefinition.IsMergeableManageableResources() {
			newPlan = append(newPlan, step)
			continue
		}

		stepType := stepActionIndividual.ResourceDefinition.Type()
		mergedStep, ok := mergedSteps[stepType]
		if !ok {
			mergedStep = &Step{
				StepAction: StepActionMerged{},
			}
			newPlan = append(newPlan, mergedStep)
			mergedSteps[stepType] = mergedStep
		}
		stepActionMerged := mergedStep.StepAction.(StepActionMerged)
		stepActionMerged[stepActionIndividual.Action] = append(
			stepActionMerged[stepActionIndividual.Action], stepActionIndividual.ResourceDefinition,
		)
		mergedStep.prerequisiteFor = append(mergedStep.prerequisiteFor, step)
		stepActionIndividual.Action = ActionSkip
		step.StepAction = stepActionIndividual
		newPlan = append(newPlan, step)
	}

	for _, step := range mergedSteps {
		nestedLogger.Infof("%s", step)
	}

	return newPlan
}

// topologicalSort sorts the steps based on their prerequisites. If the graph has cycles, it returns
// error.
func topologicalSort(ctx context.Context, plan Plan) (Plan, error) {
	dependantCount := map[*Step]int{}
	for _, step := range plan {
		if _, ok := dependantCount[step]; !ok {
			dependantCount[step] = 0
		}
		for _, prereq := range step.prerequisiteFor {
			dependantCount[prereq]++
		}
	}

	noDependantsSteps := []*Step{}
	for _, step := range plan {
		if dependantCount[step] == 0 {
			noDependantsSteps = append(noDependantsSteps, step)
		}
	}

	sortedPlan := Plan{}
	for len(noDependantsSteps) > 0 {
		step := noDependantsSteps[0]
		noDependantsSteps = noDependantsSteps[1:]
		sortedPlan = append(sortedPlan, step)
		for _, dependantStep := range step.prerequisiteFor {
			dependantCount[dependantStep]--
			if dependantCount[dependantStep] == 0 {
				noDependantsSteps = append(noDependantsSteps, dependantStep)
			}
		}
	}

	if len(sortedPlan) != len(plan) {
		return nil, errors.New("unable to sort plan: it has cycles")
	}

	return sortedPlan, nil
}

// NewPlanFromResourceBundles calculates the Plan based on a saved HostState and ResourceBundles.
func NewPlanFromResourceBundles(
	ctx context.Context,
	hst host.Host,
	savedHostState *HostState,
	resourceBundles ResourceBundles,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning changes")
	nestedCtx := log.IndentLogger(ctx)

	// Checking state
	checkResults, err := checkResourcesState(nestedCtx, hst, savedHostState, resourceBundles)
	if err != nil {
		return nil, err
	}

	// Build unsorted digraph
	plan, err := buildApplyRefreshPlan(nestedCtx, resourceBundles, checkResults)
	if err != nil {
		return nil, err
	}

	// Append destroy steps
	if savedHostState != nil {
		plan = appendDestroySteps(nestedCtx, savedHostState, resourceBundles, plan)
	}

	// Merge steps
	plan = mergeSteps(nestedCtx, plan)

	// Sort
	plan, err = topologicalSort(nestedCtx, plan)
	if err != nil {
		return nil, err
	}

	return plan, nil
}
