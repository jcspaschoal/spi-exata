package actions

import "fmt"

// The set of actions that can be used.
var (
	Create = newAction("CREATE")
	Delete = newAction("DELETE")
	Update = newAction("UPDATE")
	Get    = newAction("GET")
)

// =============================================================================

// Set of known actions.
var actions = make(map[string]Action)

// Action represents an action in the system.
type Action struct {
	value string
}

func newAction(action string) Action {
	a := Action{action}
	actions[action] = a
	return a
}

// String returns the name of the action.
func (a Action) String() string {
	return a.value
}

// Equal provides support for the go-cmp package and testing.
func (a Action) Equal(a2 Action) bool {
	return a.value == a2.value
}

// MarshalText provides support for logging and any marshal needs.
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.value), nil
}

// =============================================================================

// Parse parses the string value and returns an action if one exists.
func Parse(value string) (Action, error) {
	action, exists := actions[value]
	if !exists {
		return Action{}, fmt.Errorf("invalid action %q", value)
	}

	return action, nil
}

// MustParse parses the string value and returns an action if one exists. If
// an error occurs the function panics.
func MustParse(value string) Action {
	action, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return action
}
