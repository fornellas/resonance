package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// StepAction defines an interface for an action that can be executed from a Step.
type StepAction interface {
	// Execute actions for all bundled resources.
	Execute(ctx context.Context, hst host.Host) error
	String() string
	// Actionable returns whether any action is different from ActionOk or ActionSkip
	Actionable() bool
	// ActionResources returns a map from Action to a slice of Resource.
	ActionResourcesMap() map[Action]Resources
}

// StepActionIndividual is a StepAction which can execute a single Resource.
type StepActionIndividual struct {
	Resource Resource
	Action   Action
}

// Execute Action for the Resource.
func (sai StepActionIndividual) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)

	if sai.Action == ActionOk || sai.Action == ActionSkip {
		logger.Debugf("%s", sai)
		return nil
	}
	individuallyManageableResource := sai.Resource.MustIndividuallyManageableResource()
	name := sai.Resource.MustName()
	state := sai.Resource.State
	switch sai.Action {
	case ActionRefresh:
		refreshableManageableResource, ok := individuallyManageableResource.(RefreshableManageableResource)
		if ok {
			err := refreshableManageableResource.Refresh(ctx, hst, name)
			if err != nil {
				logger.Errorf("💥 %s", sai.Resource)
				return err
			}
			logger.Infof("%s", sai)
		}
	case ActionConfigure:
		if err := individuallyManageableResource.Configure(ctx, hst, name, state); err != nil {
			logger.Errorf("💥 %s", sai.Resource)
			return err
		}
		typeNameStateMap, err := GetTypeNameStateMap(ctx, hst, []TypeName{sai.Resource.TypeName}, false)
		if err != nil {
			logger.Errorf("💥 %s", sai.Resource)
			return err
		}
		currentState, ok := typeNameStateMap[sai.Resource.TypeName]
		if !ok {
			panic(fmt.Errorf("TypeNameStateMap missing %s", sai.Resource.TypeName))
		}
		chunks := Diff(sai.Resource.State, currentState)
		if chunks.HaveChanges() {
			logger.WithField("", chunks.String()).Errorf("💥 %s", sai.Resource)
			return errors.New(
				"likely bug in resource implementationm as state was dirty immediately after applying",
			)
		}
		logger.Infof("%s", sai)
	case ActionDestroy:
		err := individuallyManageableResource.Destroy(ctx, hst, name)
		if err != nil {
			logger.Errorf("💥 %s", sai.Resource)
			return err
		}
		logger.Infof("%s", sai)
	default:
		panic(fmt.Errorf("unexpected action %v", sai.Action))
	}
	return nil
}

func (sai StepActionIndividual) String() string {
	return fmt.Sprintf("%s %s[%s]", sai.Action.Emoji(), sai.Resource.MustType(), sai.Resource.MustName())
}

func (sai StepActionIndividual) Actionable() bool {
	return !(sai.Action == ActionOk || sai.Action == ActionSkip)
}

func (sai StepActionIndividual) ActionResourcesMap() map[Action]Resources {
	return map[Action]Resources{sai.Action: Resources{sai.Resource}}
}

// StepActionMerged is a StepAction which contains multiple merged Resource.
type StepActionMerged map[Action]Resources

// MustMergeableManageableResources returns MergeableManageableResources common to all
// Resource or panics.
func (sam StepActionMerged) MustMergeableManageableResources() MergeableManageableResources {
	var mergeableManageableResources MergeableManageableResources
	for _, resources := range sam {
		for _, resource := range resources {
			if mergeableManageableResources != nil &&
				mergeableManageableResources != resource.MustMergeableManageableResources() {
				panic(fmt.Errorf("%s: mixed resource types", sam))
			}
			mergeableManageableResources = resource.MustMergeableManageableResources()
		}
	}
	if mergeableManageableResources == nil {
		panic(fmt.Errorf("%s is empty", sam))
	}
	return mergeableManageableResources
}

