package cron

import (
	"fmt"
	"strings"
	"time"
)

// SchedulePattern is a single token within a cron field, such as a specific
// value, a range, or a step. Match reports whether the pattern matches t, and
// String renders the pattern back into its cron expression form.
type SchedulePattern interface {
	Match(t time.Time) bool
	String() string
}

/************
 * Schedule *
 ************/

// Schedule is a parsed cron expression. Each field holds the alternatives for
// one component of the expression; a nil field matches any value ("*").
type Schedule struct {
	Minute  Field
	Hour    Field
	Day     Field
	Month   Field
	Weekday Field
	Year    Field
}

// String renders the schedule back into a cron expression. The year field is
// written only when the schedule restricts it, since "*" is not accepted there.
func (s Schedule) String() string {
	if s.Year == nil {
		return fmt.Sprintf("%s %s %s %s %s",
			s.Minute,
			s.Hour,
			s.Day,
			s.Month,
			s.Weekday)
	}

	return fmt.Sprintf("%s %s %s %s %s %s",
		s.Minute,
		s.Hour,
		s.Day,
		s.Month,
		s.Weekday,
		s.Year)
}

// Match reports whether t satisfies every field of the schedule.
func (s Schedule) Match(time time.Time) bool {
	return s.MatchMinute(time) &&
		s.MatchHour(time) &&
		s.MatchDay(time) &&
		s.MatchMonth(time) &&
		s.MatchYear(time)
}

// MatchYear reports whether t's year matches the Year field, or true if the
// field is unset.
func (s Schedule) MatchYear(time time.Time) bool {
	if s.Year == nil {
		return true
	}

	return s.Year.Match(time)
}

// MatchMonth reports whether t's month matches the Month field, or true if the
// field is unset.
func (s Schedule) MatchMonth(time time.Time) bool {
	if s.Month == nil {
		return true
	}

	return s.Month.Match(time)
}

// MatchDay reports whether t matches the day-of-month and/or weekday fields.
// When both fields are restricted, a match on either one is sufficient,
// following the standard behavior of cron on most Unix-like systems.
func (s Schedule) MatchDay(time time.Time) bool {
	if len(s.Day) > 0 && len(s.Weekday) > 0 {
		// If both Day and Weekday are restricted, we require either Day or Weekday to match.
		// This is the standard behavior of cron in most Unix-like systems.
		return s.Day.Match(time) || s.Weekday.Match(time)
	} else if s.Day != nil {
		return s.Day.Match(time)
	} else if s.Weekday != nil {
		return s.Weekday.Match(time)
	} else {
		return true // No Day or Weekday specified, so always match
	}
}

// MatchHour reports whether t's hour matches the Hour field, or true if the
// field is unset.
func (s Schedule) MatchHour(time time.Time) bool {
	if s.Hour == nil {
		return true
	}

	return s.Hour.Match(time)
}

// MatchMinute reports whether t's minute matches the Minute field, or true if
// the field is unset.
func (s Schedule) MatchMinute(time time.Time) bool {
	if s.Minute == nil {
		return true
	}

	return s.Minute.Match(time)
}

func (s Schedule) nextMatchingYear(t *time.Time) bool {
	*t = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()) // Resetting lower parts of the time
	for *t = t.AddDate(1, 0, 0); t.Year() <= 2099; *t = t.AddDate(1, 0, 0) {
		if s.MatchYear(*t) {
			return true // Found a matching year
		}
	}
	return false // No matching year found
}

func (s Schedule) nextMatchingMonth(t *time.Time) bool {
	*t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()) // Resetting lower parts of the time
	year := t.Year()
	for *t = t.AddDate(0, 1, 0); t.Year() == year; *t = t.AddDate(0, 1, 0) {
		if s.MatchMonth(*t) {
			return true // Found a matching month
		}
	}
	return false // No matching month found
}

func (s Schedule) nextMatchingDay(t *time.Time) bool {
	*t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()) // Resetting lower parts of the time
	month := t.Month()
	for *t = t.AddDate(0, 0, 1); t.Month() == month; *t = t.AddDate(0, 0, 1) {
		if s.MatchDay(*t) {
			return true // Found a matching day
		}
	}
	return false // No matching day found
}

