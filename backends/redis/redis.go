package redis

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/Alhanaqtah/netra"
	"github.com/Alhanaqtah/netra/backends"

	"github.com/redis/go-redis/v9"
)

var _ netra.Backend = &Backend{}

//go:embed lock.lua
var lockLua string

//go:embed unlock.lua
var unlockLua string

//go:embed heartbeat.lua
var heartbeatLua string

var ErrClientNotProvided = errors.New("client not provided")

type Backend struct {
	client *redis.Client
}

func New(client *redis.Client) (*Backend, error) {
	if client == nil {
		return nil, ErrClientNotProvided
	}

	return &Backend{
		client: client,
	}, nil
}

func (b *Backend) TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) (bool, error) {
	res, err := b.client.Eval(ctx, lockLua, []string{lockName}, nodeID, ttl.Milliseconds()).Result()
	if err != nil {
		return false, err
	}

	code := res.(int64)

	switch code {
	case -1:
		return false, backends.ErrLockHeldByAnotherNode
	case 1:
		return false, backends.ErrAlreadyHoldingLock
	}

	return true, nil
}

func (b *Backend) TryUnlock(ctx context.Context, lockName, nodeID string) (bool, error) {
	if _, err := b.client.Eval(ctx, unlockLua, []string{lockName}, nodeID).Result(); err != nil {
		return false, err
	}

	return true, nil
}

func (b *Backend) HeartBeat(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	res, err := b.client.Eval(ctx, heartbeatLua, []string{lockName}, ttl.Milliseconds(), nodeID).Result()
	if err != nil {
		return err
	}

	code := res.(int64)

	switch code {
	case -1:
		return backends.ErrLockDoesNotExist
	case 1:
		return backends.ErrLockHeldByAnotherNode
	}

	return nil
}
