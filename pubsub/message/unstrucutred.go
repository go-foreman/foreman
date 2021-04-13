package message

import (
	"encoding/json"
	"github.com/go-foreman/foreman/runtime/scheme"
)

type Unstructured struct {
	// Object is a JSON compatible map with string, float, int, bool, []interface{}, or
	// map[string]interface{}
	// children.
	Object map[string]interface{}
}

func (u *Unstructured) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &u.Object)
}

func (u Unstructured) GroupKind() scheme.GroupKind {
	gk := scheme.GroupKind{}
	groupVal, ok := u.Object["group"]
	if ok {
		gk.Group = scheme.Group(groupVal.(string))
	}

	kindVal, ok := u.Object["kind"]
	if ok {
		gk.Kind = kindVal.(string)
	}
	return gk
}

func (u *Unstructured) GetUID() string {
	uidVal, ok := u.Object["uid"]
	if ok {
		return uidVal.(string)
	}
	return ""
}