func (s Schedule) nextMatchingHour(t *time.Time) bool {
	*t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()) // Resetting lower parts of the time
	day := t.Day()
	for *t = t.Add(time.Hour); t.Day() == day; *t = t.Add(time.Hour) {
		if s.MatchHour(*t) {
			return true // Found a matching hour
		}
	}
	return false // No matching hour found
}

func (s Schedule) nextMatchingMinute(t *time.Time) bool {
	hour := t.Hour()
	for *t = t.Add(time.Minute); t.Hour() == hour; *t = t.Add(time.Minute) {
		if s.MatchMinute(*t) {
			return true // Found a matching minute
		}
	}
	return false // No matching minute found
}

// NextMatching returns the earliest time strictly after relativeTo that the
// schedule matches, at minute granularity, or nil if there is none before the
// year 2100. The result is in relativeTo's location.
//
// A cron expression names wall-clock times, so each matching wall-clock time
// fires at most once. Following Quartz, a wall-clock time that a daylight-saving
// transition skips never matches: a schedule for 02:30 does not run on a day
// where 02:30 does not exist. A wall-clock time that occurs twice matches only
// at its first occurrence.
func (s Schedule) NextMatching(relativeTo time.Time) *time.Time {
	loc := relativeTo.Location()

	// The search runs in UTC, where the offset never changes, so every step
	// below is plain wall-clock arithmetic and cannot be knocked off a minute or
	// hour boundary by a transition. Only the answer is resolved back into loc.
	y, mo, d := relativeTo.Date()
	t := time.Date(y, mo, d, relativeTo.Hour(), relativeTo.Minute(), 0, 0, time.UTC).
		Add(time.Minute) // Start from the next minute

	for t.Year() <= 2099 {
		if !s.MatchYear(t) && !s.nextMatchingYear(&t) {
			continue // Since there is no matching year, the year is already at 2100, so this will eventually exit the loop
		}

		if !s.MatchMonth(t) && !s.nextMatchingMonth(&t) {
			continue // At this point, `nextMatchingMonth` already rolled the year over to the next one, so we need to check everything again
		}

		if !s.MatchDay(t) && !s.nextMatchingDay(&t) {
			continue // At this point, `nextMatchingDay` already rolled the month over to the next one, so we need to check everything again
		}

		if !s.MatchHour(t) && !s.nextMatchingHour(&t) {
			continue // At this point, `nextMatchingHour` already rolled the day over to the next one, so we need to check everything again
		}

		if !s.MatchMinute(t) && !s.nextMatchingMinute(&t) {
			continue // At this point, `nextMatchingMinute` already rolled the hour over to the next one, so we need to check everything again
		}

		if at, ok := inLocation(t, loc); ok && at.After(relativeTo) {
			return &at
		}

		// Either loc skips this wall-clock time, or it resolves to an instant
		// that already passed — which happens inside a repeated hour, where the
		// single firing for this wall-clock time is behind us. Try the next one.
		t = t.Add(time.Minute)
	}

	return nil // No match found
}

// inLocation resolves a wall-clock time, held as a UTC time, to the earliest
// instant in loc carrying exactly that wall clock. It reports false when loc
// skips the wall-clock time, as happens across a daylight-saving transition.
func inLocation(wall time.Time, loc *time.Location) (time.Time, bool) {
	y, mo, d := wall.Date()
	h, mi := wall.Hour(), wall.Minute()

	carries := func(t time.Time) bool {
		ty, tmo, td := t.Date()
		return ty == y && tmo == mo && td == d && t.Hour() == h && t.Minute() == mi
	}

	// time.Date silently shifts a wall-clock time that loc skips, so a result
	// that reads back differently means the time does not exist there.
	at := time.Date(y, mo, d, h, mi, 0, 0, loc)
	if !carries(at) {
		return time.Time{}, false
	}

	// A repeated wall-clock time has an instant per offset in effect around it,
	// and time.Date does not promise which one it picks. An instant carries this
	// wall clock exactly when it equals wall minus that instant's offset, so the
	// offsets on either side yield every candidate.
	for _, probe := range []time.Time{at.Add(-3 * time.Hour), at.Add(3 * time.Hour)} {
		_, offset := probe.Zone()
		if cand := time.Unix(wall.Unix()-int64(offset), 0).In(loc); carries(cand) && cand.Before(at) {
			at = cand
		}
	}

	return at, true
}

