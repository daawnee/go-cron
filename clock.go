package cron

import (
	"context"
	"time"
)

// maxSleep bounds a single wait on the wall clock. System time can be stepped
// and the time zone database can be updated while a job waits, so a sparse
// schedule wakes at least this often to take a fresh look rather than sleeping
// through months of it.
const maxSleep = time.Hour

// clock is the job's view of time. NewCronJob wires up wallClock; tests supply a
// fake so a schedule can be played through without waiting for it.
type clock interface {
	// Now reports the current time.
	Now() time.Time
	// Sleep waits for up to d, reporting false if ctx ended first. It may
	// return before d has elapsed, so callers must check the time again rather
	// than assume the deadline arrived.
	Sleep(ctx context.Context, d time.Duration) bool
}

// wallClock is the clock backed by system time.
type wallClock struct{}

// Now reports the current system time.
func (wallClock) Now() time.Time {
	return time.Now()
}

// Sleep waits for up to d, capped at maxSleep, reporting false if ctx ended
// first.
func (wallClock) Sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(min(d, maxSleep))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
