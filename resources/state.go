package resources

type State interface {
	Validate() error
}

type StateSatisfier[S State] interface {
	State
	Satisfies(StateSatisfier[S]) bool
}
