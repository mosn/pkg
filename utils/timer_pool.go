package utils

import (
	"sync"
	"time"
)

var timerPool sync.Pool

// AcquireTimer from pool.
func AcquireTimer(d time.Duration) *time.Timer {
	v := timerPool.Get()
	if v == nil {
		return newTimer(d)
	}

	tm := v.(*time.Timer)
	if tm.Reset(d) {
		panic("Received an active timer from the pool!")
	}

	return tm
}

// ReleaseTimer to pool.
func ReleaseTimer(tm *time.Timer) {
	timerPool.Put(tm)
}

func newTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}
