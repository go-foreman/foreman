package message

import (
	"testing"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/assert"
)

func TestUnstructured(t *testing.T) {
	unstr := &Unstructured{}

	gk := &scheme.GroupKind{
		Group: "testGroup",
		Kind:  "testKind",
	}

	unstr.SetGroupKind(gk)

	assert.Equal(t, gk.Kind, unstr.GroupKind().Kind)
	assert.Equal(t, gk.Group, unstr.GroupKind().Group)

	assert.Empty(t, unstr.GetUID())

	unstr.Object["uid"] = "123"
	assert.Equal(t, "123", unstr.GetUID())
}
