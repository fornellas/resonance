package state

import "fmt"

// State holds information about a resource.
// It must be marshallable by gopkg.in/yaml.v3.
// It must work with reflect.DeepEqual.
type State interface{}

func Load(name string) (*State, error) {
	return nil, fmt.Errorf("TODO")
}

func Save(State) error {
	return fmt.Errorf("TODO")
}
