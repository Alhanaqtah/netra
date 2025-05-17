## About
Netra is a simple library for leader election based on various backends to choose from, providing both low-level and high-level API for working with locks.

## Install
```
go get -u github.com/Alhanaqtah/netra
```

## API
- **TryLock**: attempts to acquire a lock.
- **HeartBeat**: attempts to extend the lifetime of the lock.
- **TryUnlock**: attempts to unlock the lock.
- **Run**: attempts to establish a lock with the frequency of the specified intervaland extends it with the frequency of the specified interval.
- **IsLeader**: returns whether the node is a leader.
- **GetNodeID**: returns node's id.

## Hooks
- **OnLocked**: executes asynchronously when a lock is set.
- **OnUnlocked**: executes asynchronously when a lock is intentionally removed.
- **OnLockLost**: executes asynchronously when the lock is lost (for example, due to network delays).

## Supported backends
- [x] Redis
- [ ] etcd

## Example
```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/Alhanaqtah/netra"
	"github.com/Alhanaqtah/netra/backends/v0.0.0/redis"
	redis_client "github.com/redis/go-redis/v9"
)

func main() {
	client := redis_client.NewClient(&redis_client.Options{
		Addr: "localhost:6379",
	})

	backend, err := redis.New(client)
	if err != nil {
		log.Println("err: ", err)
		return
	}

	n, err := netra.New(&netra.Config{
		Backend: backend,
	})
	if err != nil {
		log.Println("err: ", err)
		return
	}

	_ = n.TryLock(context.Background())

	_ = n.HeartBeat(context.Background())

	_ = n.TryUnlock(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := n.Run(ctx); err != nil {
		log.Println("err: ", err)
		return
	}
}
```
