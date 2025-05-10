package redis

import (
	"context"
	"testing"
	"time"

	backends "github.com/Alhanaqtah/netra/backends"

	"github.com/google/uuid"
	redis_client "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	redis_container "github.com/testcontainers/testcontainers-go/modules/redis"
)

const lockName = "NETRA:LOCK"

func TestRedisBackend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup test containers
	container, err := redis_container.Run(ctx, "redis:7.4.2")
	if err != nil {
		t.Errorf("failed to run redis container: %s", err)
	}

	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate redis container: %s", err)
		}
	}()

	endpoint, _ := container.Endpoint(ctx, "")

	// Init redis client
	client := redis_client.NewClient(&redis_client.Options{
		Addr: endpoint,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to ping redis: %s\n", err)
	}

	// Init redis backend
	backend, _ := New(client)

	// Tests
	tryLockTest(ctx, t, client, backend)
	tryUnlockTest(ctx, t, client, backend)
	tryHeartBeatTest(ctx, t, client, backend)
}

func tryLockTest(ctx context.Context, t *testing.T, client *redis_client.Client, backend *Backend) {
	firstNodeID := uuid.NewString()
	secondNodeID := uuid.NewString()

	err := backend.TryLock(ctx, lockName, firstNodeID, 5*time.Second)
	assert.NoError(t, err)

	err = backend.TryLock(ctx, lockName, firstNodeID, 1*time.Second)
	assert.ErrorIs(t, err, backends.ErrAlreadyHoldingLock)

	err = backend.TryLock(ctx, lockName, secondNodeID, 1*time.Second)
	assert.ErrorIs(t, err, backends.ErrLockHeldByAnotherNode)

	// Clean
	if _, err := client.Del(ctx, lockName).Result(); err != nil {
		t.Errorf("failed to release lock: %s\n", err)
	}
}

func tryUnlockTest(ctx context.Context, t *testing.T, client *redis_client.Client, backend *Backend) {
	nodeID := uuid.NewString()

	if _, err := client.Set(ctx, lockName, nodeID, 1*time.Second).Result(); err != nil {
		t.Errorf("faield to set lock: %s\n", err)
	}

	err := backend.TryUnlock(ctx, lockName, nodeID)
	assert.NoError(t, err)
}

func tryHeartBeatTest(ctx context.Context, t *testing.T, client *redis_client.Client, backend *Backend) {
	firstNodeID := uuid.NewString()
	secondNodeID := uuid.NewString()
	ttl := 100 * time.Millisecond

	// Lock doesn't exist
	err := backend.HeartBeat(ctx, lockName, firstNodeID, ttl)
	assert.ErrorIs(t, err, backends.ErrLockDoesNotExist)

	if _, err := client.Set(ctx, lockName, firstNodeID, ttl).Result(); err != nil {
		t.Errorf("failed to set lock: %s\n", err)
	}

	// Successful case
	err = backend.HeartBeat(ctx, lockName, firstNodeID, ttl)
	assert.NoError(t, err)

	// Lock held by another node
	err = backend.HeartBeat(ctx, lockName, secondNodeID, ttl)
	assert.ErrorIs(t, err, backends.ErrLockHeldByAnotherNode)

	// Check if heart beat extends the lock ttl
	err = backend.HeartBeat(ctx, lockName, firstNodeID, 1*time.Second)
	assert.NoError(t, err)

	if ttl, err := client.TTL(ctx, lockName).Result(); err != nil {
		t.Errorf("failed to get lock's ttl: %s\n", err)
	} else {
		assert.GreaterOrEqual(t, ttl, 700*time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	// Check if expiration works
	err = backend.HeartBeat(ctx, lockName, firstNodeID, 1*time.Second)
	assert.ErrorIs(t, err, backends.ErrLockDoesNotExist)

	// Clean
	if _, err := client.Del(ctx, lockName).Result(); err != nil {
		t.Errorf("failed to release lock: %s\n", err)
	}
}