// Field is one component of a Schedule (minute, hour, day, and so on). It holds
// a set of alternatives and matches when any one of them matches. A nil Field
// matches any value ("*").
type Field []SchedulePattern

// String renders the field as a comma-separated cron expression, or "*" if the
// field is nil.
func (f Field) String() string {
	if f == nil {
		return "*"
	}

	var r []string
	for _, e := range f {
		r = append(r, e.String())
	}

	return strings.Join(r, ",")
}

// Match reports whether any alternative in the field matches t, or true if the
// field is nil.
func (f Field) Match(t time.Time) bool {
	if f == nil {
		return true
	}

	for _, e := range f {
		if e.Match(t) {
			return true
		}
	}

	return false
}

/**********
 * Minute *
 **********/

// Single

// Minute matches a specific minute of the hour (0–59), e.g. "5".
type Minute int

// String returns the cron representation of the minute.
func (m Minute) String() string {
	return fmt.Sprintf("%d", m)
}

// Match reports whether t falls in the given minute.
func (m Minute) Match(t time.Time) bool {
	return int(m) == t.Minute()
}

// Range

// MinuteRange matches an inclusive range of minutes, e.g. "5-10".
type MinuteRange struct {
	From int
	To   int
}

// String returns the cron representation of the minute range.
func (m MinuteRange) String() string {
	return fmt.Sprintf("%d-%d", m.From, m.To)
}

// Match reports whether t's minute lies within the range.
func (m MinuteRange) Match(t time.Time) bool {
	min := t.Minute()
	return m.From <= min && min <= m.To
}

// Every N

// MinuteEvery matches every Nth minute from the top of the hour, e.g. "*/15".
type MinuteEvery int

// String returns the cron representation of the minute step.
func (m MinuteEvery) String() string {
	return fmt.Sprintf("*/%d", m)
}

// Match reports whether t's minute is a multiple of the step.
func (m MinuteEvery) Match(t time.Time) bool {
	return t.Minute()%int(m) == 0
}

// Every N starting at

// MinuteEveryFrom matches every Nth minute starting at a given minute, e.g.
// "5/15".
type MinuteEveryFrom struct {
	From  int
	Every int
}

// String returns the cron representation of the minute step.
func (m MinuteEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", m.From, m.Every)
}

// Match reports whether t's minute is at or after the start and on the step.
func (m MinuteEveryFrom) Match(t time.Time) bool {
	min := t.Minute()
	return m.From <= min && (min-m.From)%m.Every == 0
}

// Every N in range

// MinuteEveryInRange matches every Nth minute within an inclusive range, e.g.
// "5-50/15".
type MinuteEveryInRange struct {
	From  int
	To    int
	Every int
}

// String returns the cron representation of the minute step within a range.
func (m MinuteEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", m.From, m.To, m.Every)
}

// Match reports whether t's minute lies within the range and on the step.
func (m MinuteEveryInRange) Match(t time.Time) bool {
	min := t.Minute()
	return m.From <= min && min <= m.To && (min-m.From)%m.Every == 0
}

/********
 * Hour *
 ********/

// Single

// Hour matches a specific hour of the day (0–23), e.g. "9".
type Hour int

// String returns the cron representation of the hour.
func (h Hour) String() string {
	return fmt.Sprintf("%d", h)
}

// Match reports whether t falls in the given hour.
func (h Hour) Match(t time.Time) bool {
	return int(h) == t.Hour()
}

// Range

// HourRange matches an inclusive range of hours, e.g. "8-10".
type HourRange struct {
	From int
	To   int
}

// String returns the cron representation of the hour range.
func (h HourRange) String() string {
	return fmt.Sprintf("%d-%d", h.From, h.To)
}

// Match reports whether t's hour lies within the range.
func (h HourRange) Match(t time.Time) bool {
	hour := t.Hour()
	return h.From <= hour && hour <= h.To
}

