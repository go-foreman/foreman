package scheme

// Object interface must be supported by all message types registered with Scheme. Since objects in a scheme are
// expected to be marshalled to the wire, the interface an Object must provide to the Scheme allows
// marshallers to set the kind, and group the object is represented as
type Object interface {
	GroupKind() GroupKind
	SetGroupKind(gk *GroupKind)
}

type TypeMeta struct {
	Kind  string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	Group string `json:"group,omitempty" protobuf:"bytes,2,opt,name=group"`
}

func (t TypeMeta) GroupKind() GroupKind {
	return GroupKind{Group: Group(t.Group), Kind: t.Kind}
}

func (t *TypeMeta) SetGroupKind(gk *GroupKind) {
	t.Group = gk.Group.String()
	t.Kind = gk.Kind
}
