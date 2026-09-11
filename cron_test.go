package cron

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

// fakeClock plays a fixed window of time through a job without waiting for it.
// Sleep jumps straight to each deadline; once the window is spent it blocks
// until the job's context ends, modelling a job waiting for a firing that never
// comes.
type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	stop  time.Time
	once  sync.Once
	spent chan struct{} // closed when the window is first exhausted
}

func newFakeClock(from, to time.Time) *fakeClock {
	return &fakeClock{now: from, stop: to, spent: make(chan struct{})}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) bool {
	c.mu.Lock()
	if at := c.now.Add(d); !at.After(c.stop) {
		c.now = at
		c.mu.Unlock()

		return true
	}
	c.mu.Unlock()

	c.once.Do(func() { close(c.spent) })
	<-ctx.Done()

	return false
}

func execute(t *testing.T, expected map[string]bool, from time.Time, to time.Time, cronExpression string, timeZone string) {
	t.Helper()

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(expected))
	unexpected := 0

	// A job fires strictly after the time it reads, so the clock starts a minute
	// early to keep `from` itself eligible.
	clk := newFakeClock(from.Add(-time.Minute), to)

	cj, err := newCronJob(context.Background(), cronExpression, timeZone, func(_ context.Context, tick time.Time) {
		key := tick.Format("2006-01-02 15:04")

		mu.Lock()
		defer mu.Unlock()

		if expected[key] {
			delete(expected, key)
			wg.Done()
		} else {
			unexpected++
		}
	}, clk)
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}
	defer cj.Stop()

	// Wait for the whole window to play out and every expected firing to arrive,
	// with a safety timeout so a missing firing fails the test instead of
	// hanging forever. The job may also run out of firings before the window
	// ends, in which case it stops on its own.
	done := make(chan struct{})
	go func() {
		select {
		case <-clk.spent:
		case <-cj.Done():
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	mu.Lock()
	defer mu.Unlock()

	if l := len(expected); l != 0 {
		t.Errorf("Job expected but did not fire at\n%v", reflect.ValueOf(expected).MapKeys())
	}

	if unexpected != 0 {
		t.Errorf("Job fired unexpectedly %d times.", unexpected)
	}
}

func TestCron_EveryMinute(t *testing.T) {
	from := time.Date(2018, 11, 11, 0, 1, 0, 0, time.UTC)
	to := time.Date(2018, 11, 11, 0, 5, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-11 00:01": true,
		"2018-11-11 00:02": true,
		"2018-11-11 00:03": true,
		"2018-11-11 00:04": true,
		"2018-11-11 00:05": true,
	}

	execute(t, expected, from, to, "* * * * *", "UTC")
}

func TestCron_FivePastHour(t *testing.T) {
	from := time.Date(2018, 11, 11, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 11, 11, 2, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-11 00:05": true,
		"2018-11-11 01:05": true,
	}

	execute(t, expected, from, to, "5 * * * *", "UTC")
}

func TestCron_OnceEachMonth(t *testing.T) {
	from := time.Date(2018, 10, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 12, 15, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-10-15 15:15": true,
		"2018-11-15 15:15": true,
	}

	execute(t, expected, from, to, "15 15 15 * *", "UTC")
}

func TestCron_OnceEachMonthOnWeekday(t *testing.T) {
	from := time.Date(2018, 10, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 12, 15, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-10-19 15:15": true,
		"2018-11-20 15:15": true,
	}

	execute(t, expected, from, to, "15 15 20W * *", "UTC")
}

func TestCron_LastDayOfMonth(t *testing.T) {
	from := time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2000, 5, 15, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2000-01-31 12:00": true,
		"2000-02-29 12:00": true,
		"2000-03-31 12:00": true,
		"2000-04-30 12:00": true,
	}

	execute(t, expected, from, to, "0 12 L * *", "UTC")
}

