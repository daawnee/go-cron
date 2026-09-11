# CLAUDE.md

Guidance for working in this repository.

## What this is

A Go library (`module github.com/daawnee/cron`, Go 1.26.3) that parses cron expressions and fires a
callback when a schedule is due. Single flat package `cron` — no `cmd/`,
`internal/`, or external dependencies (stdlib only).

## Layout

Flat package; one concern per file:

- `cron.go` — the job runner. Public `NewCronJob` + unexported `newCronJob` core.
- `clock.go` — the `clock` seam and `wallClock`, its system-time implementation.
- `parser.go` — `Parse`: regex-driven parsing of a cron string into a `Schedule`.
- `schedule.go` — `Schedule`, `Field`, the `SchedulePattern` interface, and the
  ~35 small pattern types (one per cron token, e.g. `MinuteRange`, `WeekdayNth`).
- `*_test.go` — white-box tests (`package cron`).

## Public API (keep this surface minimal)

- `NewCronJob(ctx, cronExpression, timeZone string, fire func(context.Context, time.Time)) (*CronJob, error)`
  — the primary entry point. The returned `*CronJob` exposes `Stop()` and
  `Done() <-chan struct{}` plus read-only `Schedule` / `Location` fields.
- `Parse(cronExpression string) (Schedule, error)` and the `Schedule` / `Field` /
  pattern types — exposed for callers who want to inspect or match schedules
  directly (`Schedule.Match`, `Schedule.NextMatching`).

Everything else (the dispatch loop, the clock) is unexported.

## Key design decisions

- **Expression fields**: `minute hour day-of-month month day-of-week [year]`.
  The first five are mandatory; the sixth **year** field is optional, so standard
  five-field cron expressions parse. Special characters: `* , - / L W #`. See
  `README.md` for the grammar.
- **No `*` year**: an omitted year already matches every year, so writing `*`
  there is rejected rather than accepted as a synonym. This is deliberate
  interop defence, not pedantry. Quartz's six-field form is
  `sec min hour dom mon dow`, so a Quartz expression's sixth field is the day of
  week where this library reads a year. Of ten real Quartz expressions, nine
  already fail here on `?`, field count, or a value out of range — the only one
  that used to parse silently as an unrelated schedule was
  `0 0 12 * * *` (Quartz for noon daily, read here as midnight on the 12th), and
  it did so purely because `*` was accepted as a year. For the same reason, do
  **not** add `?` as a synonym for `*`: rejecting it is what turns Quartz
  expressions into loud errors instead of wrong schedules.
- **Step semantics**: `*/N` means "every Nth value from the start of the field's
  range", so it is exactly equivalent to `min-max/N`: day `*/2` is `1-31/2`
  (1,3,5…), month `*/2` is `1-12/2` (Jan,Mar,May…), and year `*/N` counts from
  1900. For the zero-based fields (minute, hour, weekday) this coincides with
  `value % N == 0`, which is why only `DayEvery`, `MonthEvery` and `YearEvery`
  subtract a field minimum — the others are a bare modulus and are correct as
  such. `TestEveryStep_EquivalentToExplicitRange` guards the invariant across
  all six fields; don't "simplify" the offsets away.
- **DST semantics**: a cron expression names wall-clock times, and each one fires
  at most once. Following Quartz, a wall-clock time a transition *skips* never
  matches (`30 2 * * *` does not run on a spring-forward day), and one that
  occurs *twice* matches only at its first occurrence. `NextMatching` runs its
  whole search in **UTC**, where no offset ever changes, so hour and minute
  arithmetic cannot be knocked off a boundary; `inLocation` then resolves the
  answer back into the caller's zone, rejecting skipped times and preferring the
  earliest instant for repeated ones (`time.Date` does not promise which one it
  picks, and for `Australia/Lord_Howe` it picks the later). Searching in the
  caller's zone instead is what used to break half-hour DST
  (`Australia/Lord_Howe`), one-off jumps (`Asia/Kathmandu` gained 15 minutes in
  1986), and historical offsets carrying seconds (`America/St_Johns` at
  −3:30:52). `dst_test.go` sweeps all of those against a brute-force oracle —
  keep the zone list when touching this code.
