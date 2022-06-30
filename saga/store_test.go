package saga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpts(t *testing.T) {
	storeOpts := &filterOptions{}
	opts := []FilterOption{WithStatus("xxx"), WithSagaId("yyy"), WithSagaName("zzz")}
	for _, o := range opts {
		o(storeOpts)
	}

	assert.Equal(t, storeOpts.status, "xxx")
	assert.Equal(t, storeOpts.sagaId, "yyy")
	assert.Equal(t, storeOpts.sagaName, "zzz")
}

func TestStatusFromStr(t *testing.T) {
	type test struct {
		input  string
		errStr string
		res    status
	}

	tests := []test{
		{input: "xxx", errStr: "unknown status string"},
		{input: "created", res: sagaStatusCreated},
		{input: "in_progress", res: sagaStatusInProgress},
		{input: "recovering", res: sagaStatusRecovering},
		{input: "compensating", res: sagaStatusCompensating},
		{input: "completed", res: sagaStatusCompleted},
		{input: "failed", res: sagaStatusFailed},
	}

	for _, tc := range tests {
		st, err := statusFromStr(tc.input)
		if tc.errStr != "" {
			assert.Error(t, err)
			assert.EqualError(t, err, tc.errStr)
		} else {
			assert.Equal(t, tc.res, st)
		}
	}
}
