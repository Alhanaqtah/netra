package backends

import "errors"

var (
	ErrLockExists        = errors.New("the lock is held by another node")
	ErrAlreadyLockHolder = errors.New("already the lock holder")
)
