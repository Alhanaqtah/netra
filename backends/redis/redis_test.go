package redis

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {

	client := redis.NewClient(&redis.Options{})

	backend, err := New(client)

	assert.ErrorIs(t, err, nil)
	assert.Equal(t, backend.client, client)

	backend, err = New(nil)

	var expectedBackend *Backend = nil
	assert.ErrorIs(t, err, ErrClientNotProvided)
	assert.Equal(t, backend, expectedBackend)

}
