package timers

import (
	"math"
	"time"
)

// NextRebindingMoment calculates next time we need renew or rebound lease.
// The calculated time will be somewhere between now and given time t
//
// @t is either T2 or end of Lease time
func NextRebindingMoment(t time.Time) time.Duration {
	now := time.Now()
	return time.Duration(math.Max(float64(60*time.Second), float64(t.Sub(now)/2)))
}

func SafeReset(timer *time.Timer, d time.Duration) time.Duration {
	SafeStop(timer)

	timer.Reset(d)

	return d
}

func SafeStop(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
			// draining timers channel
		default:
			// timer channel already drained
		}
	}
}
