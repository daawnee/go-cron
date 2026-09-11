// Package cron parses cron expressions and fires callbacks when they are due.
//
// Expressions use a six-field format: minute, hour, day-of-month, month,
// day-of-week, and year. The standard special characters (* , - / L W #) are
// supported; see the package README for the full grammar.
//
// Parse turns an expression into a Schedule, which can be queried directly via
// Match and NextMatching. NewCronJob drives a Schedule from the wall clock and
// invokes a callback on each matching minute, returning a handle to stop it.
package cron

import (
	"context"
	"sync"
	"time"
)

/***********
 * CronJob *
 ***********/

// CronJob is a running cron job. Use Stop to cancel it and Done to observe when
// it has fully stopped. The Schedule and Location fields are read-only metadata
// describing the parsed job.
type CronJob struct {
	ctxCncl context.CancelFunc
	wg      sync.WaitGroup
	closed  chan struct{} // closed once the dispatch loop has finished; fire callbacks are not tracked

	Schedule Schedule
	Location *time.Location
}

// NewCronJob starts a cron job that invokes fire, in a new goroutine, on every
// whole minute that matches cronExpression interpreted in the given IANA
// timeZone. The ctx passed to NewCronJob and the matching time (in timeZone) are
// passed to fire — callbacks run under the caller's context, not the job's
// internal one.
//
// It returns an error if cronExpression or timeZone is invalid. On success it
// returns a *CronJob; call CronJob.Stop to cancel the job and CronJob.Done to
// observe when it has stopped. The job also stops on its own when ctx is
// cancelled, or once the schedule has no further firings.
func NewCronJob(ctx context.Context, cronExpression string, timeZone string, fire func(context.Context, time.Time)) (*CronJob, error) {
	return newCronJob(ctx, cronExpression, timeZone, fire, wallClock{})
}

// newCronJob is the testable core of NewCronJob. The clock supplies the job's
// view of time: production passes wallClock, while tests pass a fake that jumps
// from firing to firing, decoupling them from real time. Validation runs before
// the dispatch loop starts, so an invalid job never spawns a goroutine.
func newCronJob(ctx context.Context, cronExpression string, timeZone string, fire func(context.Context, time.Time), clk clock) (*CronJob, error) {
	schedule, err := Parse(cronExpression)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return nil, err
	}

	// jobCtx is derived from the caller's ctx and drives the job's own lifecycle
	// (the clock and the dispatch loop); Stop cancels it. Callbacks, however,
	// receive the caller's original ctx (see below).
	jobCtx, cancel := context.WithCancel(ctx)

	cj := &CronJob{
		ctxCncl:  cancel,
		closed:   make(chan struct{}),
		Schedule: schedule,
		Location: location,
	}

	cj.wg.Add(1)
	go func() {
		defer cj.wg.Done()

		for {
			now := clk.Now().In(location)

			next := schedule.NextMatching(now)
			if next == nil {
				// The schedule has no further firings.
				return
			}

			if !clk.Sleep(jobCtx, next.Sub(now)) {
				return
			}

			if clk.Now().Before(*next) {
				// Sleep came back early, either because it capped the wait or
				// because the system clock moved while it ran. The firing is
				// still ahead, so derive it again rather than firing early.
				continue
			}

			// Callbacks are fire-and-forget and receive the caller's original
			// ctx, not the job's internal one — so stopping the job does not
			// cancel work already handed to fire. The job neither tracks nor
			// waits for callbacks; managing their lifetime is the caller's
			// responsibility.
			go fire(ctx, *next)
		}
	}()

	// Signal completion once the dispatch loop has returned — whether the job
	// ended via Stop, parent context cancellation, or a closed clock channel.
	go func() {
		cj.wg.Wait()
		close(cj.closed)
	}()

	return cj, nil
}

// Stop cancels the job and blocks until the dispatch loop has returned, so no
// further callbacks will be started. Callbacks already running are unaffected:
// they run under the context passed to NewCronJob, which Stop does not cancel,
// and are not waited for — managing their lifetime is the caller's
// responsibility. Stop is safe to call more than once and from multiple
// goroutines.
func (c *CronJob) Stop() {
	c.ctxCncl()
	<-c.closed
}

// Done returns a channel that is closed once the dispatch loop has exited —
// regardless of whether the job was stopped via Stop, parent context
// cancellation, or the schedule running out of firings. It does not track
// in-flight fire callbacks.
func (c *CronJob) Done() <-chan struct{} {
	return c.closed
}
