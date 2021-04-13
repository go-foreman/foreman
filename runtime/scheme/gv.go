package scheme

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

func (gk GroupKind) Empty() bool {
	return gk.Group.Empty() && len(gk.Kind) == 0
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