package resources

import "context"

type SingleProvisioner interface {
	Load(context.Context, State) error
	Apply(context.Context, State) error
}

type SingleProvisionerResolver interface {
	SingleProvisioner
	Resolve(context.Context, State) error
}

type GroupProvisioner interface {
	Load(context.Context, []State) error
	Apply(context.Context, []State) error
}

type GroupProvisionerResolver interface {
	GroupProvisioner
	Resolve(context.Context, []State) error
}
