package cron

import (
	"sort"
	"testing"
	"time"
)

func mustLoad(t *testing.T, zone string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatalf("cannot load zone %q: %v", zone, err)
	}

	return loc
}

// wallOf renders t's wall clock as a UTC time, the form the search works in.
func wallOf(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, t.Hour(), t.Minute(), 0, 0, time.UTC)
}

func TestInLocation(t *testing.T) {
	for _, c := range []struct {
		name   string
		zone   string
		wall   time.Time // read as a wall clock, not an instant
		exists bool
		at     time.Time // expected instant, when it exists
	}{
		{
			name:   "ordinary time resolves to itself",
			zone:   "America/New_York",
			wall:   time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
			exists: true, at: time.Date(2024, 6, 15, 16, 0, 0, 0, time.UTC),
		},
		{
			name:   "wall clock skipped by spring forward does not exist",
			zone:   "America/New_York",
			wall:   time.Date(2024, 3, 10, 2, 30, 0, 0, time.UTC),
			exists: false,
		},
		{
			name:   "wall clock repeated by fall back takes the first instant",
			zone:   "America/New_York",
			wall:   time.Date(2024, 11, 3, 1, 30, 0, 0, time.UTC),
			exists: true, at: time.Date(2024, 11, 3, 5, 30, 0, 0, time.UTC),
		},
		{
			name:   "half-hour spring forward skips a half hour",
			zone:   "Australia/Lord_Howe",
			wall:   time.Date(2024, 10, 6, 2, 15, 0, 0, time.UTC),
			exists: false,
		},
		{
			name:   "half-hour fall back takes the first instant",
			zone:   "Australia/Lord_Howe",
			wall:   time.Date(2024, 4, 7, 1, 45, 0, 0, time.UTC),
			exists: true, at: time.Date(2024, 4, 6, 14, 45, 0, 0, time.UTC),
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			at, ok := inLocation(c.wall, mustLoad(t, c.zone))
			if ok != c.exists {
				t.Fatalf("exists = %t, want %t (got %s)", ok, c.exists,
					at.Format("2006-01-02 15:04:05 MST"))
			}
			if !c.exists {
				return
			}
			if !at.Equal(c.at) {
				t.Errorf("resolved to %s, want %s",
					at.Format("2006-01-02 15:04:05 MST"), c.at.UTC().Format("2006-01-02 15:04:05 MST"))
			}
			// The instant must carry the wall clock it was asked for.
			if got, want := wallOf(at), c.wall; !got.Equal(want) {
				t.Errorf("carries wall clock %s, want %s",
					got.Format("2006-01-02 15:04"), want.Format("2006-01-02 15:04"))
			}
		})
	}
}

