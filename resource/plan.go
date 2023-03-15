package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

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
			}
			return err
		}
	case ActionApply:
		if err := individuallyManageableResource.Apply(ctx, hst, name, state); err != nil {
			logger.Errorf("💥 %s", sai.Resource)
			return err
		}
		diffHasChanges, _, _, err := sai.Resource.CheckState(ctx, hst, nil)
		if err != nil {
			logger.Errorf("💥 %s", sai.Resource)
			return err
		}
		if diffHasChanges {
			logger.Errorf("💥 %s", sai.Resource)
			return errors.New(
				"likely bug in resource implementationm as state was dirty immediately after applying",
			)
		}
	case ActionDestroy:
		err := individuallyManageableResource.Destroy(ctx, hst, name)
		if err != nil {
			logger.Errorf("💥 %s", sai.Resource)
		}
		return err
	default:
		panic(fmt.Errorf("unexpected action %v", sai.Action))
	}
	return nil
}

func (sai StepActionIndividual) String() string {
	return fmt.Sprintf("%s[%s %s]", sai.Resource.MustType(), sai.Action.Emoji(), sai.Resource.MustName())
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

// Execute the required Action for each Resource.
func (sam StepActionMerged) Execute(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)

	if !sam.Actionable() {
		logger.Debugf("%s", sam)
		return nil
	}

	checkResources := Resources{}
	configureActionParameters := map[Action]map[Name]State{}
	refreshNames := []Name{}
	for action, resources := range sam {
		for _, resource := range resources {
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resource.MustName())
			} else {
				if configureActionParameters[action] == nil {
					configureActionParameters[action] = map[Name]State{}
				}
				if !resource.Destroy {
					configureActionParameters[action][resource.MustName()] = resource.State
				} else {
					configureActionParameters[action][resource.MustName()] = nil
				}
			}
			if action != ActionDestroy {
				checkResources = append(checkResources, resource)
			}
		}
	}

	if err := sam.MustMergeableManageableResources().ConfigureAll(
		ctx, hst, configureActionParameters,
	); err != nil {
		logger.Errorf("💥%s", sam.StringNoAction())
		return err
	}

	for _, resource := range checkResources {
		diffHasChanges, _, _, err := resource.CheckState(ctx, hst, nil)
		if err != nil {
			logger.Errorf("💥%s", sam.StringNoAction())
			return err
		}
		if diffHasChanges {
			logger.Errorf("💥%s", sam.StringNoAction())
			return errors.New(
				"likely bug in resource implementationm as state was dirty immediately after applying",
			)
		}
	}

	for _, name := range refreshNames {
		refreshableManageableResource, ok := sam.MustMergeableManageableResources().(RefreshableManageableResource)
		if ok {
			err := refreshableManageableResource.Refresh(ctx, hst, name)
			if err != nil {
				logger.Errorf("💥%s", sam.StringNoAction())
			}
			return err
		}
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

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan struct {
	Steps                 []*Step
	InitialResourcesState ResourcesState
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

func (p Plan) executeSteps(ctx context.Context, hst host.Host) error {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	logger.Info("🛠️  Applying changes")
	if p.Actionable() {
		for _, step := range p.Steps {
			if !step.Actionable() {
				continue
			}
			err := step.Execute(nestedCtx, hst)
			if err != nil {
				return err
			}
			nestedLogger.Infof("%s", step)
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

// Print the whole plan
func (p Plan) Print(ctx context.Context) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Plan")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	nestedNestedCtx := log.IndentLogger(nestedCtx)
	nestedNestedLogger := log.GetLogger(nestedNestedCtx)

	if !p.Actionable() {
		nestedLogger.Infof("👌 Nothing to do")
	}

	for _, step := range p.Steps {
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
				diffs, ok := p.InitialResourcesState.TypeNameDiffsMap[resource.TypeName]
				if !ok {
					panic(fmt.Sprintf("diffs not found at InitialResourcesState: %s", resource))
				}

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

				if DiffsHasChanges(diffs) {
					diffMatchPatch := diffmatchpatch.New()
					if len(resources) > 1 {
						nestedNestedLogger.WithField("", diffMatchPatch.DiffPrettyText(diffs)).
							Infof("%s %s", action.Emoji(), resource)
					} else {
						nestedLogger.WithField("", diffMatchPatch.DiffPrettyText(diffs)).
							Infof("%s %s", action.Emoji(), resource)
					}
				}
			}
		} else {
			nestedLogger.Debugf("%s", step)
		}
	}
}

func (p Plan) addBundleSteps(
	ctx context.Context,
	bundle Bundle,
	intendedAction Action,
) Plan {
	logger := log.GetLogger(ctx)

	logger.Info("👷 Building plan")

	var lastBundleLastStep *Step
	for _, resources := range bundle {
		bundleSteps := []*Step{}
		refresh := false
		var step *Step
		for i, resource := range resources {
			step = &Step{}
			p.Steps = append(p.Steps, step)

			// Dependant on previous resources
			if i == 0 && lastBundleLastStep != nil {
				lastBundleLastStep.prerequisiteFor = append(lastBundleLastStep.prerequisiteFor, step)
			}

			// Action
			action := intendedAction
			if resource.Destroy {
				action = ActionDestroy
			}
			if action != ActionDestroy {
				cleanState, ok := p.InitialResourcesState.TypeNameCleanMap[resource.TypeName]
				if !ok {
					panic(fmt.Errorf("%v missing check result", resource))
				}
				if cleanState {
					if refresh && resource.Refreshable() {
						action = ActionRefresh
					} else {
						action = ActionOk
					}
				} else {
					action = ActionApply
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
				Resource: resource,
				Action:   action,
			}

			// Refresh
			if action == ActionApply {
				refresh = true
			}
		}
		lastBundleLastStep = step
	}

	return p
}

func prependDestroyStepsToPlan(
	ctx context.Context,
	steps []*Step,
	bundle Bundle,
	savedHostState HostState,
) []*Step {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, resource := range savedHostState.Resources {
		if bundle.HasTypeName(resource.TypeName) {
			continue
		}
		prerequisiteFor := []*Step{}
		if len(steps) > 0 {
			prerequisiteFor = append(prerequisiteFor, steps[0])
		}
		step := &Step{
			StepAction: StepActionIndividual{
				Resource: resource,
				Action:   ActionDestroy,
			},
			prerequisiteFor: prerequisiteFor,
		}
		nestedLogger.Infof("%s", step)
		steps = append([]*Step{step}, steps...)
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

func topologicalSortPlan(ctx context.Context, steps []*Step) ([]*Step, error) {
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
		return nil, errors.New("unable to sort steps: it has cycles")
	}

	return sortedSteps, nil
}

func NewPlanFromSavedStateAndBundle(
	ctx context.Context,
	hst host.Host,
	bundle Bundle,
	savedHostState *HostState,
	initialResourcesState ResourcesState,
	intendedAction Action,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning changes")
	nestedCtx := log.IndentLogger(ctx)

	// Plan
	plan := Plan{
		Steps:                 []*Step{},
		InitialResourcesState: initialResourcesState,
	}

	// Add Bundle Steps
	plan = plan.addBundleSteps(nestedCtx, bundle, intendedAction)

	// Prepend destroy steps
	if savedHostState != nil {
		plan.Steps = prependDestroyStepsToPlan(nestedCtx, plan.Steps, bundle, *savedHostState)
	}

	// Merge steps
	plan.Steps = mergePlanSteps(nestedCtx, plan.Steps)

	// Sort
	var err error
	plan.Steps, err = topologicalSortPlan(nestedCtx, plan.Steps)
	if err != nil {
		return Plan{}, err
	}

	return plan, nil
}

func prependDestroyStepsFromPlanBundleToPlan(
	ctx context.Context,
	steps []*Step,
	bundles, planBundle Bundle,
) []*Step {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, planResources := range planBundle {
		for _, resource := range planResources {
			if bundles.HasTypeName(resource.TypeName) {
				continue
			}
			prerequisiteFor := []*Step{}
			if len(steps) > 0 {
				prerequisiteFor = append(prerequisiteFor, steps[0])
			}
			step := &Step{
				StepAction: StepActionIndividual{
					Resource: resource,
					Action:   ActionDestroy,
				},
				prerequisiteFor: prerequisiteFor,
			}
			nestedLogger.Infof("%s", step)
			steps = append([]*Step{step}, steps...)
		}
	}
	return steps
}

func NewRollbackPlan(
	ctx context.Context,
	hst host.Host,
	bundle Bundle,
	initialResources Resources,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning rollback")
	nestedCtx := log.IndentLogger(ctx)
	rollbackBundle := NewBundleFromResources(initialResources)

	// ResourcesState
	initialResourcesState, err := NewResourcesState(ctx, hst, initialResources)
	if err != nil {
		logger.Fatal(err)
	}

	// Plan
	plan := Plan{
		Steps:                 []*Step{},
		InitialResourcesState: initialResourcesState,
	}

	// Add Bundle Steps
	plan = plan.addBundleSteps(nestedCtx, rollbackBundle, ActionNone)

	// Prepend destroy steps
	plan.Steps = prependDestroyStepsFromPlanBundleToPlan(nestedCtx, plan.Steps, rollbackBundle, bundle)

	// Merge steps
	plan.Steps = mergePlanSteps(nestedCtx, plan.Steps)

	// Sort
	plan.Steps, err = topologicalSortPlan(nestedCtx, plan.Steps)
	if err != nil {
		return Plan{}, err
	}

	return plan, nil
}
