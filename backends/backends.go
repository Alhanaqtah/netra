package backends

import "errors"

// Custom backends' errors.
var (
	ErrLockHeldByAnotherNode = errors.New("lock is held by another node")
	ErrAlreadyHoldingLock    = errors.New("already holding the lock")
	ErrLockDoesNotExist      = errors.New("lock does not exist")
)
