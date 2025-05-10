// Redis backend.
package redis

import (
	"context"
	_ "embed"
	"errors"
	"time"

	netra "github.com/Alhanaqtah/netra/v0.0.0"
	"github.com/Alhanaqtah/netra/v0.0.0/backends"

	"github.com/redis/go-redis/v9"
)

var _ netra.Backend = &Backend{}

//go:embed lock.lua
var lockLua string

//go:embed unlock.lua
var unlockLua string

//go:embed heartbeat.lua
var heartbeatLua string

// ErrClientNotProvided returns if redis client was not provided.
var ErrClientNotProvided = errors.New("client not provided")

// Backend is redis client wrapper.
type Backend struct {
	client *redis.Client
}

// New creates new backend instance.
func New(client *redis.Client) (*Backend, error) {
	if client == nil {
		return nil, ErrClientNotProvided
	}

	return &Backend{
		client: client,
	}, nil
}

// TryLock attempts to acquire a lock.
func (b *Backend) TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	res, err := b.client.Eval(ctx, lockLua, []string{lockName}, nodeID, ttl.Milliseconds()).Result()
	if err != nil {
		return err
	}

	code, _ := res.(int64)

	switch code {
	case -1:
		return backends.ErrLockHeldByAnotherNode
	case 1:
		return backends.ErrAlreadyHoldingLock
	}

	return nil
}

// TryUnlock attempts to unlock the lock.
func (b *Backend) TryUnlock(ctx context.Context, lockName, nodeID string) error {
	if _, err := b.client.Eval(ctx, unlockLua, []string{lockName}, nodeID).Result(); err != nil {
		return err
	}

	return nil
}

// HeartBeat attempts to extend the lifetime of the lock.
func (b *Backend) HeartBeat(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	res, err := b.client.Eval(ctx, heartbeatLua, []string{lockName}, ttl.Milliseconds(), nodeID).Result()
	if err != nil {
		return err
	}

	code, _ := res.(int64)

	switch code {
	case -1:
		return backends.ErrLockDoesNotExist
	case 1:
		return backends.ErrLockHeldByAnotherNode
	}

	return nil
}
