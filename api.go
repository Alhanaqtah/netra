package netra

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLockName = "LEADER_729522b5-a3b3-47d1-875e-c65abbc04a3e"
	defaultTTL      = 1 * time.Second
)

var (
	ErrBackendNotProvided = errors.New("backend not provided")
)

type Backend interface {
}

type Netra struct {
	lockName string
	nodeID   string
	ttl      time.Duration

	onElected func()
	onRevoked func()

	isLeader atomic.Bool
	backend  Backend
}

type Config struct {
	LockName string
	TTL      time.Duration

	OnElected func()
	OnRevoked func()

	Backend Backend
}

func New(cfg *Config) (*Netra, error) {
	netra := &Netra{}

	if cfg.LockName != "" {
		netra.lockName = cfg.LockName
	} else {
		netra.lockName = defaultLockName
	}

	netra.nodeID = uuid.NewString()

	if cfg.TTL >= 0 {
		netra.ttl = cfg.TTL
	} else {
		netra.ttl = defaultTTL
	}

	if cfg.OnElected != nil {
		netra.onElected = cfg.OnElected
	} else {
		netra.onElected = func() {}
	}

	if cfg.OnRevoked != nil {
		netra.onRevoked = cfg.OnRevoked
	} else {
		netra.onRevoked = func() {}
	}

	if cfg.Backend != nil {
		netra.backend = cfg.Backend
	} else {
		return nil, ErrBackendNotProvided
	}

	return netra, nil
}

func (n *Netra) Start() error {
	return nil
}

func (n *Netra) Stop() error {
	return nil
}

func (n *Netra) TryBecomeLeader() {

}

func (n *Netra) IsLeader() bool {
	return n.isLeader.Load()
}
