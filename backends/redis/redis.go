package redis

import (
	"context"
	_ "embed"
	"errors"
	"netra/backends"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed unlock.lua
var unlockLua string

//go:embed unlock.lua
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
	ok, err := b.client.SetNX(ctx, lockName, nodeID, ttl).Result()
	if err != nil {
		return false, err
	}

	if !ok {
		return false, backends.ErrLockHeldByAnotherNode
	}

	return true, nil
}

func (b *Backend) TryUnlock(ctx context.Context, lockName, nodeID string) (bool, error) {
	if _, err := b.client.Eval(ctx, unlockLua, []string{lockName}, nodeID).Result(); err != nil {
		return false, err
	}

	return true, nil
}

func (b *Backend) HeartBeat(ctx context.Context, lockName, nodeID string) error {
	res, err := b.client.Eval(ctx, heartbeatLua, []string{lockName}, nodeID).Result()
	if err != nil {
		return err
	}

	code := res.(int64)

	if code == -1 {
		return backends.ErrLockDoesNotExist
	}
	if code == 1 {
		return backends.ErrAlreadyHoldingLock
	}

	return nil
}
