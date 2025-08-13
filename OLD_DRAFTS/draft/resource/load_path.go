package resource

import "context"

type ResourceState struct {
	FileState       FileState       `json:"File"`
	APTPackageState APTPackageState `json:"APTPackage"`
}

type ResourcesFile []ResourceState

func LoadPath(ctx context.Context, path string) (States, error) {
	return []State{}, nil
}
