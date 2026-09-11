# Cron package for Go

This package implements a standard [Cron](https://en.wikipedia.org/wiki/Cron) scheduling.

## Features

### Schedule definition

#### Follows the standard _cron expression_ format
```
┌─────────── minute (0 - 59)
│ ┌───────── hour (0 - 23)
│ │ ┌─────── day of the month (1 - 31)
│ │ │ ┌───── month (1 - 12)
│ │ │ │ ┌─── day of the week (0 - 6) (Sunday to Saturday)
│ │ │ │ │ ┌─ year (1900-2099, optional)
* * * * * 2026
```

The first five fields are mandatory. The sixth **year** field is optional: when
it is omitted, the expression is interpreted as a standard five-field cron
expression matching every year. For example, `30 8 * * 1-5` runs at 08:30 on
weekdays, in every year.

A bare `*` in the year field is **rejected** — omit the field instead, which
says the same thing. Steps like `*/4` are still fine. This is deliberate:
six-field [Quartz](https://www.quartz-scheduler.org/) expressions put the day of
week where this library reads a year, so `0 0 12 * * *` means noon every day in
Quartz but would read here as midnight on the 12th. Rejecting the bare `*`
turns that into a clear error rather than a wrong schedule.

#### Supported schedule definition features
```
Field        | Allowed values  | Allowed special characters
-----------------------------------------------------------
Minutes	     | 0–59            | * , - /
Hours	     | 0–23            | * , - /	
Day of month | 1–31            | * , - / L W
Month        | 1–12 or JAN–DEC | * , - /	
Day of week  | 0–6 or SUN–SAT  | * , - / L #
Year         | 1900–2099       | , - / and */N (a bare * is rejected)
```

__Special characters__

For the meaning and usage of the special characters (__`L`__, __`W`__, __`#`__, and __`/`__), please refer to the corresponding [Wikipedia page](https://en.wikipedia.org/wiki/Cron#Non-standard_characters).

One point worth spelling out, because it is a common misreading: `/` is a **step
across the field's range**, not a modulus. It starts at the first value the field
allows. So `*/2` means the 1st, 3rd, 5th … in day of month, which starts at 1,
but 0, 2, 4 … in minutes, which starts at 0. `*/N` is always exactly equivalent
to writing `min-max/N` out in full.

### Time Zones

Each job also defines the time zone it is to be interpreted in.
The time zone has to be defined according to the [IANA Time Zone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) format.

---

## Installation

```sh
go get github.com/daawnee/cron
```

```go
import "github.com/daawnee/cron"
```

The package has no dependencies beyond the standard library.

---

## Usage

### Creating a new cron job

```go
func NewCronJob(ctx context.Context, cronExpression string, timeZone string, fire func(context.Context, time.Time)) (*CronJob, error)
```
Starts a cron job that invokes `fire` in a new goroutine on every whole minute that matches the schedule. The `ctx` you pass to `NewCronJob` and the matching time (in the job's time zone) are passed to `fire` — callbacks run under your context, not the job's internal one, so stopping the job does not cancel work already handed to `fire`.

It will return with an error in case of an invalid `cronExpression` or `timeZone`. On success it returns a `*CronJob` handle. The job also stops on its own when `ctx` is cancelled.

* `ctx` - the context
* `cronExpression` - in standard [cron format](https://en.wikipedia.org/wiki/Cron#Overview). See above.
* `timeZone` - in [IANA format](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)
* `fire` - called as a goroutine, with the matching time, each time the job is due.

The job sleeps until its next due time rather than waking every minute to check, so an hourly or daily schedule costs almost nothing while it waits. `NewCronJob` wires that up automatically; there is nothing else to configure.

A cron expression names wall-clock times, and each one fires at most once. Across a daylight-saving transition that means a time the clock *skips* never fires — a job set for 02:30 does not run on the day the clock jumps from 02:00 to 03:00 — and a time that happens *twice* fires only on its first occurrence. This matches [Quartz](https://www.quartz-scheduler.org/documentation/2.3.1-SNAPSHOT/faq.html), and it holds in awkward zones too, including ones whose transition is half an hour (`Australia/Lord_Howe`) rather than a whole one.

There is no catch-up: if the process is suspended past a firing, the job works out its next due time on waking, and the missed one is skipped rather than run late.

### Controlling a running job

```go
func (c *CronJob) Stop()
func (c *CronJob) Done() <-chan struct{}
```

* `Stop` cancels the job and blocks until the dispatch loop has returned, so no further callbacks will be started. Callbacks already running are unaffected: they run under the `ctx` you passed to `NewCronJob` (which `Stop` does not cancel) and are **not** waited for — managing their lifetime (e.g. with your own `sync.WaitGroup`) is your responsibility. `Stop` is safe to call more than once and from multiple goroutines.
* `Done` returns a channel that is closed once the dispatch loop has exited — whether via `Stop`, cancellation of the parent `ctx`, or the schedule running out of firings.

### Example

```go
ctx := context.Background()

job, err := cron.NewCronJob(
    ctx,
    "0 0 * * *",     // Triggered every day at midnight
    "Europe/London", // according to the London, UK time zone
    func(ctx context.Context, firedAt time.Time) {
        // Do something here.
        // This runs in its own goroutine for every firing.
    },
)
if err != nil {
    // Handle an invalid cron expression or time zone.
}
defer job.Stop() // Stop the job (and wait for it to drain) when no longer needed.
```

### Inspecting a schedule without running one

A schedule can be parsed and queried on its own, with no goroutine and no clock —
useful for validating user input, showing someone when their job will next run,
or testing scheduling logic.

```go
func Parse(cronExpression string) (Schedule, error)

func (s Schedule) Match(t time.Time) bool
func (s Schedule) NextMatching(relativeTo time.Time) *time.Time
func (s Schedule) String() string
```

* `Match` reports whether a given time satisfies the schedule, to the minute.
* `NextMatching` returns the earliest time strictly after `relativeTo` that the schedule matches, in `relativeTo`'s time zone, or `nil` if there is none before the year 2100. It follows the same wall-clock rules described above, and it is what drives a running job.
* `String` renders the schedule back into an expression that `Parse` accepts again.

```go
schedule, err := cron.Parse("0 9 * * MON-FRI")
if err != nil {
    // Handle an invalid cron expression.
}

if next := schedule.NextMatching(time.Now()); next != nil {
    fmt.Println("next run:", next)
}
```

The parsed `Schedule` also exposes its fields — `Minute`, `Hour`, `Day`, `Month`,
`Weekday` and `Year` — each a `Field` holding the alternatives allowed in that
position, so a schedule can be examined rather than only executed.

---

## Licence

[MIT](LICENSE).
