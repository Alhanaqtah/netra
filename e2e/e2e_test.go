package e2e

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	netra "github.com/Alhanaqtah/netra/v0.0.0"
	redis_backend "github.com/Alhanaqtah/netra/v0.0.0/backends/redis"

	redis_client "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	lockName = "NETRA:LOCK"
	lockTTL  = 1 * time.Second
)

func TestE2E(t *testing.T) {
	ctx := context.Background()

	containerCtx, containerCacnel := context.WithTimeout(ctx, 10*time.Second)
	defer containerCacnel()

	container, err := redis.Run(containerCtx, "redis:7.4.2")
	if err != nil {
		t.Fatalf("failed to run redis container: %s\n", err)
	}

	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(ctx, 1*time.Second)
		defer terminateCancel()

		if err := container.Terminate(terminateCtx); err != nil {
			t.Logf("failed to terminate redis container: %s\n", err)
		}
	}()

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get container's endpoint: %s\n", err)
	}

	// Redis connections
	client1 := newClient(ctx, t, endpoint)
	defer client1.Close()

	client2 := newClient(ctx, t, endpoint)
	defer client2.Close()

	client3 := newClient(ctx, t, endpoint)
	defer client3.Close()

	// Backends
	backend1 := newBackend(t, client1)
	backend2 := newBackend(t, client2)
	backend3 := newBackend(t, client3)

	runTest(t, ctx, backend1, backend2, backend3)
	lowLevelAPITest(t, ctx, backend1)
}

// This test checks [netra.Run] behaviour on leader election and stopping by context cancellation
func runTest(t *testing.T, ctx context.Context, backend1, backend2, backend3 netra.Backend) {
	var counter atomic.Int32

	onLock := func() {
		counter.Add(1)

		if counter.Load() > 1 {
			t.Error("split brain happened")
		}
	}

	netra1, _ := netra.New(&netra.Config{
		LockName: lockName,
		LockTTL:  lockTTL,
		OnLocked: onLock,
		Backend:  backend1,
	})

	netra2, _ := netra.New(&netra.Config{
		LockName: lockName,
		LockTTL:  lockTTL,
		OnLocked: onLock,
		Backend:  backend2,
	})

	netra3, _ := netra.New(&netra.Config{
		LockName: lockName,
		LockTTL:  lockTTL,
		OnLocked: onLock,
		Backend:  backend3,
	})

	wg := sync.WaitGroup{}
	wg.Add(3)

	netra1Ctx, netra1Cancel := context.WithTimeout(ctx, 5*time.Second)
	defer netra1Cancel()

	netra2Ctx, netra2Cancel := context.WithTimeout(ctx, 10*time.Second)
	defer netra2Cancel()

	netra3Ctx, netra3Cancel := context.WithTimeout(ctx, 20*time.Second)
	defer netra3Cancel()

	go func() {
		defer func() {
			counter.Add(-1)
			wg.Done()
		}()

		if err := netra1.Run(netra1Ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Error("netra1.Run: ", err)
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer func() {
			counter.Add(-1)
			wg.Done()
		}()

		if err := netra2.Run(netra2Ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Error("netra2.Run: ", err)
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer func() {
			counter.Add(-1)
			wg.Done()
		}()

		if err := netra3.Run(netra3Ctx); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Error("netra3.Run: ", err)
			}
		}
	}()

	wg.Wait()

	// Waiting until lock expire
	time.Sleep(lockTTL)
}

func lowLevelAPITest(t *testing.T, ctx context.Context, backend netra.Backend) {
	n, _ := netra.New(&netra.Config{
		TryLockInterval: 1 * time.Microsecond,
		Backend:         backend,
	})

	tryLockCtx, tryLockCancel := context.WithTimeout(ctx, 1*time.Second)
	defer tryLockCancel()

	err := n.TryLock(tryLockCtx)
	assert.NoError(t, err)

	heartBeatCtx, heartBeatCancel := context.WithTimeout(ctx, 1*time.Second)
	defer heartBeatCancel()

	err = n.HeartBeat(heartBeatCtx)
	assert.NoError(t, err)

	tryUnlockCtx, tryUnlockCancel := context.WithTimeout(ctx, 1*time.Second)
	defer tryUnlockCancel()

	err = n.TryUnlock(tryUnlockCtx)
	assert.NoError(t, err)
}

func newClient(ctx context.Context, t *testing.T, endpoint string) *redis_client.Client {
	ctxWithTimeout, cancelWithTimeout := context.WithTimeout(ctx, 1*time.Second)
	defer cancelWithTimeout()

	client := redis_client.NewClient(&redis_client.Options{
		Addr: endpoint,
	})

	if _, err := client.Ping(ctxWithTimeout).Result(); err != nil {
		t.Fatalf("failed to PING redis by client1: %s\n", err)
	}

	return client
}

func newBackend(t *testing.T, redis *redis_client.Client) netra.Backend {
	backend, err := redis_backend.New(redis)
	require.NotNil(t, backend)
	require.NoError(t, err)

	return backend
}