// Every N

// HourEvery matches every Nth hour from midnight, e.g. "*/3".
type HourEvery int

// String returns the cron representation of the hour step.
func (h HourEvery) String() string {
	return fmt.Sprintf("*/%d", h)
}

// Match reports whether t's hour is a multiple of the step.
func (h HourEvery) Match(t time.Time) bool {
	return t.Hour()%int(h) == 0
}

// Every N starting at

// HourEveryFrom matches every Nth hour starting at a given hour, e.g. "2/3".
type HourEveryFrom struct {
	From  int
	Every int
}

// String returns the cron representation of the hour step.
func (h HourEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", h.From, h.Every)
}

// Match reports whether t's hour is at or after the start and on the step.
func (h HourEveryFrom) Match(t time.Time) bool {
	hour := t.Hour()
	return h.From <= hour && (hour-h.From)%h.Every == 0
}

// Every N in range

// HourEveryInRange matches every Nth hour within an inclusive range, e.g.
// "9-17/2".
type HourEveryInRange struct {
	From  int
	To    int
	Every int
}

// String returns the cron representation of the hour step within a range.
func (h HourEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", h.From, h.To, h.Every)
}

// Match reports whether t's hour lies within the range and on the step.
func (h HourEveryInRange) Match(t time.Time) bool {
	hour := t.Hour()
	return h.From <= hour && hour <= h.To && (hour-h.From)%h.Every == 0
}

/*******
 * Day *
 *******/

// Single

// Day matches a specific day of the month (1–31), e.g. "15".
type Day int

// String returns the cron representation of the day.
func (d Day) String() string {
	return fmt.Sprintf("%d", d)
}

// Match reports whether t falls on the given day of the month.
func (d Day) Match(t time.Time) bool {
	return int(d) == t.Day()
}

// Weekday

// DayOnWeekday matches the nearest weekday (Mon–Fri) to a given day of the
// month without crossing into an adjacent month, e.g. "15W".
type DayOnWeekday int

// String returns the cron representation of the nearest-weekday rule.
func (d DayOnWeekday) String() string {
	return fmt.Sprintf("%dW", d)
}

// Match reports whether t is the nearest weekday to the configured day.
func (d DayOnWeekday) Match(t time.Time) bool {
	rt := time.Date(t.Year(), t.Month(), int(d), 0, 0, 0, 0, t.Location())
	if rt.Month() != t.Month() {
		rt = rt.AddDate(0, 0, -rt.Day())
	}

	switch rt.Weekday() {
	case time.Saturday:
		if rt.Day() == 1 {
			rt = rt.AddDate(0, 0, 2)
		} else {
			rt = rt.AddDate(0, 0, -1)
		}
	case time.Sunday:
		if rt.AddDate(0, 0, 1).Month() != rt.Month() {
			rt = rt.AddDate(0, 0, -2)
		} else {
			rt = rt.AddDate(0, 0, 1)
		}
	}

	return rt.Day() == t.Day()
}

// Last

// DayLast matches the last day of the month, written "L".
type DayLast struct{}

// String returns the cron representation of the last-day rule.
func (d DayLast) String() string {
	return "L"
}

// Match reports whether t is the last day of its month.
func (d DayLast) Match(t time.Time) bool {
	return t.Month() != t.AddDate(0, 0, 1).Month()
}

// Last weekday

// DayLastWeekday matches the last weekday (Mon–Fri) of the month, written "LW".
type DayLastWeekday struct{}

// String returns the cron representation of the last-weekday rule.
func (d DayLastWeekday) String() string {
	return "LW"
}

// Match reports whether t is the last weekday of its month.
func (d DayLastWeekday) Match(t time.Time) bool {
	rt := time.Date(t.Year(), t.Month(), 31, 0, 0, 0, 0, t.Location())
	if rt.Month() != t.Month() {
		rt = rt.AddDate(0, 0, -rt.Day())
	}

	switch rt.Weekday() {
	case time.Saturday:
		rt = rt.AddDate(0, 0, -1)
	case time.Sunday:
		rt = rt.AddDate(0, 0, -2)
	}

	return rt.Day() == t.Day()
}