func (sam StepActionMerged) buildParameters() (
	Resources,
	map[Action]map[Name]State,
	Names,
	map[TypeName]Resource,
) {
	checkResources := Resources{}
	configureActionParameters := map[Action]map[Name]State{}
	refreshNames := Names{}
	typeNameResourceMap := map[TypeName]Resource{}
	for action, resources := range sam {
		for _, resource := range resources {
			typeNameResourceMap[resource.TypeName] = resource
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resource.MustName())
			} else {
				add := false
				var state State

				if resource.Destroy {
					state = nil
					add = true
				} else {
					if action == ActionDestroy {
						panic(fmt.Errorf("action is destroy but resonance.Destroy is false: %s", resource))
					}
					if action.Actionable() {
						state = resource.State
						add = true
					}
				}

				if add {
					checkResources = append(checkResources, resource)
					if configureActionParameters[action] == nil {
						configureActionParameters[action] = map[Name]State{}
					}
					configureActionParameters[action][resource.MustName()] = state
				}
			}
		}
	}

	return checkResources, configureActionParameters, refreshNames, typeNameResourceMap
}

func (sam StepActionMerged) Type() Type {
	var tpePtr *Type
	for _, resources := range sam {
		for _, resource := range resources {
			if tpePtr != nil {
				if resource.TypeName.Type() != *tpePtr {
					panic(fmt.Errorf("multiple resource types: %s", sam))
				}
			} else {
				tpe := resource.TypeName.Type()
				tpePtr = &tpe
			}
		}
	}
	if tpePtr == nil {
		panic(fmt.Errorf("empty: %s", sam))
	}
	return *tpePtr
}

// Execute the required Action for each Resource.
func (sam StepActionMerged) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)

	if !sam.Actionable() {
		logger.Debugf("%s", sam)
		return nil
	}

	// Build parameters
	checkResources, configureActionParameters, refreshNames, typeNameResourceMap := sam.buildParameters()

	// ConfigureAll
	if err := sam.MustMergeableManageableResources().ConfigureAll(
		ctx, hst, configureActionParameters,
	); err != nil {
		logger.Errorf("💥 %s", sam.StringNoAction())
		return err
	}

	// Check
	typeNameStateMap, err := GetTypeNameStateMap(ctx, hst, checkResources.TypeNames(), false)
	if err != nil {
		logger.Errorf("💥 %s", sam.StringNoAction())
		return err
	}
	for typeName, currentState := range typeNameStateMap {
		resource, ok := typeNameResourceMap[typeName]
		if !ok {
			panic(fmt.Sprintf("typeNameResourceMap missing %s", typeName))
		}
		chunks := Diff(resource.State, currentState)
		if chunks.HaveChanges() {
			logger.WithField("", chunks.String()).Errorf("💥 %s", typeName)
			return fmt.Errorf(
				"likely bug in resource implementationm as state was dirty immediately after applying: %s",
				typeName,
			)
		}
	}
	logger.Infof("%s", sam)

	// Refresh
	for _, name := range refreshNames {
		refreshableManageableResource, ok := sam.MustMergeableManageableResources().(RefreshableManageableResource)
		if ok {
			err := refreshableManageableResource.Refresh(ctx, hst, name)
			if err != nil {
				logger.Errorf("💥 %s", sam.StringNoAction())
			}
		}
		logger.Infof("%s %s", ActionRefresh.Emoji(), MustNewTypeName(sam.Type(), name))
	}

	return nil
}

func (sam StepActionMerged) String() string {
	var tpe Type
	names := []string{}
	for action, resources := range sam {
		for _, resource := range resources {
			tpe = resource.MustType()
			names = append(names, fmt.Sprintf("%s %s", action.Emoji(), string(resource.MustName())))
		}
	}
	sort.Strings(names)
	return fmt.Sprintf("%s[%s]", tpe, strings.Join(names, ", "))
}

