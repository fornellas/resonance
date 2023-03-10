package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/version"
)

// Name is a name that globally uniquely identifies a resource instance of a given type.
// Eg: for File type a Name would be the file absolute path such as /etc/issue.
type Name string

// Action to be executed for a resource.
type Action int

const (
	ActionNone Action = iota
	// ActionOk means state is as expected
	ActionOk
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

// StateParameters is a Type specific interface for defining resource state as configured by users.
type StateParameters interface {
	// Validate whether the parameters are OK.
	Validate() error
}

// InternalState is a Type specific interface for defining resource state that's internal, often
// required to enable rollbacks.
type InternalState interface{}

// FullState is the full state of a resource.
type FullState struct {
	StateParameters StateParameters `yaml:"parameters"`
	InternalState   InternalState   `yaml:"internal"`
}

type fullStateUnmarshalSchema struct {
	StateParameters yaml.Node `yaml:"parameters"`
	InternalState   yaml.Node `yaml:"internal"`
}

func (fs FullState) String() string {
	bytes, err := yaml.Marshal(&fs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// Parameters for ManageableResource.
type Parameters map[Name]StateParameters

// DiffsHasChanges return true when the diff contains no changes.
func DiffsHasChanges(diffs []diffmatchpatch.Diff) bool {
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			return true
		}
	}
	return false
}

// ManageableResource defines a common interface for managing resource state.
type ManageableResource interface {
	// ValidateName validates the name of the resource
	ValidateName(name Name) error

	// GetFullState gets the full state of the resource.
	// If resource is not present, then returns nil.
	GetFullState(ctx context.Context, hst host.Host, name Name) (*FullState, error)

	// DiffStates compares the desired StateParameters against current FullState.
	// If StateParameters is met by FullState, return an empty slice; otherwise,
	// return Diff's from FullState to StateParameters showing what needs change.
	DiffStates(
		ctx context.Context, hst host.Host,
		desiredStateParameters StateParameters, currentFullState FullState,
	) ([]diffmatchpatch.Diff, error)
}

// DiffManageableResourceState diffs whether the resource is at the desired state.
// When changes are pending returns true, otherwise false.
func DiffManageableResourceState(
	ctx context.Context,
	hst host.Host,
	manageableResource ManageableResource,
	stateParameters StateParameters,
	fullState FullState,
) (bool, []diffmatchpatch.Diff, error) {
	diffs, err := manageableResource.DiffStates(ctx, hst, stateParameters, fullState)
	return DiffsHasChanges(diffs), diffs, err
}

func Diff(a, b interface{}) []diffmatchpatch.Diff {
	aBytes, err := yaml.Marshal(a)
	if err != nil {
		panic(err)
	}
	aStr := string(aBytes)

	bBytes, err := yaml.Marshal(b)
	if err != nil {
		panic(err)
	}
	bStr := string(bBytes)

	return diffmatchpatch.New().DiffMain(aStr, bStr, false)
}

// ManageableResourceStateHasPendingChanges whether the resource is at the desired state.
// When changes are pending returns true, otherwise false.
func ManageableResourceStateHasPendingChanges(
	ctx context.Context,
	manageableResource ManageableResource,
	hst host.Host,
	name Name,
	stateParameters *StateParameters,
) (bool, []diffmatchpatch.Diff, *FullState, error) {
	logger := log.GetLogger(ctx)

	typeName := MustNewTypeNameFromManageableResourceName(manageableResource, name)

	fullState, err := manageableResource.GetFullState(ctx, hst, name)
	if err != nil {
		logger.Errorf("💥%s", typeName)
		return false, []diffmatchpatch.Diff{}, nil, err
	}
	if fullState == nil {
		if stateParameters != nil {
			return true, Diff(nil, stateParameters), nil, nil
		} else {
			return false, []diffmatchpatch.Diff{}, nil, nil
		}
	}

	if stateParameters == nil {
		return true, Diff(fullState, nil), nil, nil
	}
	diffHasChanges, diffs, err := DiffManageableResourceState(
		ctx, hst, manageableResource, *stateParameters, *fullState,
	)
	if err != nil {
		logger.Errorf("💥%s", typeName)
		return false, []diffmatchpatch.Diff{}, nil, err
	}

	if diffHasChanges {
		diffMatchPatch := diffmatchpatch.New()
		logger.WithField("", diffMatchPatch.DiffPrettyText(diffs)).
			Errorf("%s", typeName)
		return true, diffs, fullState, nil
	} else {
		logger.Infof("✅%s", typeName)
		return false, diffs, fullState, nil
	}
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

	// Apply configures the resource to given state.
	// Must be idempotent.
	Apply(ctx context.Context, hst host.Host, name Name, stateParameters StateParameters) error

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

	// ConfigureAll configures all resource to given state.
	// Must be idempotent.
	ConfigureAll(ctx context.Context, hst host.Host, actionParameters map[Action]Parameters) error
}

// Type is the name of the resource type.
type Type string

// IndividuallyManageableResourceTypeMap maps Type to IndividuallyManageableResource.
var IndividuallyManageableResourceTypeMap = map[Type]IndividuallyManageableResource{}

// MergeableManageableResourcesTypeMap maps Type to MergeableManageableResources.
var MergeableManageableResourcesTypeMap = map[Type]MergeableManageableResources{}

// ManageableResourcesStateParametersMap maps Type to its StateParameters.
var ManageableResourcesStateParametersMap = map[Type]StateParameters{}

// ManageableResourcesInternalStateMap maps Type to its InternalState.
var ManageableResourcesInternalStateMap = map[Type]InternalState{}

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

func NewTypeFromManageableResource(manageableResource ManageableResource) Type {
	return Type(reflect.TypeOf(manageableResource).Name())
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

func (tn TypeName) MustType() Type {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe
}

func (tn TypeName) MustName() Name {
	_, name, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return name
}

// ManageableResource returns an instance for the resource type.
func (tn TypeName) MustManageableResource() ManageableResource {
	tpe, _, err := tn.typeName()
	if err != nil {
		panic(err)
	}
	return tpe.ManageableResource()
}

func MustNewTypeNameFromStr(typeNameStr string) TypeName {
	typeName := TypeName(typeNameStr)
	if err := typeName.Validate(); err != nil {
		panic(err)
	}
	return typeName
}

func MustNewTypeNameFromManageableResourceName(manageableResource ManageableResource, name Name) TypeName {
	return MustNewTypeNameFromStr(
		fmt.Sprintf("%s[%s]", NewTypeFromManageableResource(manageableResource), name),
	)
}

// ResourceKey is a unique identifier for a Resource that
// can be used as keys in maps.
type ResourceKey string

func (rk ResourceKey) MustTypeName() TypeName {
	return MustNewTypeNameFromStr(string(rk))
}

func (rk ResourceKey) MustManageableResource() ManageableResource {
	return rk.MustTypeName().MustManageableResource()
}

func (rk ResourceKey) MustName() Name {
	return rk.MustTypeName().MustName()
}

func (rk ResourceKey) MustType() Type {
	return rk.MustTypeName().MustType()
}

// Resource holds a single resource.
type Resource struct {
	TypeName TypeName `yaml:"resource"`
	// When nil, means resource is to be destroyed
	StateParameters    *StateParameters   `yaml:"state"`
	ManageableResource ManageableResource `yaml:"-"`
}

type resourceUnmarshalSchema struct {
	TypeName            TypeName  `yaml:"resource"`
	NodeStateParameters yaml.Node `yaml:"state"`
}

// FIXME should not panic
func (r *Resource) UnmarshalYAML(node *yaml.Node) error {
	var unmarshalSchema resourceUnmarshalSchema
	node.KnownFields(true)
	if err := node.Decode(&unmarshalSchema); err != nil {
		return err
	}

	manageableResource := unmarshalSchema.TypeName.MustManageableResource()
	tpe := unmarshalSchema.TypeName.MustType()
	name := unmarshalSchema.TypeName.MustName()
	if err := manageableResource.ValidateName(name); err != nil {
		return err
	}

	stateParameters, ok := ManageableResourcesStateParametersMap[tpe]
	if !ok {
		panic(fmt.Errorf("Type %s missing from ManageableResourcesStateParametersMap", tpe))
	}
	stateParametersType := reflect.ValueOf(stateParameters).Type()
	stateParametersValue := reflect.New(stateParametersType)
	stateParameters = stateParametersValue.Interface().(StateParameters)
	var stateParametersPtr *StateParameters
	err := unmarshalSchema.NodeStateParameters.Decode(stateParameters)
	if err != nil {
		return err
	}
	if err := stateParameters.Validate(); err != nil {
		return err
	}
	stateParametersPtr = &stateParameters

	*r = Resource{
		TypeName:           unmarshalSchema.TypeName,
		StateParameters:    stateParametersPtr,
		ManageableResource: manageableResource,
	}
	return nil
}

func (r Resource) MustType() Type {
	return r.TypeName.MustType()
}

func (r Resource) MustName() Name {
	return r.TypeName.MustName()
}

func (r Resource) String() string {
	return string(r.TypeName)
}

func (r Resource) ResourceKey() ResourceKey {
	return ResourceKey(r.String())
}

// Refreshable returns whether the resource is refreshable or not.
func (r Resource) Refreshable() bool {
	_, ok := r.ManageableResource.(RefreshableManageableResource)
	return ok
}

// IsIndividuallyManageableResource returns true only if ManageableResource is of type IndividuallyManageableResource.
func (r Resource) IsIndividuallyManageableResource() bool {
	_, ok := r.ManageableResource.(IndividuallyManageableResource)
	return ok
}

// MustIndividuallyManageableResource returns IndividuallyManageableResource from ManageableResource or
// panics if it isn't of the required type.
func (r Resource) MustIndividuallyManageableResource() IndividuallyManageableResource {
	individuallyManageableResource, ok := r.ManageableResource.(IndividuallyManageableResource)
	if !ok {
		panic(fmt.Errorf("%s is not IndividuallyManageableResource", r))
	}
	return individuallyManageableResource
}

// IsMergeableManageableResources returns true only if ManageableResource is of type MergeableManageableResources.
func (r Resource) IsMergeableManageableResources() bool {
	_, ok := r.ManageableResource.(MergeableManageableResources)
	return ok
}

// MustMergeableManageableResources returns MergeableManageableResources from ManageableResource or
// panics if it isn't of the required type.
func (r Resource) MustMergeableManageableResources() MergeableManageableResources {
	mergeableManageableResources, ok := r.ManageableResource.(MergeableManageableResources)
	if !ok {
		panic(fmt.Errorf("%s is not MergeableManageableResources", r))
	}
	return mergeableManageableResources
}

func MustNewResource(typeName TypeName, stateParameters *StateParameters) Resource {
	return Resource{
		TypeName:           typeName,
		StateParameters:    stateParameters,
		ManageableResource: typeName.MustManageableResource(),
	}
}

// Bundle is the schema used to declare multiple resources at a single file.
type Bundle []Resource

func (b Bundle) Validate() error {
	resourceMap := map[TypeName]bool{}
	for _, resource := range b {
		if _, ok := resourceMap[resource.TypeName]; ok {
			return fmt.Errorf("duplicate resource %s", resource.TypeName)
		}
		resourceMap[resource.TypeName] = true
	}
	return nil
}

// LoadBundle loads resources from given Yaml file path.
func LoadBundle(ctx context.Context, path string) (Bundle, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("%s", path)
	f, err := os.Open(path)
	if err != nil {
		return Bundle{}, fmt.Errorf("failed to load resource file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	bundle := Bundle{}

	for {
		docBundle := Bundle{}
		if err := decoder.Decode(&docBundle); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Bundle{}, fmt.Errorf("failed to load resource file: %s: %w", path, err)
		}
		if err := docBundle.Validate(); err != nil {
			return bundle, fmt.Errorf("resource file validation failed: %s: %w", path, err)
		}
		bundle = append(bundle, docBundle...)
	}

	return bundle, nil
}

// Bundles holds all resources for a host.
type Bundles []Bundle

// GetCleanStateMap returns a map of whether resources have a clean state or not.
func (bs Bundles) GetCleanStateMap(
	ctx context.Context,
	hst host.Host,
) (map[ResourceKey]bool, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	cleanStateMap := map[ResourceKey]bool{}

	for _, bundle := range bs {
		for _, resource := range bundle {
			currentFullState, err := resource.ManageableResource.GetFullState(
				nestedCtx, hst, resource.MustName(),
			)
			if err != nil {
				return nil, err
			}

			cleanState := false
			if currentFullState != nil {
				if resource.StateParameters == nil {
					nestedLogger.WithField("", "Resource to be destroyed").
						Infof("%s %s", ActionDestroy.Emoji(), resource)
				} else {
					diffHasChanges, diffs, err := DiffManageableResourceState(
						nestedCtx,
						hst,
						resource.ManageableResource,
						*resource.StateParameters,
						*currentFullState,
					)
					if err != nil {
						return nil, err
					}
					cleanState = !diffHasChanges
					if diffHasChanges {
						diffMatchPatch := diffmatchpatch.New()
						nestedLogger.WithField("", diffMatchPatch.DiffPrettyText(diffs)).
							Infof("%s %s", ActionApply.Emoji(), resource)
					}
				}

			} else {
				if resource.StateParameters == nil {
					cleanState = true
				} else {
					diffMatchPatch := diffmatchpatch.New()
					nestedLogger.WithField("", diffMatchPatch.DiffPrettyText(Diff("", resource.StateParameters))).
						Infof("%s %s", ActionApply.Emoji(), resource)

				}
			}

			if cleanState {
				nestedLogger.Infof("%s %s", ActionOk.Emoji(), resource)
			}

			cleanStateMap[resource.ResourceKey()] = cleanState
		}
	}

	return cleanStateMap, nil
}

func (bs Bundles) GetHostState(
	ctx context.Context, hst host.Host, excludeResourceKeys []ResourceKey,
) (HostState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Reading host state")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	nestedNestedCtx := log.IndentLogger(nestedCtx)
	hostState := HostState{
		Version:            version.GetVersion(),
		ResourceFullStates: []ResourceFullState{},
	}
	for _, bundle := range bs {
		for _, resource := range bundle {
			for _, excludeResourceKey := range excludeResourceKeys {
				if excludeResourceKey == resource.ResourceKey() {
					continue
				}
			}
			nestedLogger.Infof("%s", resource)
			fullState, err := resource.ManageableResource.GetFullState(
				nestedNestedCtx, hst, resource.MustName(),
			)
			if err != nil {
				return hostState, err
			}
			hostState.ResourceFullStates = append(hostState.ResourceFullStates, ResourceFullState{
				ResourceKey: resource.ResourceKey(),
				FullState:   fullState,
			})
		}
	}
	return hostState, nil
}

func (bs Bundles) Validate() error {
	resourceMap := map[TypeName]bool{}

	for _, bundle := range bs {
		for _, resource := range bundle {
			if _, ok := resourceMap[resource.TypeName]; ok {
				return fmt.Errorf("duplicate resource %s", resource.TypeName)
			}
			resourceMap[resource.TypeName] = true
		}
	}
	return nil
}

// HasResourceKey returns true if Resource is contained at Bundles.
func (bs Bundles) HasResourceKey(resourceKey ResourceKey) bool {
	for _, bundle := range bs {
		for _, rd := range bundle {
			if rd.ResourceKey() == resourceKey {
				return true
			}
		}
	}
	return false
}

// LoadBundles search for .yaml files at root, each having the Bundle schema,
// loads and returns all of them.
// Bundles is sorted by alphabetical order.
func LoadBundles(ctx context.Context, root string) (Bundles, error) {
	logger := log.GetLogger(ctx)
	logger.Infof("📂 Loading resources from %s", root)
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	bundles := Bundles{}

	paths := []string{}
	if err := filepath.Walk(root, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			nestedLogger.Debugf("Skipping %s", path)
			return nil
		}
		nestedLogger.Debugf("Found resources file %s", path)
		paths = append(paths, path)
		return nil
	}); err != nil {
		return bundles, err
	}
	sort.Strings(paths)

	for _, path := range paths {
		bundle, err := LoadBundle(nestedCtx, path)
		if err != nil {
			return bundles, err
		}
		bundles = append(bundles, bundle)
	}

	if err := bundles.Validate(); err != nil {
		return bundles, err
	}

	return bundles, nil
}

// NewBundlesFromHostState creates a single bundle from a HostState.
func NewBundlesFromHostState(hostState *HostState) Bundles {
	if hostState == nil {
		return Bundles{}
	}
	bundle := Bundle{}
	for _, resourceFullState := range hostState.ResourceFullStates {
		var stateParameters *StateParameters
		if resourceFullState.FullState != nil {
			stateParameters = &resourceFullState.FullState.StateParameters
		}
		bundle = append(bundle, MustNewResource(
			resourceFullState.ResourceKey.MustTypeName(),
			stateParameters,
		))
	}
	return Bundles{bundle}
}

type ResourceFullState struct {
	ResourceKey ResourceKey `yaml:"resource"`
	FullState   *FullState  `yaml:"full_state"`
}

type resourceFullStateUnmarshalSchema struct {
	ResourceKey ResourceKey               `yaml:"resource"`
	FullState   *fullStateUnmarshalSchema `yaml:"full_state"`
}

func (rfs *ResourceFullState) UnmarshalYAML(node *yaml.Node) error {
	var resourceFullStateSchema resourceFullStateUnmarshalSchema
	node.KnownFields(true)
	if err := node.Decode(&resourceFullStateSchema); err != nil {
		return err
	}

	resourceKey := resourceFullStateSchema.ResourceKey

	var fullState *FullState

	if resourceFullStateSchema.FullState != nil {
		tpe := resourceKey.MustType()

		// StateParameters
		stateParameters, ok := ManageableResourcesStateParametersMap[tpe]
		if !ok {
			panic(fmt.Errorf("Type %s missing from ManageableResourcesInternalStateMap", tpe))
		}
		stateParametersType := reflect.ValueOf(stateParameters).Type()
		stateParametersValue := reflect.New(stateParametersType)
		stateParameters = stateParametersValue.Interface().(StateParameters)
		if err := resourceFullStateSchema.FullState.StateParameters.Decode(stateParameters); err != nil {
			return err
		}

		// InternalState
		internalState, ok := ManageableResourcesInternalStateMap[tpe]
		if !ok {
			panic(fmt.Errorf("Type %s missing from ManageableResourcesInternalStateMap", tpe))
		}
		internalStateType := reflect.ValueOf(internalState).Type()
		internalStateValue := reflect.New(internalStateType)
		internalState = internalStateValue.Interface().(InternalState)
		if err := resourceFullStateSchema.FullState.InternalState.Decode(internalState); err != nil {
			return err
		}

		fullState = &FullState{
			StateParameters: stateParameters,
			InternalState:   internalState,
		}
	}

	// ResourceFullState
	*rfs = ResourceFullState{
		ResourceKey: resourceKey,
		FullState:   fullState,
	}

	return nil
}

func (rfs *ResourceFullState) MustResource() Resource {
	return MustNewResource(rfs.ResourceKey.MustTypeName(), &rfs.FullState.StateParameters)
}

// HostState holds the state
type HostState struct {
	// Version of the binary used to put the host in this state.
	Version            version.Version     `yaml:"version"`
	ResourceFullStates []ResourceFullState `yaml:"states"`
}

func (hs HostState) Merge(hostState HostState) HostState {
	resourceFullStates := append([]ResourceFullState{}, hs.ResourceFullStates...)

	for _, resourceFullState := range hostState.ResourceFullStates {
		for _, rfs := range hs.ResourceFullStates {
			if rfs.ResourceKey == resourceFullState.ResourceKey {
				continue
			}
			resourceFullStates = append(resourceFullStates, resourceFullState)
		}
	}

	return HostState{
		Version:            hs.Version,
		ResourceFullStates: resourceFullStates,
	}
}

func (hs HostState) String() string {
	bytes, err := yaml.Marshal(&hs)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (hs HostState) ResourceKeys() []ResourceKey {
	resourceKeys := []ResourceKey{}

	for _, resourceFullState := range hs.ResourceFullStates {
		resourceKeys = append(resourceKeys, resourceFullState.ResourceKey)
	}

	return resourceKeys
}

// Validate whether current host state matches HostState.
func (hs HostState) Validate(
	ctx context.Context,
	hst host.Host,
) error {
	logger := log.GetLogger(ctx)
	logger.Info("🔎 Validating host state")
	nestedCtx := log.IndentLogger(ctx)

	fail := false

	for _, resourceFullState := range hs.ResourceFullStates {
		diffHasChanges, _, _, err := ManageableResourceStateHasPendingChanges(
			nestedCtx,
			resourceFullState.MustResource().ManageableResource,
			hst,
			resourceFullState.MustResource().MustName(),
			resourceFullState.MustResource().StateParameters,
		)
		if err != nil {
			return err
		}
		if diffHasChanges {
			fail = true
		}
	}

	if fail {
		return errors.New("state is dirty: this means external changes happened to the host that should be addressed before proceeding. Check refresh / restore commands and / or fix the changes manually")
	}

	return nil
}

// Refresh updates HostState.Resources to only contain resources for which
// its check passes.
func (hs HostState) Refresh(ctx context.Context, hst host.Host) (HostState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🔁 Refreshing state")
	nestedCtx := log.IndentLogger(ctx)

	newHostState := HostState{
		Version:            version.GetVersion(),
		ResourceFullStates: []ResourceFullState{},
	}

	for _, resourceFullState := range hs.ResourceFullStates {
		diffHasChanges, _, _, err := ManageableResourceStateHasPendingChanges(
			nestedCtx,
			resourceFullState.MustResource().ManageableResource,
			hst,
			resourceFullState.MustResource().MustName(),
			resourceFullState.MustResource().StateParameters,
		)
		if err != nil {
			return HostState{}, err
		}
		if diffHasChanges {
			continue
		}

		newHostState.ResourceFullStates = append(newHostState.ResourceFullStates, resourceFullState)
	}

	return newHostState, nil
}

func (hs HostState) GetFullState(resourceKey ResourceKey) *FullState {
	for _, resourceFullState := range hs.ResourceFullStates {
		if resourceFullState.ResourceKey == resourceKey {
			return resourceFullState.FullState
		}
	}
	return nil
}

// StepAction defines an interface for an action that can be executed from a Step.
type StepAction interface {
	// Execute actions for all bundled resources.
	Execute(ctx context.Context, hst host.Host) error
	String() string
	// Actionable returns whether any action is different from ActionOk or ActionSkip
	Actionable() bool
	// ActionResources returns a map from Action to a slice of Resource.
	ActionResources() map[Action][]Resource
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
	stateParameters := sai.Resource.StateParameters
	switch sai.Action {
	case ActionRefresh:
		refreshableManageableResource, ok := individuallyManageableResource.(RefreshableManageableResource)
		if ok {
			err := refreshableManageableResource.Refresh(ctx, hst, name)
			if err != nil {
				logger.Errorf("💥%s", sai.Resource)
			}
			return err
		}
	case ActionApply:
		if err := individuallyManageableResource.Apply(
			ctx, hst, name, *stateParameters,
		); err != nil {
			logger.Errorf("💥%s", sai.Resource)
			return err
		}
		diffHasChanges, _, _, err := ManageableResourceStateHasPendingChanges(
			ctx, individuallyManageableResource, hst, name, stateParameters,
		)
		if err != nil {
			logger.Errorf("💥%s", sai.Resource)
			return err
		}
		if diffHasChanges {
			logger.Errorf("💥%s", sai.Resource)
			return errors.New(
				"likely bug in resource implementationm as state was dirty immediately after applying",
			)
		}
	case ActionDestroy:
		err := individuallyManageableResource.Destroy(ctx, hst, name)
		if err != nil {
			logger.Errorf("💥%s", sai.Resource)
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

func (sai StepActionIndividual) ActionResources() map[Action][]Resource {
	return map[Action][]Resource{sai.Action: []Resource{sai.Resource}}
}

// StepActionMerged is a StepAction which contains multiple merged Resource.
type StepActionMerged map[Action][]Resource

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

	checkResources := []Resource{}
	configureActionParameters := map[Action]Parameters{}
	refreshNames := []Name{}
	for action, resources := range sam {
		for _, resource := range resources {
			if action == ActionRefresh {
				refreshNames = append(refreshNames, resource.MustName())
			} else {
				if configureActionParameters[action] == nil {
					configureActionParameters[action] = Parameters{}
				}
				if resource.StateParameters != nil {
					configureActionParameters[action][resource.MustName()] = *resource.StateParameters
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
		diffHasChanges, _, _, err := ManageableResourceStateHasPendingChanges(
			ctx,
			sam.MustMergeableManageableResources(),
			hst,
			resource.MustName(),
			resource.StateParameters,
		)
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

func (sam StepActionMerged) ActionResources() map[Action][]Resource {
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

func (s Step) ActionResources() map[Action][]Resource {
	return s.StepAction.ActionResources()
}

// Plan is a directed graph which contains the plan for applying resources to a host.
type Plan []*Step

// Graphviz returns a DOT directed graph containing the apply plan.
func (p Plan) Graphviz() string {
	var buff bytes.Buffer
	fmt.Fprint(&buff, "digraph resonance {\n")
	for _, step := range p {
		fmt.Fprintf(&buff, "  node [shape=box] \"%s\"\n", step)
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
		for _, step := range p {
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

func (p Plan) validate(ctx context.Context, hst host.Host) (HostState, error) {
	logger := log.GetLogger(ctx)
	logger.Info("🕵️ Validating")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	nestedNestedCtx := log.IndentLogger(nestedCtx)

	hostState := HostState{
		Version:            version.GetVersion(),
		ResourceFullStates: []ResourceFullState{},
	}
	for _, step := range p {
		for action, resources := range step.ActionResources() {
			for _, resource := range resources {
				if !action.SaveState() {
					continue
				}
				nestedLogger.Infof("%s", resource)
				diffHasChanges, _, fullState, err := ManageableResourceStateHasPendingChanges(
					nestedNestedCtx,
					resource.ManageableResource,
					hst,
					resource.MustName(),
					resource.StateParameters,
				)
				if err != nil {
					return HostState{}, err
				}
				if diffHasChanges {
					return hostState, errors.New("host state was dirty at the end, likely meaning there's a bug in the implementationm of one of the resources")
				}
				hostState.ResourceFullStates = append(
					hostState.ResourceFullStates,
					ResourceFullState{
						ResourceKey: resource.ResourceKey(),
						FullState:   fullState,
					},
				)
			}
		}
	}
	return hostState, nil
}

// Execute every Step from the Plan.
func (p Plan) Execute(ctx context.Context, hst host.Host) (HostState, error) {
	logger := log.GetLogger(ctx)
	nestedCtx := log.IndentLogger(ctx)
	logger.Info("⚙️  Executing plan")

	if err := p.executeSteps(nestedCtx, hst); err != nil {
		return HostState{}, err
	}

	hostState, err := p.validate(nestedCtx, hst)
	if err != nil {
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
	}

	for _, step := range p {
		if step.Actionable() {
			nestedLogger.Infof("%s", step)
		} else {
			nestedLogger.Debugf("%s", step)
		}
	}
}

func (p Plan) HasResourceKey(resourceKey ResourceKey) bool {
	for _, step := range p {
		for _, resources := range step.StepAction.ActionResources() {
			for _, resource := range resources {
				if resource.ResourceKey() == resourceKey {
					return true
				}
			}
		}
	}
	return false
}

func newPartialPlanFromBundles(
	ctx context.Context,
	bundles Bundles,
	cleanStateMap map[ResourceKey]bool,
	intendedAction Action,
) Plan {
	logger := log.GetLogger(ctx)

	logger.Info("👷 Building plan")
	plan := Plan{}

	var lastBundleLastStep *Step
	for _, bundle := range bundles {
		bundleSteps := []*Step{}
		refresh := false
		var step *Step
		for i, resource := range bundle {
			step = &Step{}
			plan = append(plan, step)

			// Dependant on previous bundle
			if i == 0 && lastBundleLastStep != nil {
				lastBundleLastStep.prerequisiteFor = append(lastBundleLastStep.prerequisiteFor, step)
			}

			// Action
			action := intendedAction
			if resource.StateParameters == nil {
				action = ActionDestroy
			}
			if action != ActionDestroy {
				cleanState, ok := cleanStateMap[resource.ResourceKey()]
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

	return plan
}

func prependDestroyStepsToPlan(
	ctx context.Context,
	plan Plan,
	bundles Bundles,
	savedHostState HostState,
) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, resourceFullState := range savedHostState.ResourceFullStates {
		if bundles.HasResourceKey(resourceFullState.ResourceKey) {
			continue
		}
		prerequisiteFor := []*Step{}
		if len(plan) > 0 {
			prerequisiteFor = append(prerequisiteFor, plan[0])
		}
		step := &Step{
			StepAction: StepActionIndividual{
				Resource: Resource{
					TypeName:           resourceFullState.MustResource().TypeName,
					ManageableResource: resourceFullState.MustResource().ManageableResource,
					StateParameters:    resourceFullState.MustResource().StateParameters,
				},
				Action: ActionDestroy,
			},
			prerequisiteFor: prerequisiteFor,
		}
		nestedLogger.Infof("%s", step)
		plan = append(Plan{step}, plan...)
	}
	return plan
}

func mergePlanSteps(ctx context.Context, plan Plan) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("📦 Merging resources")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)

	newPlan := Plan{}

	mergedSteps := map[Type]*Step{}
	for _, step := range plan {
		stepActionIndividual := step.StepAction.(StepActionIndividual)
		if !stepActionIndividual.Resource.IsMergeableManageableResources() {
			newPlan = append(newPlan, step)
			continue
		}

		stepType := stepActionIndividual.Resource.MustType()
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
			stepActionMerged[stepActionIndividual.Action], stepActionIndividual.Resource,
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

func topologicalSortPlan(ctx context.Context, plan Plan) (Plan, error) {
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

func NewPlanFromSavedStateAndBundles(
	ctx context.Context, hst host.Host, bundles Bundles, savedHostState *HostState,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning changes")
	nestedCtx := log.IndentLogger(ctx)

	// State
	cleanStateMap, err := bundles.GetCleanStateMap(nestedCtx, hst)
	if err != nil {
		return Plan{}, err
	}

	// Add Bundles
	plan := newPartialPlanFromBundles(nestedCtx, bundles, cleanStateMap, ActionNone)

	// Prepend destroy steps
	if savedHostState != nil {
		plan = prependDestroyStepsToPlan(nestedCtx, plan, bundles, *savedHostState)
	}

	// Merge steps
	plan = mergePlanSteps(nestedCtx, plan)

	// Sort
	plan, err = topologicalSortPlan(nestedCtx, plan)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func prependDestroyStepsFromPlanBundlesToPlan(
	ctx context.Context,
	plan Plan,
	bundles, planBundles Bundles,
) Plan {
	logger := log.GetLogger(ctx)
	logger.Info("💀 Determining resources to destroy")
	nestedCtx := log.IndentLogger(ctx)
	nestedLogger := log.GetLogger(nestedCtx)
	for _, planBundle := range planBundles {
		for _, resource := range planBundle {
			if bundles.HasResourceKey(resource.ResourceKey()) {
				continue
			}
			prerequisiteFor := []*Step{}
			if len(plan) > 0 {
				prerequisiteFor = append(prerequisiteFor, plan[0])
			}
			step := &Step{
				StepAction: StepActionIndividual{
					Resource: Resource{
						TypeName:           resource.TypeName,
						ManageableResource: resource.ManageableResource,
						StateParameters:    resource.StateParameters,
					},
					Action: ActionDestroy,
				},
				prerequisiteFor: prerequisiteFor,
			}
			nestedLogger.Infof("%s", step)
			plan = append(Plan{step}, plan...)
		}
	}
	return plan
}

func NewRollbackPlan(
	ctx context.Context,
	hst host.Host,
	bundles Bundles,
	savedHostState *HostState,
	initialHostState HostState,
) (Plan, error) {
	logger := log.GetLogger(ctx)
	logger.Info("📝 Planning rollback")
	nestedCtx := log.IndentLogger(ctx)

	rollbackBundles := NewBundlesFromHostState(&initialHostState)

	// State
	cleanStateMap, err := rollbackBundles.GetCleanStateMap(nestedCtx, hst)
	if err != nil {
		return Plan{}, err
	}

	// Add Bundles
	plan := newPartialPlanFromBundles(nestedCtx, rollbackBundles, cleanStateMap, ActionNone)

	// Prepend destroy steps
	plan = prependDestroyStepsFromPlanBundlesToPlan(nestedCtx, plan, rollbackBundles, bundles)

	// Merge steps
	plan = mergePlanSteps(nestedCtx, plan)

	// Sort
	plan, err = topologicalSortPlan(nestedCtx, plan)
	if err != nil {
		return nil, err
	}

	return plan, nil
}
