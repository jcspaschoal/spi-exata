// Package resource represents the resource type in the system.
package resource

import "fmt"

// The set of resources that can be used.
var (
	Dashboard = newResource("DASHBOARD")
	Subject   = newResource("SUBJECT")
	Page      = newResource("PAGE")
	User      = newResource("User")
)

// =============================================================================

// Set of known resources.
var resources = make(map[string]Resource)

// Resource represents a resource in the system.
type Resource struct {
	value string
}

func newResource(resource string) Resource {
	r := Resource{resource}
	resources[resource] = r
	return r
}

// String returns the name of the resource.
func (r Resource) String() string {
	return r.value
}

// Equal provides support for the go-cmp package and testing.
func (r Resource) Equal(r2 Resource) bool {
	return r.value == r2.value
}

// MarshalText provides support for logging and any marshal needs.
func (r Resource) MarshalText() ([]byte, error) {
	return []byte(r.value), nil
}

// =============================================================================

// Parse parses the string value and returns a resource if one exists.
func Parse(value string) (Resource, error) {
	resource, exists := resources[value]
	if !exists {
		return Resource{}, fmt.Errorf("invalid resource %q", value)
	}

	return resource, nil
}

// MustParse parses the string value and returns a resource if one exists. If
// an error occurs the function panics.
func MustParse(value string) Resource {
	resource, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return resource
}
