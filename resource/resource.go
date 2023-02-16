package resource

import (
	"reflect"

	"github.com/fornellas/resonance/inventory"
	"github.com/fornellas/resonance/state"
)

// Resource manages state.
type Resource interface {
	// A unique name that identifies the resource (eg: its primary key).
	Name() string
	// Reads current resource state
	ReadState() (state.State, error)
	Run(inventory inventory.Inventory, parameters interface{}) (state.State, error)
}

func ApplyResource(resource Resource, inventory inventory.Inventory, parameters interface{}) error {
	savedState, err := state.Load(resource.Name())
	if err != nil {
		return err
	}

	if savedState != nil {
		preState, err := resource.ReadState()
		if err != nil {
			return err
		}
		if reflect.DeepEqual(*savedState, preState) {
			return nil
		}
	}

	appyState, err := resource.Run(inventory, parameters)
	if err != nil {
		return err
	}
	if err := state.Save(appyState); err != nil {
		return err
	}

	return nil
}