- **Callback model**: `fire` is invoked as `go fire(ctx, tick)` — one goroutine
  per firing, fire-and-forget. The `ctx` handed to `fire` is the **caller's
  original context**, not the job's internal `jobCtx`, so `Stop` does not cancel
  in-flight callbacks. No backpressure; a slow callback can overlap with the next
  firing. The job does **not** track callbacks: `Stop`/`Done` wait only for the
  dispatch loop, and managing callback lifetime is the caller's job (deliberate —
  see lifecycle note).
- **Dispatch loop**: the runner sleeps until the next due time rather than
  ticking. Each pass asks `Schedule.NextMatching` for the next firing, sleeps
  that long, and fires — so a daily job wakes a handful of times a day instead of
  1440, and a repeated wall-clock time fires once, because `NextMatching` reports
  it once. `Sleep` may return early, so the loop re-checks the clock and derives
  the firing again rather than trusting the deadline arrived; that is what keeps
  a stepped system clock from firing a job early. When `NextMatching` returns nil
  the schedule is spent and the job ends on its own.
- **Clock injection seam**: `NewCronJob` wires up `wallClock`; the unexported
  `newCronJob` takes a `clock` (`Now` + `Sleep`) so tests jump from firing to
  firing instead of waiting. Validation runs *before* the loop starts, so an
  invalid job never spawns a goroutine. `wallClock.Sleep` caps any single wait at
  `maxSleep` (one hour) so a sparse schedule takes a fresh look periodically
  rather than sleeping through a clock step or a tzdata update; the cap lives in
  the clock, not the loop, because that is where the drift it guards against
  comes from. Nothing exercises the cap — every test supplies a fake clock.
- **Lifecycle**: `newCronJob` derives an internal `jobCtx` via
  `context.WithCancel(ctx)` that drives the clock and dispatch loop (the caller's
  original `ctx` is reserved for callbacks). A `sync.WaitGroup` tracks only the
  dispatch goroutine (callbacks are deliberately untracked — caller's
  responsibility); a watcher goroutine closes the `closed` channel once the loop
  exits. `Stop()` cancels `jobCtx` and waits on `closed` (idempotent — it only
  cancels and receives, never closes). `Done()` returns `closed`, so it fires on
  *any* termination path: `Stop`, parent-`ctx` cancellation, or the schedule
  running out of firings.
- **`MatchDay` semantics**: when both day-of-month and weekday are restricted, a
  match on *either* is sufficient (standard Unix cron behavior).

## Conventions

- Tests are white-box (`package cron`) so they can reach the unexported seam
  (`newCronJob`, the `clock` interface). Add new firing/lifecycle tests there.
- Firing tests go through the `execute` harness in `cron_test.go`: it runs the
  job on a `fakeClock` that jumps straight from one firing to the next across
  the window `from`..`to`, then asserts the job fired at exactly the
  `"2006-01-02 15:04"` keys in the `expected` map — neither missing nor extra.
  The clock starts a minute before `from`, since a job fires strictly *after*
  the time it reads. A whole test is usually just a `from`, a `to`, an
  `expected` map, and one `execute` call, and an entire simulated year costs no
  wall-clock time.
- Every exported identifier has a doc comment starting with its name; keep this
  up when adding to the public surface.
- Pattern types implement `SchedulePattern` (`Match` + `String`); `String` must
  round-trip back to valid cron syntax.

## Known quirks / non-standard behavior

- Step values aren't bounded by each field's max (e.g. minute `*/99` parses).
- `NextMatching` is bounded to years ≤ 2099, so a job scheduled past that ends
  rather than waiting forever.
- There is no catch-up. A job whose process is suspended past a firing derives
  the next one from the current time on waking, so the missed firing is skipped
  rather than run late.

## Build / test

```sh
go build ./...
go vet ./...
go test -race ./...    # firing is concurrent — always run with -race
gofmt -l .             # must print nothing
```

Single test (there are no subtests, so `-run` takes the function name):

```sh
go test -race -run '^TestCron_LastDayOfMonth$' .
```

## Keeping this file current

Update CLAUDE.md whenever the public API, file layout, or a design decision
above changes. Recent direction has been to *shrink* the public surface (the
ticker parameter removed, the clock unexported and then replaced outright by the
`clock` seam) — preserve that bias toward a minimal exported API.
