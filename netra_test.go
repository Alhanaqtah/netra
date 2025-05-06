package netra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alhanaqtah/netra/backends"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	lockName = "lock-name"
	nodeID   = "node-id"
	lockTTL  = 1 * time.Second
	someErr  = errors.New("some error")
)

type StubBackend struct{}

func (sb StubBackend) TryLock(ctx context.Context, lockName, nodeID string, ttl time.Duration) (bool, error) {
	return false, nil
}

func (sb StubBackend) TryUnlock(ctx context.Context, lockName, nodeID string) (bool, error) {
	return false, nil
}

func (sb StubBackend) HeartBeat(ctx context.Context, lockName, nodeID string, ttl time.Duration) error {
	return nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		cfg           Config
		expectedError error
	}{
		{
			name: "Succesfull",
			cfg: Config{
				Backend: StubBackend{},
			},
			expectedError: nil,
		},
		{
			name:          "Error: Backend not provided",
			cfg:           Config{},
			expectedError: ErrBackendNotProvided,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(&tc.cfg)

			assert.ErrorIs(t, err, tc.expectedError)
		})
	}
}

func TestTryLock(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *MockBackend)
		expectedIsLeader bool
		expectedOk       bool
		expectedError    error
	}{
		{
			name: "Succesfull case",
			setup: func(m *MockBackend) {
				m.On("TryLock",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(true, nil).Once()
			},
			expectedIsLeader: true,
			expectedOk:       true,
			expectedError:    nil,
		},
		{
			name: "Error case",
			setup: func(m *MockBackend) {
				m.On("TryLock",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(false, someErr).Once()
			},
			expectedIsLeader: false,
			expectedOk:       false,
			expectedError:    someErr,
		},
	}

	backend := new(MockBackend)

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

			ok, err := n.TryLock(context.Background())

			assert.Equal(t, tc.expectedIsLeader, n.isLeader.Load())
			assert.Equal(t, tc.expectedOk, ok)
			assert.ErrorIs(t, err, tc.expectedError)

			// Clean
			n.isLeader.Store(false)
		})
	}
}

func TestUnlock(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *MockBackend)
		expectedIsLeader bool
		expectedOk       bool
		expectedError    error
	}{
		{
			name: "Succesfull case",
			setup: func(m *MockBackend) {
				m.On("TryUnlock",
					mock.Anything,
					lockName,
					nodeID,
				).Return(true, nil).Once()
			},
			expectedIsLeader: false,
			expectedOk:       true,
			expectedError:    nil,
		},
		{
			name: "Error case",
			setup: func(m *MockBackend) {
				m.On("TryUnlock",
					mock.Anything,
					lockName,
					nodeID,
				).Return(false, someErr).Once()
			},
			expectedIsLeader: true,
			expectedOk:       false,
			expectedError:    someErr,
		},
	}

	backend := new(MockBackend)

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

			ok, err := n.TryUnlock(context.Background())

			assert.Equal(t, tc.expectedIsLeader, n.isLeader.Load())
			assert.Equal(t, tc.expectedOk, ok)
			assert.ErrorIs(t, err, tc.expectedError)
		})

		n.isLeader.Store(true)
	}
}

func TestHeartBeat(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(m *MockBackend)
		expectedIsLeader bool
		expectedError    error
	}{
		{
			name: "Successfull case",
			setup: func(m *MockBackend) {
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
			setup: func(m *MockBackend) {
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
			setup: func(m *MockBackend) {
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
			setup: func(m *MockBackend) {
				m.On("HeartBeat",
					mock.Anything,
					lockName,
					nodeID,
					lockTTL,
				).Return(someErr).Once()
			},
			expectedIsLeader: false,
			expectedError:    someErr,
		},
	}

	backend := new(MockBackend)

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
	n := Netra{
		nodeID: nodeID,
	}

	assert.Equal(t, nodeID, n.GetNodeID())
}
