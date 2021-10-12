package scheme

import (
	"strings"

	"github.com/pkg/errors"
)

type Group string

// Empty returns true if group is empty
func (gv Group) Empty() bool {
	return len(gv) == 0
}

// String puts "group" into a string, it's possible in the future for this type to change
func (gv Group) String() string {
	return string(gv)
}

// GroupKind specifies a Group and a Kind
type GroupKind struct {
	Group Group
	Kind  string
}

// Empty says whether GroupKind is empty or not valid. An empty group is allowed whole kind isn't.
func (gk GroupKind) Empty() bool {
	return len(gk.Kind) == 0 || (gk.Group.Empty() && len(gk.Kind) == 0)
}

func (gk GroupKind) String() string {
	if len(gk.Group) == 0 {
		return gk.Kind
	}
	return gk.Group.String() + "." + gk.Kind
}

// Identifier used as uniq key in schema
func (gk GroupKind) Identifier() string {
	return gk.String()
}

func FromString(str string) (GroupKind, error) {
	items := strings.Split(str, ".")

	if len(items) != 2 {
		return GroupKind{}, errors.Errorf("error creating GroupKind from '%s'", str)
	}

	return GroupKind{Group: Group(items[0]), Kind: items[1]}, nil
}
