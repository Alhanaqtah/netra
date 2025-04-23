package netra

import (
	"context"
	"errors"
	"netra/backends"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLockName = "NETRA:LOCK:729522b5-a3b3-47d1-875e-c65abbc04a3e"
	defaultTTL      = 1 * time.Second
)

var (
	ErrBackendNotProvided = errors.New("backend not provided")
)

type Backend interface {
	TryLock(ctx context.Context, lockName, nodeID string) (bool, error)
	TryUnlock(ctx context.Context, lockName, nodeID string) (bool, error)
	HeartBeat(ctx context.Context, lockName, nodeID string) error
}

type Netra struct {
	lockName   string
	nodeID     string
	lockTTL    time.Duration
	onLocked   func()
	onUnlocked func()
	isLeader   atomic.Bool
	backend    Backend
}

type Config struct {
	LockName   string
	LockTTL    time.Duration
	OnLocked   func()
	OnUnlocked func()
	Backend    Backend
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
		netra.lockTTL = defaultTTL
	}

	if cfg.OnLocked != nil {
		netra.onLocked = cfg.OnLocked
	} else {
		netra.onLocked = func() {}
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
	ok, err := n.backend.TryLock(ctx, n.lockName, n.nodeID)

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
			go n.onUnlocked()
		}

		return err
	}

	return nil
}

func (n *Netra) Run(ctx context.Context) error {
	tryLockInterval := n.lockTTL
	heatBeatInterval := time.Duration(float64(n.lockTTL) * 0.9) // 90% of lock's ttl

	for {
		if !n.isLeader.Load() {
			ctx, cancel := context.WithTimeout(ctx, tryLockInterval)

			if _, err := n.backend.TryLock(ctx, n.lockName, n.nodeID); err != nil {
				cancel()

				if errors.Is(err, backends.ErrLockExists) || errors.Is(err, context.DeadlineExceeded) {
					time.Sleep(tryLockInterval)
					continue
				}

				return err
			}

			cancel()

			n.isLeader.Store(true)
			go n.onLocked()

			time.Sleep(heatBeatInterval)
		}

		if n.isLeader.Load() {
			ctx, cancel := context.WithTimeout(ctx, heatBeatInterval)

			if err := n.HeartBeat(ctx); err != nil {
				cancel()

				if errors.Is(err, backends.ErrLockExists) || errors.Is(err, context.DeadlineExceeded) {
					n.isLeader.Store(false)
					go n.onUnlocked()

					time.Sleep(tryLockInterval)

					continue
				}

				return err
			}

			cancel()

			time.Sleep(heatBeatInterval)
		}
	}
}

func (n *Netra) IsLeader() bool {
	return n.isLeader.Load()
}

func (n *Netra) GetNodeID() string {
	return n.nodeID
}
