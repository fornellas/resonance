package resources

import "fmt"

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
	// ActionConfigure means that the state of the resource is not as expected and it that it must be configured.
	ActionConfigure
	// ActionDestroy means that the resource is no longer needed and is to be destroyed.
	ActionDestroy
	// ActionCount has the number of existing actions
	ActionCount
)

var actionEmojiMap = map[Action]string{
	ActionOk:        "✅",
	ActionSkip:      "💨",
	ActionRefresh:   "🔄",
	ActionConfigure: "🔧",
	ActionDestroy:   "💀",
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
	ActionOk:        "OK",
	ActionSkip:      "Skip",
	ActionRefresh:   "Refresh",
	ActionConfigure: "Configure",
	ActionDestroy:   "Destroy",
}

func (a Action) Actionable() bool {
	if a == ActionRefresh {
		return true
	}
	if a == ActionConfigure {
		return true
	}
	if a == ActionDestroy {
		return true
	}
	return false
}

func (a Action) String() string {
	str, ok := actionStrMap[a]
	if !ok {
		panic(fmt.Errorf("invalid action %d", a))
	}
	return str
}