func TestCron_LastDayOfFebruary(t *testing.T) {
	from := time.Date(2000, 2, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2001, 3, 20, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2000-02-29 12:00": true,
		"2001-02-28 12:00": true,
	}

	execute(t, expected, from, to, "0 12 L FEB *", "UTC")
}

func TestCron_MondaysAt10and11(t *testing.T) {
	from := time.Date(2018, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 11, 30, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-05 10:00": true,
		"2018-11-05 11:00": true,
		"2018-11-12 10:00": true,
		"2018-11-12 11:00": true,
		"2018-11-19 10:00": true,
		"2018-11-19 11:00": true,
		"2018-11-26 10:00": true,
		"2018-11-26 11:00": true,
	}

	execute(t, expected, from, to, "0 10,11 * * MON", "UTC")
}

func TestCron_LastMondayAt10and11(t *testing.T) {
	from := time.Date(2018, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 11, 30, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-26 10:00": true,
		"2018-11-26 11:00": true,
	}

	execute(t, expected, from, to, "0 10,11 * * MONL", "UTC")
}

func TestCron_2ndMondayAt10and11(t *testing.T) {
	from := time.Date(2018, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2018, 11, 30, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-12 10:00": true,
		"2018-11-12 11:00": true,
	}

	execute(t, expected, from, to, "0 10,11 * * MON#2", "UTC")
}

func TestCron_ExactDateTime(t *testing.T) {
	from := time.Date(2017, 11, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2019, 11, 23, 0, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"2018-11-22 10:00": true,
	}

	execute(t, expected, from, to, "0 10 22 11 * 2018", "UTC")
}

func TestCron_EitherDayOrWeekday(t *testing.T) {
	from := time.Date(1978, 9, 30, 3, 0, 0, 0, time.UTC)
	to := time.Date(1978, 10, 31, 3, 0, 0, 0, time.UTC)
	expected := map[string]bool{
		"1978-10-01 12:12": true, // Because it's the first day of the month
		"1978-10-02 12:12": true, // The rest, because they are Mondays
		"1978-10-09 12:12": true,
		"1978-10-16 12:12": true,
		"1978-10-23 12:12": true,
		"1978-10-30 12:12": true,
	}

	execute(t, expected, from, to, "12 12 1 * MON", "UTC")
}

func TestCron_Stop(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	fireCtx := make(chan context.Context, 1)

	// A one minute window: the job fires once, then waits for a firing that
	// never comes.
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	cj, err := newCronJob(parent, "* * * * *", "UTC",
		func(ctx context.Context, _ time.Time) { fireCtx <- ctx },
		newFakeClock(from, from.Add(time.Minute)))
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}

	// Capture the context handed to the callback.
	var ctx context.Context
	select {
	case ctx = <-fireCtx:
	case <-time.After(time.Second):
		t.Fatal("Job did not fire.")
	}

	cj.Stop()

	// Stopping ends the dispatch loop...
	select {
	case <-cj.Done():
	case <-time.After(time.Second):
		t.Fatal("Job was not done after stop.")
	}

	// ...but does not cancel the context handed to the callback: that follows the
	// caller's parent context, not the job's lifecycle.
	select {
	case <-ctx.Done():
		t.Fatal("Callback context was cancelled by Stop; it should follow the parent context.")
	default:
	}

	// Cancelling the parent context does cancel the callback's context.
	cancelParent()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Callback context was not cancelled after the parent context was cancelled.")
	}
}

func TestCron_StopIsIdempotent(t *testing.T) {
	// An empty window, so the job waits rather than ever firing.
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	cj, err := newCronJob(context.Background(), "* * * * *", "UTC",
		func(context.Context, time.Time) {},
		newFakeClock(from, from))
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}

	// Multiple/concurrent Stop calls must neither panic nor block forever.
	done := make(chan struct{})
	go func() {
		cj.Stop()
		cj.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Repeated Stop calls did not return.")
	}
}

