package draft

type Resource any

type File struct{}

type APTPackage struct{}

type State struct {
	Files       []File
	APTPackages []APTPackage
}

func (s *State) HasResource(Resource) bool { return false }

func (s *State) Resources() []Resource { return []Resource{} }

type Store interface {
	Load() *State
}

func Apply(targetState *State, store Store) {
	committedState := store.Load()

	missingCommitResources := []Resource{}

	for _, resource := range targetState.Resources() {
		if !committedState.HasResource(resource) {
			missingCommitResources = append(missingCommitResources, resource)
		}
	}

}