// Range

// DayRange matches an inclusive range of days of the month, e.g. "5-10".
type DayRange struct {
	From int
	To   int
}

// String returns the cron representation of the day range.
func (d DayRange) String() string {
	return fmt.Sprintf("%d-%d", d.From, d.To)
}

// Match reports whether t's day lies within the range, wrapping past month-end
// when From is greater than To.
func (d DayRange) Match(t time.Time) bool {
	day := t.Day()
	if d.From <= d.To {
		return d.From <= day && day <= d.To
	} else {
		return d.From <= day || day <= d.To
	}
}

// Every N

// DayEvery matches every Nth day of the month counting from the 1st, e.g.
// "*/2" matches the 1st, 3rd, 5th and so on.
type DayEvery int

// String returns the cron representation of the day step.
func (d DayEvery) String() string {
	return fmt.Sprintf("*/%d", d)
}

// Match reports whether t's day is on the step, counting from the 1st.
func (d DayEvery) Match(t time.Time) bool {
	return (t.Day()-1)%int(d) == 0
}

// Every N starting at

// DayEveryFrom matches every Nth day starting at a given day, e.g. "1/5".
type DayEveryFrom struct {
	From  int
	Every int
}

// String returns the cron representation of the day step.
func (d DayEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", d.From, d.Every)
}

// Match reports whether t's day is at or after the start and on the step.
func (d DayEveryFrom) Match(t time.Time) bool {
	day := t.Day()
	return d.From <= day && (day-d.From)%d.Every == 0
}

// Every N in range

// DayEveryInRange matches every Nth day within an inclusive range, e.g.
// "5-20/5".
type DayEveryInRange struct {
	From  int
	To    int
	Every int
}

// String returns the cron representation of the day step within a range.
func (d DayEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", d.From, d.To, d.Every)
}

// Match reports whether t's day lies within the range and on the step.
func (d DayEveryInRange) Match(t time.Time) bool {
	day := t.Day()
	return d.From <= day && day <= d.To && (day-d.From)%d.Every == 0
}

/*********
 * Month *
 *********/

var monthToStr = map[time.Month]string{1: "JAN", 2: "FEB", 3: "MAR", 4: "APR", 5: "MAY", 6: "JUN",
	7: "JUL", 8: "AUG", 9: "SEP", 10: "OCT", 11: "NOV", 12: "DEC"}

// Single

// Month matches a specific month (1–12 or JAN–DEC). Numeric controls whether
// String renders the value as a number or a three-letter name.
type Month struct {
	At      time.Month
	Numeric bool
}

// String returns the cron representation of the month.
func (m Month) String() string {
	if m.Numeric {
		return fmt.Sprintf("%d", m.At)
	} else {
		return monthToStr[m.At]
	}
}

// Match reports whether t falls in the given month.
func (m Month) Match(t time.Time) bool {
	return m.At == t.Month()
}

// Range

// MonthRange matches an inclusive range of months, e.g. "1-5" or "JAN-MAY".
// Numeric controls whether String renders the bounds as numbers or names.
type MonthRange struct {
	From    time.Month
	To      time.Month
	Numeric bool
}

// String returns the cron representation of the month range.
func (m MonthRange) String() string {
	if m.Numeric {
		return fmt.Sprintf("%d-%d", m.From, m.To)
	} else {
		return fmt.Sprintf("%s-%s", monthToStr[m.From], monthToStr[m.To])
	}
}

// Match reports whether t's month lies within the range.
func (m MonthRange) Match(t time.Time) bool {
	am := t.Month()
	return m.From <= am && am <= m.To
}

// Every N

// MonthEvery matches every Nth month counting from January, e.g. "*/2" matches
// January, March, May and so on.
type MonthEvery int

// String returns the cron representation of the month step.
func (m MonthEvery) String() string {
	return fmt.Sprintf("*/%d", m)
}

// Match reports whether t's month is on the step, counting from January.
func (m MonthEvery) Match(t time.Time) bool {
	return (int(t.Month())-1)%int(m) == 0
}

// Every N starting at

// MonthEveryFrom matches every Nth month starting at a given month, e.g. "3/2".
type MonthEveryFrom struct {
	From  time.Month
	Every int
}