func TestCron_DoneOnScheduleExhausted(t *testing.T) {
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// The only year this schedule names is already in the past, so it has no
	// next firing. The job ends on its own, without Stop or a cancelled context,
	// and Done must still fire.
	cj, err := newCronJob(context.Background(), "0 0 1 1 * 2000", "UTC",
		func(context.Context, time.Time) {},
		newFakeClock(from, from))
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}

	select {
	case <-cj.Done():
	case <-time.After(time.Second):
		t.Fatal("Job was not done after the schedule ran out of firings.")
	}
}

func TestCron_DoneOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// An empty window, so the job waits rather than ever firing.
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	cj, err := newCronJob(ctx, "* * * * *", "UTC",
		func(context.Context, time.Time) {},
		newFakeClock(from, from))
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}

	// Cancelling the parent context stops the job without calling Stop; Done must
	// still fire.
	cancel()

	select {
	case <-cj.Done():
	case <-time.After(time.Second):
		t.Fatal("Job was not done after the parent context was cancelled.")
	}
}

func TestCron_FiresOnceAcrossFallBack(t *testing.T) {
	// 01:30 happens twice in New York on 2024-11-03, once on eastern daylight
	// time and again on eastern standard time. A schedule names wall-clock
	// times, so the job must fire once, not twice.
	from := time.Date(2024, 11, 3, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 11, 3, 12, 0, 0, 0, time.UTC)

	expected := map[string]bool{
		"2024-11-03 01:30": true,
	}

	execute(t, expected, from, to, "30 1 * * *", "America/New_York")
}

func TestCron_SkipsWallClockTimeLostToSpringForward(t *testing.T) {
	// 02:30 does not exist in New York on 2024-03-10, so the job fires on the
	// days either side of it and not on the day itself.
	from := time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 3, 12, 12, 0, 0, 0, time.UTC)

	expected := map[string]bool{
		"2024-03-09 02:30": true,
		// Nothing on the 10th: the clock jumps from 02:00 to 03:00 that day.
		"2024-03-11 02:30": true,
		"2024-03-12 02:30": true,
	}

	execute(t, expected, from, to, "30 2 * * *", "America/New_York")
}

// earlyClock always returns from Sleep short of the deadline, modelling a system
// clock that moved while the job was waiting. It gives up after a fixed number
// of wakes so a test cannot spin forever.
type earlyClock struct {
	mu    sync.Mutex
	now   time.Time
	wakes int
	once  sync.Once
	spent chan struct{} // closed once the wakes run out
}

func newEarlyClock(now time.Time) *earlyClock {
	return &earlyClock{now: now, spent: make(chan struct{})}
}

func (c *earlyClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *earlyClock) Sleep(ctx context.Context, d time.Duration) bool {
	c.mu.Lock()
	if c.wakes < 20 {
		c.wakes++
		c.now = c.now.Add(d / 2) // Never quite arrives.
		c.mu.Unlock()

		return true
	}
	c.mu.Unlock()

	c.once.Do(func() { close(c.spent) })
	<-ctx.Done()

	return false
}

func TestCron_DoesNotFireEarly(t *testing.T) {
	var mu sync.Mutex
	fired := 0

	clk := newEarlyClock(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

	cj, err := newCronJob(context.Background(), "0 0 * * *", "UTC",
		func(context.Context, time.Time) {
			mu.Lock()
			defer mu.Unlock()

			fired++
		}, clk)
	if err != nil {
		t.Fatalf("Unexpected error creating cron job: %v", err)
	}
	defer cj.Stop()

	// Every wake lands short of midnight, so none of them may fire.
	select {
	case <-clk.spent:
	case <-time.After(5 * time.Second):
		t.Fatal("Job did not exhaust the clock's early wakes.")
	}

	mu.Lock()
	defer mu.Unlock()

	if fired != 0 {
		t.Errorf("Job fired %d times before its due time.", fired)
	}
}