func (sam StepActionMerged) StringNoAction() string {
	var tpe Type
	names := []string{}
	for _, resources := range sam {
		for _, resource := range resources {
			tpe = resource.MustType()
			names = append(names, string(resource.MustName()))
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

func (sam StepActionMerged) ActionResourcesMap() map[Action]Resources {
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

func (s Step) ActionResourcesMap() map[Action]Resources {
	return s.StepAction.ActionResourcesMap()
}

func (s Step) HasTypeName(typeName TypeName) bool {
	for _, resources := range s.ActionResourcesMap() {
		for _, resource := range resources {
			if resource.TypeName == typeName {
				return true
			}
		}
	}
	return false
}

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan struct {
	Steps            []*Step
	TypeNameStateMap TypeNameStateMap
}

// Graphviz returns a DOT directed graph containing the apply plan.
func (p Plan) Graphviz() string {
	var buff bytes.Buffer
	fmt.Fprint(&buff, "digraph resonance {\n")
	for _, step := range p.Steps {
		fmt.Fprintf(&buff, "  node [shape=box] \"%s\"\n", step)
	}
	for _, step := range p.Steps {
		for _, dependantStep := range step.prerequisiteFor {
			fmt.Fprintf(&buff, "  \"%s\" -> \"%s\"\n", step.String(), dependantStep.String())
		}
	}
	fmt.Fprint(&buff, "}\n")
	return buff.String()
}

func topologicalSortPlan(ctx context.Context, steps []*Step) ([]*Step, error) {
	logger := log.GetLogger(ctx)
	logger.Debug("Sorting plan")

	dependantCount := map[*Step]int{}
	for _, step := range steps {
		if _, ok := dependantCount[step]; !ok {
			dependantCount[step] = 0
		}
		for _, prereq := range step.prerequisiteFor {
			dependantCount[prereq]++
		}
	}

	noDependantsSteps := []*Step{}
	for _, step := range steps {
		if dependantCount[step] == 0 {
			noDependantsSteps = append(noDependantsSteps, step)
		}
	}

	sortedSteps := []*Step{}
	for len(noDependantsSteps) > 0 {
		step := noDependantsSteps[0]
		noDependantsSteps = noDependantsSteps[1:]
		sortedSteps = append(sortedSteps, step)
		for _, dependantStep := range step.prerequisiteFor {
			dependantCount[dependantStep]--
			if dependantCount[dependantStep] == 0 {
				noDependantsSteps = append(noDependantsSteps, dependantStep)
			}
		}
	}

	if len(sortedSteps) != len(steps) {
		return nil, errors.New("unable to sort steps: try 'graph' command to debug it")
	}

	return sortedSteps, nil
}

func (p Plan) executeSteps(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	logger.Info("🛠️  Applying changes")
	if p.Actionable() {
		steps, err := topologicalSortPlan(nestedCtx, p.Steps)
		if err != nil {
			return err
		}

		for _, step := range steps {
			if !step.Actionable() {
				continue
			}
			err := step.Execute(nestedCtx, hst)
			if err != nil {
				return err
			}
		}
	} else {
		nestedLogger.Infof("👌 Nothing to do")
	}
	return nil
}

// Execute every Step from the Plan.
func (p Plan) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	logger.Info("⚙️  Executing plan")

	if err := p.executeSteps(nestedCtx, hst); err != nil {
		return err
	}

	return nil
}

func (p Plan) Actionable() bool {
	for _, step := range p.Steps {
		if step.StepAction.Actionable() {
			return true
		}
	}
	return false
}

func (p Plan) printResource(
	ctx context.Context,
	step *Step,
	resource Resource,
	resources Resources,
) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	currentState, ok := p.TypeNameStateMap[resource.TypeName]
	if !ok {
		panic(fmt.Sprintf("State not found at TypeNameStateMap: %s", resource.TypeName))
	}

	chunks := Diff(currentState, resource.State)

	var action Action
	for actionResources, stepResources := range step.ActionResourcesMap() {
		for _, stepResource := range stepResources {
			if stepResource.TypeName == resource.TypeName {
				action = actionResources
			}
		}
	}
	if action == ActionNone {
		panic(fmt.Sprintf("can not find action: %s", resource))
	}

	if chunks.HaveChanges() {
		if len(resources) > 1 {
			nestedLogger.WithField("", chunks.String()).Infof("%s %s", action.Emoji(), resource)
		} else {
			logger.WithField("", chunks.String()).Infof("%s %s", action.Emoji(), resource)
		}
	} else {
		if len(resources) > 1 {
			logger.Infof("  %s %s", action.Emoji(), resource)
		} else {
			logger.Infof("%s %s", action.Emoji(), resource)
		}
	}
}

// Print the whole plan
func (p Plan) Print(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Plan")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	if !p.Actionable() {
		nestedLogger.Infof("👌 Nothing to do")
	}

	steps, err := topologicalSortPlan(nestedCtx, p.Steps)
	if err != nil {
		return err
	}

	for _, step := range steps {
		if step.Actionable() {
			resources := Resources{}
			for _, stepResources := range step.ActionResourcesMap() {
				resources = append(resources, stepResources...)
			}
			sort.Sort(resources)

			if len(resources) > 1 {
				nestedLogger.Infof("%s", step)
			}

			for _, resource := range resources {
				p.printResource(nestedCtx, step, resource, resources)
			}
		} else {
			nestedLogger.Debugf("%s", step)
		}
	}

	return nil
}

func (p Plan) addBundleSteps(
	ctx context.Context,
	hst host.Host,
	newBundle Bundle,
	intendedAction Action,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("👷 Building plan")

	var lastBundleLastStep *Step
	for _, newResources := range newBundle {
		bundleSteps := []*Step{}
		refresh := false
		var step *Step
		for i, newResource := range newResources {
			step = &Step{}
			p.Steps = append(p.Steps, step)

			// Dependant on previous resources
			if i == 0 && lastBundleLastStep != nil {
				lastBundleLastStep.prerequisiteFor = append(lastBundleLastStep.prerequisiteFor, step)
			}

			// Action
			action := intendedAction
			currentState, ok := p.TypeNameStateMap[newResource.TypeName]
			if !ok {
				panic(fmt.Errorf("State missing from TypeNameStateMap: %s", newResource.TypeName))
			}

			chunks := Diff(currentState, newResource.State)

			clean := !chunks.HaveChanges()

			if newResource.Destroy {
				if clean {
					action = ActionOk
				} else {
					action = ActionDestroy
				}
			}
			if action != ActionDestroy {
				if clean {
					if refresh && newResource.Refreshable() {
						action = ActionRefresh
					} else {
						action = ActionOk
					}
				} else {
					action = ActionConfigure
				}
			}

			// Prerequisites
			bundleSteps = append(bundleSteps, step)
			if i > 0 {
				dependantStep := bundleSteps[i-1]
				dependantStep.prerequisiteFor = append(dependantStep.prerequisiteFor, step)
			}

			// StepAction
			step.StepAction = StepActionIndividual{
				Resource: newResource,
				Action:   action,
			}

			// Refresh
			if action == ActionConfigure {
				refresh = true
			}
		}
		lastBundleLastStep = step
	}

	return p, nil
}

func addDestroyStepsToPlan(
	ctx context.Context,
	steps []*Step,
	newBundle Bundle,
	previousBundle Bundle,
) []*Step {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	previousResources := previousBundle.Resources()
	for previousIdx, previousResource := range previousResources {
		if newBundle.HasTypeName(previousResource.TypeName) {
			continue
		}

		prerequisiteFor := []*Step{}
		if previousIdx+1 < len(previousResources) {
			for _, previousResourcePrereqFor := range previousResources[previousIdx+1:] {
				for _, step := range steps {
					if step.HasTypeName(previousResourcePrereqFor.TypeName) {
						prerequisiteFor = []*Step{step}
						break
					}
				}
				if len(prerequisiteFor) > 0 {
					break
				}
			}
		}

		resource := previousResource
		resource.State = nil
		resource.Destroy = true

		step := &Step{
			StepAction: StepActionIndividual{
				Resource: resource,
				Action:   ActionDestroy,
			},
			prerequisiteFor: prerequisiteFor,
		}
		nestedLogger.Infof("%s", step)
		steps = append(steps, step)
	}
	return steps
}

func mergePlanSteps(ctx context.Context, steps []*Step) []*Step {
	logger := log.GetLogger(ctx)
	logger.Info("📦 Merging resources")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	newSteps := []*Step{}

	mergedSteps := map[Type]*Step{}
	for _, step := range steps {
		stepActionIndividual := step.StepAction.(StepActionIndividual)
		if !stepActionIndividual.Resource.IsMergeableManageableResources() {
			newSteps = append(newSteps, step)
			continue
		}

		stepType := stepActionIndividual.Resource.MustType()
		mergedStep, ok := mergedSteps[stepType]
		if !ok {
			mergedStep = &Step{
				StepAction: StepActionMerged{},
			}
			newSteps = append(newSteps, mergedStep)
			mergedSteps[stepType] = mergedStep
		}
		stepActionMerged := mergedStep.StepAction.(StepActionMerged)
		stepActionMerged[stepActionIndividual.Action] = append(
			stepActionMerged[stepActionIndividual.Action], stepActionIndividual.Resource,
		)
		mergedStep.prerequisiteFor = append(mergedStep.prerequisiteFor, step)
		stepActionIndividual.Action = ActionSkip
		step.StepAction = stepActionIndividual
		newSteps = append(newSteps, step)
	}

	for _, step := range mergedSteps {
		nestedLogger.Infof("%s", step)
	}

	return newSteps
}

func NewPlan(
	ctx context.Context,
	hst host.Host,
	newBundle Bundle,
	previousBundle *Bundle,
	typeNameStateMap TypeNameStateMap,
	intendedAction Action,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning changes")
	nestedCtx := log.IndentLogger(ctx)

	// Plan
	plan := Plan{
		Steps:            []*Step{},
		TypeNameStateMap: typeNameStateMap,
	}

	// Add Bundle Steps
	var err error
	plan, err = plan.addBundleSteps(nestedCtx, hst, newBundle, intendedAction)
	if err != nil {
		return Plan{}, err
	}

	// Prepend destroy steps
	if previousBundle != nil {
		plan.Steps = addDestroyStepsToPlan(nestedCtx, plan.Steps, newBundle, *previousBundle)
	}

	// Merge steps
	plan.Steps = mergePlanSteps(nestedCtx, plan.Steps)

	return plan, nil
}

func NewRollbackBundle(
	newBundle Bundle,
	previousBundle *Bundle,
	typeNameStateMap TypeNameStateMap,
	intendedAction Action,
) Bundle {
	if previousBundle == nil {
		previousBundle = &Bundle{}
	}

	rollbackBundle := *previousBundle

	rollbackResources := Resources{}

	for _, newResources := range newBundle {
		for _, newResource := range newResources {
			if previousBundle.HasTypeName(newResource.TypeName) {
				continue
			}
			state, ok := typeNameStateMap[newResource.TypeName]
			if !ok {
				panic(fmt.Sprintf("state missing from TypeNameResourceStateMap: %s", newResource.TypeName))
			}
			rollbackResource := NewResource(
				newResource.TypeName, state, state == nil,
			)

			// FIXME insert at rollbackBundle at the correct place, immediately before any
			// of the following newResources's position at rollbackBundle

			rollbackResources = append(rollbackResources, rollbackResource)

		}
	}

	if len(rollbackResources) > 0 {
		rollbackBundle = append(Bundle{rollbackResources}, rollbackBundle...)
	}

	return rollbackBundle
}
