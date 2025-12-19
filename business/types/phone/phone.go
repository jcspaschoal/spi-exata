// Package phone represents a phone number in the system.
package phone

import (
	"database/sql"
	"fmt"
	"regexp"
)

// Phone represents a phone number in the system.
type Phone struct {
	value string
}

// String returns the value of the phone number.
func (p Phone) String() string {
	return p.value
}

// Equal provides support for the go-cmp package and testing.
func (p Phone) Equal(p2 Phone) bool {
	return p.value == p2.value
}

// MarshalText provides support for logging and any marshal needs.
func (p Phone) MarshalText() ([]byte, error) {
	return []byte(p.value), nil
}

// =============================================================================

// phoneRegEx allows for an optional +, followed by digits, spaces, or hyphens.
var phoneRegEx = regexp.MustCompile(`^\+?[0-9\s-]{3,20}$`)

// Parse parses the string value and returns a phone number if the value complies
// with the rules for a phone number.
func Parse(value string) (Phone, error) {
	if !phoneRegEx.MatchString(value) {
		return Phone{}, fmt.Errorf("invalid phone %q", value)
	}

	return Phone{value}, nil
}

// MustParse parses the string value and returns a phone number if the value
// complies with the rules for a phone number. If an error occurs the function panics.
func MustParse(value string) Phone {
	phone, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return phone
}

// =============================================================================

// Null represents a phone number in the system that can be empty.
type Null struct {
	value string
	valid bool
}

// ToSQLNullString converts a Null value to a sql NullString.
func ToSQLNullString(n Null) sql.NullString {
	return sql.NullString{
		String: n.value,
		Valid:  n.valid,
	}
}

// String returns the value of the phone number.
func (n Null) String() string {
	if !n.valid {
		return "NULL"
	}

	return n.value
}

// Equal provides support for the go-cmp package and testing.
func (n Null) Equal(n2 Null) bool {
	return n.value == n2.value && n.valid == n2.valid
}

// MarshalText provides support for logging and any marshal needs.
func (n Null) MarshalText() ([]byte, error) {
	return []byte(n.value), nil
}

// =============================================================================

// ParseNull parses the string value and returns a phone number if the value complies
// with the rules for a phone number.
func ParseNull(value string) (Null, error) {
	if value == "" {
		return Null{}, nil
	}

	if !phoneRegEx.MatchString(value) {
		return Null{}, fmt.Errorf("invalid phone %q", value)
	}

	return Null{value, true}, nil
}

// MustParseNull parses the string value and returns a phone number if the value
// complies with the rules for a phone number. If an error occurs the function panics.
func MustParseNull(value string) Null {
	phone, err := ParseNull(value)
	if err != nil {
		panic(err)
	}

	return phone
}
