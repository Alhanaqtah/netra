package redis

import (
	"context"
	_ "embed"
	"netra/backends"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed unlock.lua
var unlockLua string

type Backend struct {
	client *redis.Client
}

func New(ctx context.Context, client *redis.Client) (*Backend, error) {
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
		return false, backends.ErrLockExists
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
	return nil
}
