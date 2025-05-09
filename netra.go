// Netra is a simple library for creating distributed locks
// based on various backends to choose from, providing both
// low-level and high-level API for working with locks.
package netra

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/Alhanaqtah/netra/backends"

	"github.com/google/uuid"
)

const (
	defaultLockName          = "NETRA:LOCK:729522b5-a3b3-47d1-875e-c65abbc04a3e"
	defaultLockTTL           = 1 * time.Second
	defaultTryLockInterval   = 1 * time.Second
	defaultHeartBeatInterval = 500 * time.Millisecond
)

// ErrBackendNotProvided is returned when trying to create a nertra client without providing a backend.
var ErrBackendNotProvided = errors.New("backend not provided")

// Backend describes the methods that the storage backend must implement.
type Backend interface {
	TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) error
	TryUnlock(ctx context.Context, lockName, nodeID string) error
	HeartBeat(ctx context.Context, lockName, nodeID string, ttl time.Duration) error
}

type Netra struct {
	lockName         string
	nodeID           string
	lockTTL          time.Duration
	tryLockInterval  time.Duration
	hearBeatInterval time.Duration
	onLocked         func()
	onLockLost       func()
	onUnlocked       func()
	isLeader         atomic.Bool
	backend          Backend
}

type Config struct {
	// LockName is used to specify the name of the lock in the database
	// in which the nodeID will be stored. By default.
	// By default is "NETRA:LOCK:729522b5-a3b3-47d1-875e-c65abbc04a3e".
	LockName string
	// NodeID is used to retain the leader by storing by lockName.
	// If not specified, a default UUID is generated.
	NodeID string
	// LockTTL sets the lock lifetime. By default is 1 second.
	LockTTL time.Duration
	// TryLockInterval is the interval at which an attempt will be made
	// to set a lock by the [netra.Run] function.
	TryLockInterval time.Duration
	// HearBeatInterval is the interval at which an attempt will be made
	// to extend the lock by the [netra.Run] function.
	HearBeatInterval time.Duration
	// OnLocked hook that runs asynchronously when a lock is set.
	OnLocked func()
	// OnLockLost hook that is executed asynchronously
	// when a lock is intentionally removed.
	OnLockLost func()
	// OnUnlocked hook that is executed asynchronously
	// when the lock is lost (for example, due to network delays).
	OnUnlocked func()
	// Backend stores the implementation of the interface [Backend] implementing interaction with storage.
	Backend Backend
}

// New creates a new Netra instance.
func New(cfg *Config) (*Netra, error) {
	netra := &Netra{}

	if cfg.LockName != "" {
		netra.lockName = cfg.LockName
	} else {
		netra.lockName = defaultLockName
	}

	if cfg.NodeID != "" {
		netra.nodeID = cfg.NodeID
	} else {
		netra.nodeID = uuid.NewString()
	}

	if cfg.LockTTL > 0 {
		netra.lockTTL = cfg.LockTTL
	} else {
		netra.lockTTL = defaultLockTTL
	}

	if cfg.TryLockInterval > 0 {
		netra.tryLockInterval = cfg.TryLockInterval
	} else {
		netra.tryLockInterval = defaultTryLockInterval
	}

	if cfg.HearBeatInterval > 0 {
		netra.hearBeatInterval = cfg.HearBeatInterval
	} else {
		netra.hearBeatInterval = defaultHeartBeatInterval
	}

	if cfg.OnLocked != nil {
		netra.onLocked = cfg.OnLocked
	} else {
		netra.onLocked = func() {}
	}

	if cfg.OnLockLost != nil {
		netra.onLockLost = cfg.OnLockLost
	} else {
		netra.onLockLost = func() {}
	}

	if cfg.OnUnlocked != nil {
		netra.onUnlocked = cfg.OnUnlocked
	} else {
		netra.onUnlocked = func() {}
	}

	if cfg.Backend != nil {
		netra.backend = cfg.Backend
	} else {
		return nil, ErrBackendNotProvided
	}

	return netra, nil
}

// TryLock attempts to acquire a lock. If successful, it asynchronously executes the [Config.OnLocked] hook
// and starts considering the node as a leader, otherwise it returns error.
func (n *Netra) TryLock(ctx context.Context) error {
	err := n.backend.TryLock(ctx, n.lockName, n.nodeID, n.lockTTL)

	if err == nil {
		n.isLeader.Store(true)
		go n.onLocked()
	}

	return err
}

// TryUnlock attempts to unlock the lock. If successful, it asynchronously executes the [Config.OnUnlocked] hook
// and stops considering the node as a leader, otherwise it returns an error.
func (n *Netra) TryUnlock(ctx context.Context) error {
	err := n.backend.TryUnlock(ctx, n.lockName, n.nodeID)

	if err == nil {
		n.isLeader.Store(false)
		go n.onUnlocked()
	}

	return err
}

// HeartBeat attempts to extend the lifetime of the lock. If this fails, it asynchronously
// executes the [Config.OnLockLost] hook and stops considering the node as a leader.
func (n *Netra) HeartBeat(ctx context.Context) error {
	if err := n.backend.HeartBeat(ctx, n.lockName, n.nodeID, n.lockTTL); err != nil {
		if errors.Is(err, backends.ErrLockHeldByAnotherNode) || errors.Is(err, backends.ErrLockDoesNotExist) {
			go n.onLockLost()
		}

		n.isLeader.Store(false)

		return err
	}

	return nil
}

// Run tries to establish a lock with a frequency of [Config.TryLockInterval] and
// extends it with a frequency of [Config.HeartBeatInterval] until an unhandled error occurs.
// If the lead is lost, it starts trying to establish it again.
func (n *Netra) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			tryLockTicker := time.NewTicker(n.tryLockInterval)

			for range tryLockTicker.C {
				if n.isLeader.Load() {
					break
				}

				tryLockСtx, tryLockCancel := context.WithTimeout(ctx, n.tryLockInterval)

				if err := n.TryLock(tryLockСtx); err != nil {
					tryLockCancel()

					if errors.Is(err, backends.ErrLockHeldByAnotherNode) {
						continue
					}

					// Must be checked before [tryLockCtx] to distinguish
					// [context.Canceled] and [context.DeadlineExceeded] errors
					if ctx.Err() != nil {
						return err
					}

					if errors.Is(tryLockСtx.Err(), context.DeadlineExceeded) {
						continue
					}

					return err
				}

				tryLockCancel()

				break
			}

			tryLockTicker.Stop()

			heartBeatTicker := time.NewTicker(n.hearBeatInterval)

			for range heartBeatTicker.C {
				if !n.isLeader.Load() {
					break
				}

				heartBeatCtx, heartBeatCancel := context.WithTimeout(ctx, n.hearBeatInterval)

				if err := n.HeartBeat(ctx); err != nil {
					heartBeatCancel()

					if errors.Is(err, backends.ErrLockHeldByAnotherNode) {
						break
					}

					// Must be checked before [tryLockCtx] to distinguish
					// [context.Canceled] and [context.DeadlineExceeded] errors
					if ctx.Err() != nil {
						return err
					}

					if errors.Is(heartBeatCtx.Err(), context.DeadlineExceeded) {
						continue
					}

					return err
				}

				heartBeatCancel()
			}

			heartBeatTicker.Stop()
		}
	}
}

// IsLeader returns whether the node is a leader.
func (n *Netra) IsLeader() bool {
	return n.isLeader.Load()
}

// Returns node's id.
func (n *Netra) GetNodeID() string {
	return n.nodeID
}