// String returns the cron representation of the month step.
func (m MonthEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", m.From, m.Every)
}

// Match reports whether t's month is at or after the start and on the step.
func (m MonthEveryFrom) Match(t time.Time) bool {
	am := int(t.Month())
	return int(m.From) <= am && (am-int(m.From))%m.Every == 0
}

// Every N in range

// MonthEveryInRange matches every Nth month within an inclusive range, e.g.
// "2-6/2".
type MonthEveryInRange struct {
	From  time.Month
	To    time.Month
	Every int
}

// String returns the cron representation of the month step within a range.
func (m MonthEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", m.From, m.To, m.Every)
}

// Match reports whether t's month lies within the range and on the step.
func (m MonthEveryInRange) Match(t time.Time) bool {
	am := int(t.Month())
	return int(m.From) <= am && am <= int(m.To) && (am-int(m.From))%m.Every == 0
}

/***********
 * Weekday *
 ***********/

var weekdayToStr = [...]string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}

// Single

// Weekday matches a specific day of the week (0–6 or SUN–SAT). Numeric controls
// whether String renders the value as a number or a three-letter name.
type Weekday struct {
	At      time.Weekday
	Numeric bool
}

// String returns the cron representation of the weekday.
func (w Weekday) String() string {
	if w.Numeric {
		return fmt.Sprintf("%d", w.At)
	} else {
		return weekdayToStr[w.At]
	}
}

// Match reports whether t falls on the given weekday.
func (w Weekday) Match(t time.Time) bool {
	return w.At == t.Weekday()
}

// Last

// WeekdayLast matches the last occurrence of a weekday in the month, e.g.
// "MONL". Numeric controls whether String renders the value as a number or a
// name.
type WeekdayLast struct {
	At      time.Weekday
	Numeric bool
}

// String returns the cron representation of the last-weekday rule.
func (w WeekdayLast) String() string {
	if w.Numeric {
		return fmt.Sprintf("%dL", w.At)
	} else {
		return fmt.Sprintf("%sL", weekdayToStr[w.At])
	}
}

// Match reports whether t is the last occurrence of the weekday in its month.
func (w WeekdayLast) Match(t time.Time) bool {
	return w.At == t.Weekday() &&
		t.AddDate(0, 0, 7).Month() != t.Month()
}

// #th

// WeekdayNth matches the Nth occurrence of a weekday in the month, e.g. "MON#2".
// Numeric controls whether String renders the weekday as a number or a name.
type WeekdayNth struct {
	At      time.Weekday
	Nth     int
	Numeric bool
}

// String returns the cron representation of the Nth-weekday rule.
func (w WeekdayNth) String() string {
	if w.Numeric {
		return fmt.Sprintf("%d#%d", w.At, w.Nth)
	} else {
		return fmt.Sprintf("%s#%d", weekdayToStr[w.At], w.Nth)
	}
}

// Match reports whether t is the Nth occurrence of the weekday in its month.
func (w WeekdayNth) Match(t time.Time) bool {
	return w.At == t.Weekday() &&
		t.AddDate(0, 0, -w.Nth*7).Month() != t.Month() &&
		t.AddDate(0, 0, -(w.Nth-1)*7).Month() == t.Month()
}

// Range

// WeekdayRange matches an inclusive range of weekdays, e.g. "2-5" or "TUE-FRI".
// Numeric controls whether String renders the bounds as numbers or names.
type WeekdayRange struct {
	From    time.Weekday
	To      time.Weekday
	Numeric bool
}

// String returns the cron representation of the weekday range.
func (w WeekdayRange) String() string {
	if w.Numeric {
		return fmt.Sprintf("%d-%d", w.From, w.To)
	} else {
		return fmt.Sprintf("%s-%s", weekdayToStr[w.From], weekdayToStr[w.To])
	}
}

// Match reports whether t's weekday lies within the range, wrapping past the
// end of the week when From is greater than To.
func (w WeekdayRange) Match(t time.Time) bool {
	aw := t.Weekday()
	if w.From <= w.To {
		return w.From <= aw && aw <= w.To
	} else {
		return w.From <= aw || aw <= w.To
	}
}

