package netra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alhanaqtah/netra/backends"
	netra_mocks "github.com/Alhanaqtah/netra/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	lockName = "lock-name"
	nodeID   = "node-id"
	lockTTL  = 1 * time.Second
	errSome  = errors.New("some error")
)

type StubBackend struct{}

func (sb StubBackend) TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	return nil
}

func (sb StubBackend) TryUnlock(ctx context.Context, lockName, nodeID string) error {
	return nil
}

func (sb StubBackend) HeartBeat(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	return nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		expectedNetra *Netra
		expectedError error
	}{
		{
			name: "Provide full configuration",
			cfg: &Config{
				LockName:         defaultLockName,
				NodeID:           nodeID,
				LockTTL:          lockTTL,
				TryLockInterval:  10 * time.Second,
				HearBeatInterval: 10 * time.Second,
				Backend:          StubBackend{},
			},
			expectedNetra: &Netra{
				lockName:         defaultLockName,
				nodeID:           nodeID,
				lockTTL:          lockTTL,
				tryLockInterval:  10 * time.Second,
				hearBeatInterval: 10 * time.Second,
				backend:          StubBackend{},
			},
			expectedError: nil,
		},
		{
			name: "Error: Backend not provided",
			cfg: &Config{
				LockName:         defaultLockName,
				NodeID:           nodeID,
				LockTTL:          lockTTL,
				TryLockInterval:  10 * time.Second,
				HearBeatInterval: 10 * time.Second,
			},
			expectedNetra: nil,
			expectedError: ErrBackendNotProvided,
		},
		{
			name: "Unnecessary config fields not provided, defaults expected",
			cfg: &Config{
				Backend: StubBackend{},
			},
			expectedNetra: &Netra{
				lockName:         defaultLockName,
				lockTTL:          defaultLockTTL,
				tryLockInterval:  defaultTryLockInterval,
				hearBeatInterval: defaultHeartBeatInterval,
			},
			expectedError: nil,
		},
		{
			name:          "Error: Config not provided",
			cfg:           nil,
			expectedNetra: nil,
			expectedError: ErrConfigNotProvided,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := New(tc.cfg)

			assert.ErrorIs(t, err, tc.expectedError)

			// because we cann't mock uuid.NewString()
			if tc.cfg != nil && tc.cfg.NodeID == "" {
				tc.expectedNetra.nodeID = n.nodeID
			}

			if n != nil {
				assert.Equal(t, tc.expectedNetra.lockName, n.lockName)
				assert.Equal(t, tc.expectedNetra.nodeID, n.nodeID)
				assert.Equal(t, tc.expectedNetra.lockTTL, n.lockTTL)
				assert.Equal(t, tc.expectedNetra.tryLockInterval, n.tryLockInterval)
				assert.Equal(t, tc.expectedNetra.hearBeatInterval, n.hearBeatInterval)
				assert.NotNil(t, n.backend)
			}
		})
	}
}

func TestTryLock(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *netra_mocks.MockBackend)
		expectedIsLeader bool
		expectedError    error
	}{
		{
			name: "Succesfull case",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("TryLock",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(nil).Once()
			},
			expectedIsLeader: true,
			expectedError:    nil,
		},
		{
			name: "Error case",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("TryLock",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(errSome).Once()
			},
			expectedIsLeader: false,
			expectedError:    errSome,
		},
	}

	backend := new(netra_mocks.MockBackend)

	n, err := New(&Config{
		Backend: backend,
	})
	assert.NoError(t, err)

	n.lockName = lockName
	n.nodeID = nodeID
	n.lockTTL = lockTTL

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(backend)

			err := n.TryLock(context.Background())

			assert.Equal(t, tc.expectedIsLeader, n.isLeader.Load())
			assert.ErrorIs(t, err, tc.expectedError)

			// Clean
			n.isLeader.Store(false)
		})
	}
}

func TestUnlock(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *netra_mocks.MockBackend)
		expectedIsLeader bool
		expectedError    error
	}{
		{
			name: "Succesfull case",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("TryUnlock",
					mock.Anything,
					lockName,
					nodeID,
				).Return(nil).Once()
			},
			expectedIsLeader: false,
			expectedError:    nil,
		},
		{
			name: "Error case",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("TryUnlock",
					mock.Anything,
					lockName,
					nodeID,
				).Return(errSome).Once()
			},
			expectedIsLeader: true,
			expectedError:    errSome,
		},
	}

	backend := new(netra_mocks.MockBackend)

	n, err := New(&Config{
		Backend: backend,
	})
	assert.NoError(t, err)

	n.lockName = lockName
	n.nodeID = nodeID
	n.lockTTL = lockTTL

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(backend)

			err := n.TryUnlock(context.Background())

			assert.Equal(t, tc.expectedIsLeader, n.isLeader.Load())
			assert.ErrorIs(t, err, tc.expectedError)
		})

		n.isLeader.Store(true)
	}
}

func TestHeartBeat(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *netra_mocks.MockBackend)
		expectedIsLeader bool
		expectedError    error
	}{
		{
			name: "Successful case",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("HeartBeat",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(nil).Once()
			},
			expectedIsLeader: true,
			expectedError:    nil,
		},
		{
			name: "Lock held by another node",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("HeartBeat",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(backends.ErrLockHeldByAnotherNode).Once()
			},
			expectedIsLeader: false,
			expectedError:    backends.ErrLockHeldByAnotherNode,
		},
		{
			name: "Lock doesn't exist",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("HeartBeat",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(backends.ErrLockDoesNotExist).Once()
			},
			expectedIsLeader: false,
			expectedError:    backends.ErrLockDoesNotExist,
		},
		{
			name: "Some another error",
			setup: func(m *netra_mocks.MockBackend) {
				m.On("HeartBeat",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(errSome).Once()
			},
			expectedIsLeader: false,
			expectedError:    errSome,
		},
	}

	backend := new(netra_mocks.MockBackend)

	n, err := New(&Config{
		Backend: backend,
	})
	assert.NoError(t, err)

	n.lockName = lockName
	n.nodeID = nodeID
	n.lockTTL = lockTTL
	n.isLeader.Store(true)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(backend)

			err = n.HeartBeat(context.Background())

			assert.Equal(t, tc.expectedIsLeader, n.isLeader.Load())
			assert.ErrorIs(t, err, tc.expectedError)
		})

		n.isLeader.Store(true)
	}
}

func TestIsLeader(t *testing.T) {
	n := Netra{}

	n.isLeader.Store(true)
	assert.Equal(t, true, n.IsLeader())

	n.isLeader.Store(false)
	assert.Equal(t, false, n.IsLeader())
}

func TestGetNodeID(t *testing.T) {
	n, err := New(&Config{
		NodeID:  nodeID,
		Backend: StubBackend{},
	})

	require.NoError(t, err)
	require.Equal(t, nodeID, n.GetNodeID())

	n, err = New(&Config{
		Backend: StubBackend{},
	})

	require.NoError(t, err)
	require.Equal(t, n.nodeID, n.GetNodeID())
}
