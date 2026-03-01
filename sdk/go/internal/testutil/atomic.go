package testutil

import "sync/atomic"

// UpdateAtomicMaxInt64 sets currentMax to value when value is larger.
// It is safe for concurrent callers.
func UpdateAtomicMaxInt64(currentMax *atomic.Int64, value int64) {
	for {
		current := currentMax.Load()
		if value <= current {
			return
		}
		if currentMax.CompareAndSwap(current, value) {
			return
		}
	}
}