// Every N

// WeekdayEvery matches every Nth weekday starting from Sunday, e.g. "*/2".
type WeekdayEvery int

// String returns the cron representation of the weekday step.
func (w WeekdayEvery) String() string {
	return fmt.Sprintf("*/%d", w)
}

// Match reports whether t's weekday is a multiple of the step.
func (w WeekdayEvery) Match(t time.Time) bool {
	return int(t.Weekday())%int(w) == 0
}

// Every N starting at

// WeekdayEveryFrom matches every Nth weekday starting at a given weekday, e.g.
// "1/2".
type WeekdayEveryFrom struct {
	From  time.Weekday
	Every int
}

// String returns the cron representation of the weekday step.
func (w WeekdayEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", w.From, w.Every)
}

// Match reports whether t's weekday is at or after the start and on the step.
func (w WeekdayEveryFrom) Match(t time.Time) bool {
	aw := int(t.Weekday())
	return int(w.From) <= aw && (aw-int(w.From))%w.Every == 0
}

// Every N in range

// WeekdayEveryInRange matches every Nth weekday within an inclusive range, e.g.
// "1-5/2".
type WeekdayEveryInRange struct {
	From  time.Weekday
	To    time.Weekday
	Every int
}

// String returns the cron representation of the weekday step within a range.
func (w WeekdayEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", w.From, w.To, w.Every)
}

// Match reports whether t's weekday lies within the range and on the step.
func (w WeekdayEveryInRange) Match(t time.Time) bool {
	aw := int(t.Weekday())
	return int(w.From) <= aw && aw <= int(w.To) && (aw-int(w.From))%w.Every == 0
}

/********
 * Year *
 ********/

// Single

// Year matches a specific year (1900–2099), e.g. "2018".
type Year int

// String returns the cron representation of the year.
func (m Year) String() string {
	return fmt.Sprintf("%d", m)
}

// Match reports whether t falls in the given year.
func (m Year) Match(t time.Time) bool {
	return int(m) == t.Year()
}

// Range

// YearRange matches an inclusive range of years, e.g. "1978-2000".
type YearRange struct {
	From int
	To   int
}

// String returns the cron representation of the year range.
func (m YearRange) String() string {
	return fmt.Sprintf("%d-%d", m.From, m.To)
}

// Match reports whether t's year lies within the range.
func (m YearRange) Match(t time.Time) bool {
	y := t.Year()
	return m.From <= y && y <= m.To
}

// Every N

// YearEvery matches every Nth year counting from 1900, e.g. "*/4" matches 1900,
// 1904, 1908 and so on.
type YearEvery int

// String returns the cron representation of the year step.
func (y YearEvery) String() string {
	return fmt.Sprintf("*/%d", y)
}

// Match reports whether t's year is on the step. The step counts from 1900, the
// start of the year field's range, so "*/N" is identical to "1900-2099/N".
func (y YearEvery) Match(t time.Time) bool {
	return (t.Year()-1900)%int(y) == 0
}

// Every N starting at

// YearEveryFrom matches every Nth year starting at a given year, e.g. "2000/3".
type YearEveryFrom struct {
	From  int
	Every int
}

// String returns the cron representation of the year step.
func (y YearEveryFrom) String() string {
	return fmt.Sprintf("%d/%d", y.From, y.Every)
}

// Match reports whether t's year is at or after the start and on the step.
func (y YearEveryFrom) Match(t time.Time) bool {
	year := t.Year()
	return y.From <= year && (year-y.From)%y.Every == 0
}

// Every N in range

// YearEveryInRange matches every Nth year within an inclusive range, e.g.
// "1978-2000/2".
type YearEveryInRange struct {
	From  int
	To    int
	Every int
}

// String returns the cron representation of the year step within a range.
func (y YearEveryInRange) String() string {
	return fmt.Sprintf("%d-%d/%d", y.From, y.To, y.Every)
}

// Match reports whether t's year lies within the range and on the step.
func (y YearEveryInRange) Match(t time.Time) bool {
	year := t.Year()
	return y.From <= year && year <= y.To && (year-y.From)%y.Every == 0
}
