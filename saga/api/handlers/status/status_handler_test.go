package status

import (
	"github.com/go-foreman/foreman/log"
	sagaMock "github.com/go-foreman/foreman/testing/mocks/saga"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io"
	"net/url"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeMock := sagaMock.NewMockStore(ctrl)

	statusService := NewStatusService(storeMock)

	logger := log.DefaultLogger(io.Discard)

	statusHandler := NewStatusHandler(logger, statusService)

	t.Run("get correct int", func(t *testing.T) {
		values := url.Values{
			"offset": []string{"2"},
		}

		intValue, err := statusHandler.getInt(values, "offset")
		assert.NoError(t, err)
		assert.NotNil(t, intValue)
		assert.Equal(t, *intValue, 2)
	})

	t.Run("get wrong int", func(t *testing.T) {
		values := url.Values{
			"offset": []string{"2.2"},
		}

		_, err := statusHandler.getInt(values, "offset")
		assert.Error(t, err, "is expected to be an integer")
	})

	t.Run("get non-existing value", func(t *testing.T) {
		values := url.Values{}

		intValue, err := statusHandler.getInt(values, "offset")
		assert.NoError(t, err)
		assert.Nil(t, intValue)
	})
}