// TestNextMatchingAcrossTransitions sweeps every minute around an offset change
// and checks NextMatching against a brute-force walk over wall-clock minutes,
// which is the specification it has to meet. The zones are chosen for the shapes
// that break naive implementations: whole-hour DST, half-hour DST, one-off
// fifteen and thirty minute jumps, a transition at midnight, a skipped calendar
// day, and historical offsets carrying seconds.
func TestNextMatchingAcrossTransitions(t *testing.T) {
	for _, z := range []struct {
		zone  string
		what  string
		event time.Time
	}{
		{"America/New_York", "spring forward", time.Date(2024, 3, 10, 2, 0, 0, 0, time.UTC)},
		{"America/New_York", "fall back", time.Date(2024, 11, 3, 2, 0, 0, 0, time.UTC)},
		{"Europe/London", "spring forward", time.Date(2024, 3, 31, 1, 0, 0, 0, time.UTC)},
		{"Europe/London", "fall back", time.Date(2024, 10, 27, 2, 0, 0, 0, time.UTC)},
		{"Australia/Lord_Howe", "half-hour spring forward", time.Date(2024, 10, 6, 2, 0, 0, 0, time.UTC)},
		{"Australia/Lord_Howe", "half-hour fall back", time.Date(2024, 4, 7, 2, 0, 0, 0, time.UTC)},
		{"Pacific/Chatham", "quarter-hour offset zone", time.Date(2024, 9, 29, 2, 0, 0, 0, time.UTC)},
		{"Asia/Kathmandu", "one-off fifteen minute jump", time.Date(1986, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"America/Caracas", "one-off thirty minute jump", time.Date(2016, 5, 1, 2, 30, 0, 0, time.UTC)},
		{"America/Santiago", "transition at midnight", time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC)},
		{"Pacific/Apia", "skipped calendar day", time.Date(2011, 12, 30, 0, 0, 0, 0, time.UTC)},
		{"America/St_Johns", "offset carrying seconds", time.Date(1935, 4, 1, 0, 0, 0, 0, time.UTC)},
		{"Asia/Kolkata", "historical offset carrying seconds", time.Date(1906, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Asia/Tokyo", "no transitions at all", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
	} {
		for _, expr := range []string{
			"* * * * *", "0 * * * *", "*/10 * * * *", "30 1 * * *",
			"30 2 * * *", "0 0 * * *", "0 12 * * *", "15 2 * * 1",
		} {
			t.Run(z.zone+" "+z.what+" "+expr, func(t *testing.T) {
				loc := mustLoad(t, z.zone)
				s, err := Parse(expr)
				if err != nil {
					t.Fatalf("cannot parse %q: %v", expr, err)
				}

				event := z.event.In(loc)

				// Every wall-clock match in a wide window, in wall-clock order,
				// paired with the instant that realises it.
				type hit struct{ wall, at time.Time }
				var hits []hit
				for w, end := wallOf(event.AddDate(0, 0, -3)), wallOf(event.AddDate(0, 0, 3)); w.Before(end); w = w.Add(time.Minute) {
					if !s.Match(w) {
						continue
					}
					if at, ok := inLocation(w, loc); ok {
						hits = append(hits, hit{w, at})
					}
				}

				for from := event.Add(-18 * time.Hour); from.Before(event.Add(18 * time.Hour)); from = from.Add(time.Minute) {
					fromWall := wallOf(from)
					i := sort.Search(len(hits), func(i int) bool { return hits[i].wall.After(fromWall) })
					for i < len(hits) && !hits[i].at.After(from) {
						i++ // this wall clock's single firing already passed
					}
					if i == len(hits) {
						continue // no expectation available inside the window
					}

					want := hits[i].at
					got := s.NextMatching(from)

					switch {
					case got == nil:
						t.Fatalf("from %s: returned nil, want %s",
							from.Format("2006-01-02 15:04:05 MST"), want.Format("2006-01-02 15:04:05 MST"))
					case !got.Equal(want):
						t.Fatalf("from %s:\n     got %s\n    want %s",
							from.Format("2006-01-02 15:04:05 MST"),
							got.Format("2006-01-02 15:04:05 MST"),
							want.Format("2006-01-02 15:04:05 MST"))
					case !got.After(from):
						t.Fatalf("from %s: returned %s, which is not after it",
							from.Format("2006-01-02 15:04:05 MST"), got.Format("2006-01-02 15:04:05 MST"))
					case !s.Match(wallOf(*got)):
						t.Fatalf("from %s: returned %s, which the schedule does not match",
							from.Format("2006-01-02 15:04:05 MST"), got.Format("2006-01-02 15:04:05 MST"))
					}
				}
			})
		}
	}
}

// TestNextMatchingSkipsNonexistentWallClock pins the Quartz rule: a wall-clock
// time a transition skips does not fire that day.
func TestNextMatchingSkipsNonexistentWallClock(t *testing.T) {
	ny := mustLoad(t, "America/New_York")
	s, err := Parse("30 2 * * *")
	if err != nil {
		t.Fatalf("cannot parse: %v", err)
	}

	// 2024-03-10 02:30 does not exist in New York, so the next firing is the
	// following day rather than a shifted time on the 10th.
	got := s.NextMatching(time.Date(2024, 3, 9, 12, 0, 0, 0, ny))
	if got == nil {
		t.Fatal("returned nil")
	}
	if want := time.Date(2024, 3, 11, 2, 30, 0, 0, ny); !got.Equal(want) {
		t.Errorf("got %s, want %s",
			got.Format("2006-01-02 15:04:05 MST"), want.Format("2006-01-02 15:04:05 MST"))
	}
}

// TestNextMatchingFiresRepeatedWallClockOnce pins the other half of the rule: a
// wall-clock time that happens twice fires at the first occurrence only.
func TestNextMatchingFiresRepeatedWallClockOnce(t *testing.T) {
	ny := mustLoad(t, "America/New_York")
	s, err := Parse("30 1 * * *")
	if err != nil {
		t.Fatalf("cannot parse: %v", err)
	}

	first := s.NextMatching(time.Date(2024, 11, 3, 0, 0, 0, 0, ny))
	if first == nil {
		t.Fatal("returned nil")
	}
	// The earlier of the two 01:30s, still on eastern daylight time.
	if want := time.Date(2024, 11, 3, 5, 30, 0, 0, time.UTC); !first.Equal(want) {
		t.Fatalf("first firing %s, want %s",
			first.Format("2006-01-02 15:04:05 MST"), want.Format("2006-01-02 15:04:05 MST"))
	}

	// Asking again from there skips the repeat and moves to the next day.
	next := s.NextMatching(*first)
	if next == nil {
		t.Fatal("second call returned nil")
	}
	if want := time.Date(2024, 11, 4, 1, 30, 0, 0, ny); !next.Equal(want) {
		t.Errorf("second firing %s, want %s",
			next.Format("2006-01-02 15:04:05 MST"), want.Format("2006-01-02 15:04:05 MST"))
	}
}

// TestNextMatchingKeepsLocation checks the result comes back in the location it
// was asked about, whatever the search did internally.
func TestNextMatchingKeepsLocation(t *testing.T) {
	for _, zone := range []string{"America/New_York", "Australia/Lord_Howe", "UTC"} {
		loc := mustLoad(t, zone)
		s, err := Parse("0 12 * * *")
		if err != nil {
			t.Fatalf("cannot parse: %v", err)
		}

		got := s.NextMatching(time.Date(2024, 6, 1, 0, 0, 0, 0, loc))
		if got == nil {
			t.Fatalf("%s: returned nil", zone)
		}
		if got.Location() != loc {
			t.Errorf("%s: result is in %s", zone, got.Location())
		}
	}
}
