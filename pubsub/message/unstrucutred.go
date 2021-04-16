package message

import (
	"encoding/json"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/pkg/errors"
)

type Unstructured struct {
	// Object is a JSON compatible map with string, float, int, bool, []interface{}, or
	// map[string]interface{}
	// children.
	Object map[string]interface{}
}

func (u *Unstructured) UnmarshalJSON(b []byte) error {
	u.Object = make(map[string]interface{})
	if err := json.Unmarshal(b, &u.Object); err != nil {
		return errors.Wrap(err, "unmarshalling into Unstructured")
	}

	u.walkUnstructured()

	return nil
}

func (u *Unstructured) walkUnstructured() {
	for key, val := range u.Object {
		if nestedMap, ok := val.(map[string]interface{}); ok {
			//it might be nested object. Let's see if has follows schema.Object interface
			nestedUnstructured := &Unstructured{Object: nestedMap}

			if gk := nestedUnstructured.GroupKind(); !gk.Empty() {
				u.Object[key] = nestedUnstructured
				nestedUnstructured.walkUnstructured()
			}
		}
	}
}

func (u *Unstructured) GroupKind() scheme.GroupKind {
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

func (u *Unstructured) SetGroupKind(gk *scheme.GroupKind) {
	if u.Object == nil {
		u.Object = make(map[string]interface{})
	}
	u.Object["group"] = gk.Group.String()
	u.Object["kind"] = gk.Kind
}

func (u *Unstructured) GetUID() string {
	uidVal, ok := u.Object["uid"]
	if ok {
		return uidVal.(string)
	}
	return ""
}
