package netra

import (
	"errors"
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
	TryLock() (bool, error)
	TryUnlock() (bool, error)
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

func TryLock() (bool, error) {
	return false, nil
}

func TryUnlock() (bool, error) {
	return false, nil
}

func (n *Netra) IsLeader() bool {
	return n.isLeader.Load()
}
