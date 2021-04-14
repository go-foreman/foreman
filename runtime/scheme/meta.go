package scheme

// Object interface must be supported by all API types registered with Scheme. Since objects in a scheme are
// expected to be serialized to the wire, the interface an Object must provide to the Scheme allows
// serializers to set the kind, version, and group the object is represented as
type Object interface {
	GroupKind() GroupKind
}

type TypeMeta struct {
	Kind string `json:"kind,omitempty"`
	Group string `json:"group,omitempty"`
}

func (obj TypeMeta) GroupKind() GroupKind {
	return GroupKind{Group: Group(obj.Group), Kind: obj.Kind}
}
