package netra

import (
	"context"
	"errors"
	"netra/backends"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Configuration defaults
const (
	defaultLockName          = "NETRA:LOCK:729522b5-a3b3-47d1-875e-c65abbc04a3e"
	defaultLockTTL           = 1 * time.Second
	defaultTryLockInterval   = 1 * time.Second
	defaultHeartBeatInterval = 500 * time.Millisecond
)

var (
	ErrBackendNotProvided = errors.New("backend not provided")
)

type Backend interface {
	TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) (bool, error)
	TryUnlock(ctx context.Context, lockName, nodeID string) (bool, error)
	HeartBeat(ctx context.Context, lockName, nodeID string) error
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
	LockName         string
	LockTTL          time.Duration
	TryLockInterval  time.Duration
	HearBeatInterval time.Duration
	OnLocked         func()
	OnLockLost       func()
	OnUnlocked       func()
	Backend          Backend
}

func New(cfg *Config) (*Netra, error) {
	netra := &Netra{}

	if cfg.LockName != "" {
		netra.lockName = cfg.LockName
	} else {
		netra.lockName = defaultLockName
	}

	netra.nodeID = uuid.NewString()

	if cfg.LockTTL >= 0 {
		netra.lockTTL = cfg.LockTTL
	} else {
		netra.lockTTL = defaultLockTTL
	}

	if cfg.TryLockInterval >= 0 {
		netra.tryLockInterval = cfg.TryLockInterval
	} else {
		netra.tryLockInterval = defaultTryLockInterval
	}

	if cfg.HearBeatInterval >= 0 {
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

func (n *Netra) TryLock(ctx context.Context) (bool, error) {
	ok, err := n.backend.TryLock(ctx, n.lockName, n.nodeID, n.lockTTL)

	if ok {
		n.isLeader.Store(true)
		go n.onLocked()
	}

	return ok, err
}

func (n *Netra) TryUnlock(ctx context.Context) (bool, error) {
	ok, err := n.backend.TryUnlock(ctx, n.lockName, n.nodeID)

	if ok {
		n.isLeader.Store(false)
		go n.onUnlocked()
	}

	return ok, err
}

func (n *Netra) HeartBeat(ctx context.Context) error {
	if err := n.backend.HeartBeat(ctx, n.lockName, n.nodeID); err != nil {
		if errors.Is(err, backends.ErrLockExists) {
			n.isLeader.Store(false)
			go n.onLockLost()
		}

		return err
	}

	return nil
}

func (n *Netra) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			tryLockTicker := time.NewTicker(n.tryLockInterval)

			for range tryLockTicker.C {
				if !n.isLeader.Load() {
					ctx, cancel := context.WithTimeout(ctx, n.tryLockInterval)

					ok, err := n.TryLock(ctx)
					if err != nil {
						cancel()

						if errors.Is(err, backends.ErrLockExists) || errors.Is(err, context.DeadlineExceeded) {
							continue
						}

						return err
					}

					cancel()

					if ok {
						break
					}
				} else {
					break
				}
			}

			tryLockTicker.Stop()

			heartBeatTicker := time.NewTicker(n.hearBeatInterval)

			for range heartBeatTicker.C {
				if n.isLeader.Load() {
					ctx, cancel := context.WithTimeout(ctx, n.hearBeatInterval)

					if err := n.HeartBeat(ctx); err != nil {
						cancel()

						if errors.Is(err, backends.ErrLockExists) || errors.Is(err, context.DeadlineExceeded) {
							break
						}

						return err
					}

					cancel()
				} else {
					break
				}
			}

			heartBeatTicker.Stop()
		}
	}
}

func (n *Netra) IsLeader() bool {
	return n.isLeader.Load()
}

func (n *Netra) GetNodeID() string {
	return n.nodeID
}
