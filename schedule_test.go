package cron

import (
	"regexp"
	"testing"
	"time"
)

var fieldExtractor = regexp.MustCompile("^Test([A-Z][a-z]*)")

func tester(t *testing.T, data [5]int, p SchedulePattern, expected bool) {
	if r := time.Date(data[0], time.Month(data[1]), data[2], data[3], data[4], 0, 0, time.Local); p.Match(r) != expected {
		matches := fieldExtractor.FindStringSubmatch(t.Name())
		t.Errorf("%s value should %s pattern: \"%s\"\nValue: %s",
			matches[1],
			map[bool]string{
				true:  "match",
				false: "not match",
			}[expected],
			p,
			r.Format("2006-01-02 15:04"))
	}
}

/**********
 * Minute *
 **********/

func TestMinute_String(t *testing.T) {
	m := Minute(2)

	if expected := "2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMinute_Match(t *testing.T) {
	m := Minute(5)

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 5},
	} {
		tester(t, v, m, true)
	}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 4},
	} {
		tester(t, v, m, false)
	}
}

func TestMinuteRange_String(t *testing.T) {
	m := MinuteRange{5, 10}

	if expected := "5-10"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMinuteRange_Match(t *testing.T) {
	m := MinuteRange{5, 10}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 8},
	} {
		tester(t, v, m, true)
	}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 4},
	} {
		tester(t, v, m, false)
	}
}

func TestMinuteEvery_String(t *testing.T) {
	m := MinuteEvery(2)

	if expected := "*/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMinuteEvery_Match(t *testing.T) {
	m := MinuteEvery(5)

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 5},
		{2000, 1, 1, 0, 10},
	} {
		tester(t, v, m, true)
	}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 4},
		{2000, 1, 1, 0, 8},
	} {
		tester(t, v, m, false)
	}
}

func TestMinuteEveryFrom_String(t *testing.T) {
	m := MinuteEveryFrom{1, 5}

	if expected := "1/5"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMinuteEveryFrom_Match(t *testing.T) {
	m := MinuteEveryFrom{6, 5}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 6},
		{2000, 1, 1, 0, 11},
	} {
		tester(t, v, m, true)
	}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 1},
		{2000, 1, 1, 0, 10},
	} {
		tester(t, v, m, false)
	}
}

func TestMinuteEveryInRange_String(t *testing.T) {
	m := MinuteEveryInRange{1, 30, 5}

	if expected := "1-30/5"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMinuteEveryInRange_Match(t *testing.T) {
	m := MinuteEveryInRange{6, 20, 5}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 6},
		{2000, 1, 1, 0, 11},
		{2000, 1, 1, 0, 16},
	} {
		tester(t, v, m, true)
	}

	for _, v := range [][5]int{
		{2000, 1, 1, 0, 21},
		{2000, 1, 1, 0, 1},
		{2000, 1, 1, 0, 10},
	} {
		tester(t, v, m, false)
	}
}

/********
 * Hour *
 ********/

func TestHour_String(t *testing.T) {
	h := Hour(2)

	if expected := "2"; h.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, h.String())
	}
}

func TestHour_Match(t *testing.T) {
	h := Hour(5)

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 5, 0},
		},
		false: {
			{2000, 1, 1, 4, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, h, expected)
		}
	}
}

func TestHourRange_String(t *testing.T) {
	h := HourRange{5, 10}

	if expected := "5-10"; h.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, h.String())
	}
}

func TestHourRange_Match(t *testing.T) {
	h := HourRange{5, 10}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 8, 0},
		},
		false: {
			{2000, 1, 1, 4, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, h, expected)
		}
	}
}

func TestHourEvery_String(t *testing.T) {
	h := HourEvery(2)

	if expected := "*/2"; h.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, h.String())
	}
}

func TestHourEvery_Match(t *testing.T) {
	h := HourEvery(5)

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 5, 0},
			{2000, 1, 1, 10, 0},
		},
		false: {
			{2000, 1, 1, 4},
			{2000, 1, 1, 8, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, h, expected)
		}
	}
}

func TestHourEveryFrom_String(t *testing.T) {
	h := HourEveryFrom{1, 5}

	if expected := "1/5"; h.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, h.String())
	}
}

func TestHourEveryFrom_Match(t *testing.T) {
	h := HourEveryFrom{6, 5}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 6, 0},
			{2000, 1, 1, 11, 0},
		},
		false: {
			{2000, 1, 1, 1, 0},
			{2000, 1, 1, 10, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, h, expected)
		}
	}
}

func TestHourEveryInRange_String(t *testing.T) {
	h := HourEveryInRange{4, 20, 5}

	if expected := "4-20/5"; h.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, h.String())
	}
}

func TestHourEveryInRange_Match(t *testing.T) {
	h := HourEveryInRange{6, 20, 5}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 6, 0},
			{2000, 1, 1, 11, 0},
			{2000, 1, 1, 16, 0},
		},
		false: {
			{2000, 1, 1, 21, 0},
			{2000, 1, 1, 1, 0},
			{2000, 1, 1, 10, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, h, expected)
		}
	}
}

/*******
 * Day *
 *******/

func TestDay_String(t *testing.T) {
	d := Day(2)

	if expected := "2"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDay_Match(t *testing.T) {
	d := Day(5)

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 5, 0, 0},
		},
		false: {
			{2000, 1, 4, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayOnWeekday_String(t *testing.T) {
	d := DayOnWeekday(2)

	if expected := "2W"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayOnWeekday_Match(t *testing.T) {
	// Saturday
	d := DayOnWeekday(10)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 9, 0, 0},
		},
		false: {
			{2018, 11, 10, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}

	// Sunday
	d = DayOnWeekday(11)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 12, 0, 0},
		},
		false: {
			{2018, 11, 11, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}

	// Saturday on 1st
	d = DayOnWeekday(1)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 12, 3, 0, 0},
		},
		false: {
			{2018, 12, 1, 0, 0},
			{2018, 12, 2, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}

	// Sunday on 2nd
	d = DayOnWeekday(2)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 12, 3, 0, 0},
		},
		false: {
			{2018, 12, 1, 0, 0},
			{2018, 12, 2, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}

	// Sunday on last day of month
	d = DayOnWeekday(30)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 9, 28, 0, 0},
		},
		false: {
			{2018, 9, 29, 0, 0},
			{2018, 9, 30, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}

	// Saturday on the day before the last day of month
	d = DayOnWeekday(29)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 9, 28, 0, 0},
		},
		false: {
			{2018, 9, 29, 0, 0},
			{2018, 9, 30, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayLast_String(t *testing.T) {
	d := DayLast{}

	if expected := "L"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayLast_Match(t *testing.T) {
	d := DayLast{}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 30, 0, 0},
		},
		false: {
			{2018, 11, 29, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayLastWeekday_String(t *testing.T) {
	d := DayLastWeekday{}

	if expected := "LW"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayLastWeekday_Match(t *testing.T) {
	d := DayLastWeekday{}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 8, 31, 0, 0},
			{2018, 9, 28, 0, 0},
			{2016, 2, 29, 0, 0},
			{2015, 2, 27, 0, 0},
		},
		false: {
			{2018, 9, 29, 0, 0},
			{2016, 2, 28, 0, 0},
			{2015, 2, 28, 0, 0},
			{2018, 9, 30, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayRange_String(t *testing.T) {
	d := DayRange{5, 10}

	if expected := "5-10"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayRange_Match(t *testing.T) {
	d := DayRange{5, 10}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 8, 0, 0},
		},
		false: {
			{2000, 1, 4, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayEvery_String(t *testing.T) {
	d := DayEvery(2)

	if expected := "*/2"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayEvery_Match(t *testing.T) {
	d := DayEvery(5)

	// "*/5" is "1-31/5": the 1st, 6th, 11th, ... — not the days divisible by 5.
	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 0, 0},
			{2000, 1, 6, 0, 0},
			{2000, 1, 11, 0, 0},
			{2000, 1, 31, 0, 0},
		},
		false: {
			{2000, 1, 2, 0, 0},
			{2000, 1, 4, 0, 0},
			{2000, 1, 5, 0, 0},
			{2000, 1, 10, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayEveryFrom_String(t *testing.T) {
	d := DayEveryFrom{1, 5}

	if expected := "1/5"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayEveryFrom_Match(t *testing.T) {
	d := DayEveryFrom{6, 5}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 6, 0, 0},
			{2000, 1, 11, 0, 0},
		},
		false: {
			{2000, 1, 1, 0, 0},
			{2000, 1, 10, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

func TestDayEveryInRange_String(t *testing.T) {
	d := DayEveryInRange{4, 20, 5}

	if expected := "4-20/5"; d.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, d.String())
	}
}

func TestDayEveryInRange_Match(t *testing.T) {
	d := DayEveryInRange{6, 20, 5}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 6, 0, 0},
			{2000, 1, 11, 0, 0},
			{2000, 1, 16, 0, 0},
		},
		false: {
			{2000, 1, 21, 0, 0},
			{2000, 1, 1, 0, 0},
			{2000, 1, 10, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, d, expected)
		}
	}
}

/*********
 * Month *
 *********/

func TestMonth_String(t *testing.T) {
	m := Month{time.January, true}

	if expected := "1"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}

	for i, expected := range []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"} {
		m = Month{time.Month(i + 1), false}

		if m.String() != expected {
			t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
		}
	}
}

func TestMonth_Match(t *testing.T) {
	m := Month{time.January, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 0, 0},
		},
		false: {
			{2000, 2, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, m, expected)
		}
	}
}

func TestMonthRange_String(t *testing.T) {
	m := MonthRange{time.January, time.March, true}

	if expected := "1-3"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}

	m = MonthRange{time.January, time.March, false}

	if expected := "JAN-MAR"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMonthRange_Match(t *testing.T) {
	m := MonthRange{time.January, time.March, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 0, 0},
		},
		false: {
			{2000, 5, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, m, expected)
		}
	}
}

func TestMonthEvery_String(t *testing.T) {
	m := MonthEvery(2)

	if expected := "*/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMonthEvery_Match(t *testing.T) {
	m := MonthEvery(5)

	// "*/5" is "1-12/5": January, June, November — not the months divisible by 5.
	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 1, 1, 0, 0},
			{2000, 6, 1, 0, 0},
			{2000, 11, 1, 0, 0},
		},
		false: {
			{2000, 2, 1, 0, 0},
			{2000, 4, 1, 0, 0},
			{2000, 5, 1, 0, 0},
			{2000, 10, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, m, expected)
		}
	}
}

func TestMonthEveryFrom_String(t *testing.T) {
	m := MonthEveryFrom{2, 5}

	if expected := "2/5"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMonthEveryFrom_Match(t *testing.T) {
	m := MonthEveryFrom{2, 5}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 7, 1, 0, 0},
			{2000, 12, 1, 0, 0},
		},
		false: {
			{2000, 1, 1, 0, 0},
			{2000, 10, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, m, expected)
		}
	}
}

func TestMonthEveryInRange_String(t *testing.T) {
	m := MonthEveryInRange{4, 8, 2}

	if expected := "4-8/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestMonthEveryInRange_Match(t *testing.T) {
	m := MonthEveryInRange{4, 8, 2}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 4, 1, 0, 0},
			{2000, 6, 1, 0, 0},
			{2000, 8, 1, 0, 0},
		},
		false: {
			{2000, 1, 1, 0, 0},
			{2000, 9, 1, 0, 0},
			{2000, 11, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, m, expected)
		}
	}
}

/***********
 * Weekday *
 ***********/

func TestWeekday_String(t *testing.T) {
	w := Weekday{time.Monday, true}

	if expected := "1"; w.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
	}

	for i, expected := range []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"} {
		w = Weekday{time.Weekday(i), false}

		if w.String() != expected {
			t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
		}
	}
}

func TestWeekday_Match(t *testing.T) {
	w := Weekday{time.Monday, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 12, 0, 0},
		},
		false: {
			{2018, 11, 11, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayLast_String(t *testing.T) {
	w := WeekdayLast{time.Monday, true}

	if expected := "1L"; w.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
	}

	for i, e := range []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"} {
		w = WeekdayLast{time.Weekday(i), false}

		if expected := e + "L"; w.String() != expected {
			t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
		}
	}
}

func TestWeekdayLast_Match(t *testing.T) {
	w := WeekdayLast{time.Monday, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 26, 0, 0},
		},
		false: {
			{2018, 11, 19, 0, 0},
			{2018, 11, 12, 0, 0},
			{2018, 11, 5, 0, 0},
			{2018, 11, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayNth_String(t *testing.T) {
	w := WeekdayNth{time.Monday, 2, true}

	if expected := "1#2"; w.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
	}

	for i, e := range []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"} {
		w = WeekdayNth{time.Weekday(i), 2, false}

		if expected := e + "#2"; w.String() != expected {
			t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
		}
	}
}

func TestWeekdayNth_Match(t *testing.T) {
	w := WeekdayNth{time.Monday, 2, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 12, 0, 0},
		},
		false: {
			{2018, 11, 26, 0, 0},
			{2018, 11, 19, 0, 0},
			{2018, 11, 5, 0, 0},
			{2018, 11, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayRange_String(t *testing.T) {
	w := WeekdayRange{time.Tuesday, time.Friday, true}

	if expected := "2-5"; w.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
	}

	w = WeekdayRange{time.Tuesday, time.Friday, false}

	if expected := "TUE-FRI"; w.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, w.String())
	}
}

func TestWeekdayRange_Match(t *testing.T) {
	w := WeekdayRange{time.Tuesday, time.Friday, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 8, 0, 0},
		},
		false: {
			{2018, 11, 10, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}

	w = WeekdayRange{time.Friday, time.Tuesday, true}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 10, 0, 0},
		},
		false: {
			{2018, 11, 8, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayEvery_String(t *testing.T) {
	m := WeekdayEvery(2)

	if expected := "*/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestWeekdayEvery_Match(t *testing.T) {
	w := WeekdayEvery(3)

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 25, 0, 0},
			{2018, 11, 28, 0, 0},
			{2018, 12, 1, 0, 0},
		},
		false: {
			{2018, 11, 26, 0, 0},
			{2018, 11, 27, 0, 0},
			{2018, 11, 29, 0, 0},
			{2018, 11, 30, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayEveryFrom_String(t *testing.T) {
	m := WeekdayEveryFrom{1, 2}

	if expected := "1/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestWeekdayEveryFrom_Match(t *testing.T) {
	w := WeekdayEveryFrom{1, 2}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 26, 0, 0},
			{2018, 11, 28, 0, 0},
			{2018, 11, 30, 0, 0},
		},
		false: {
			{2018, 11, 25, 0, 0},
			{2018, 11, 27, 0, 0},
			{2018, 11, 29, 0, 0},
			{2018, 12, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

func TestWeekdayEveryInRange_String(t *testing.T) {
	m := WeekdayEveryInRange{1, 5, 2}

	if expected := "1-5/2"; m.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, m.String())
	}
}

func TestWeekdayEveryInRange_Match(t *testing.T) {
	w := WeekdayEveryInRange{1, 5, 2}

	for expected, items := range map[bool][][5]int{
		true: {
			{2018, 11, 26, 0, 0},
			{2018, 11, 28, 0, 0},
			{2018, 11, 30, 0, 0},
		},
		false: {
			{2018, 11, 25, 0, 0},
			{2018, 11, 27, 0, 0},
			{2018, 11, 29, 0, 0},
			{2018, 12, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, w, expected)
		}
	}
}

/********
 * Year *
 ********/

func TestYear_String(t *testing.T) {
	y := Year(1978)

	if expected := "1978"; y.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, y.String())
	}
}

func TestYear_Match(t *testing.T) {
	y := Year(1978)

	for expected, items := range map[bool][][5]int{
		true: {
			{1978, 9, 30, 0, 5},
		},
		false: {
			{1079, 9, 30, 0, 4},
		},
	} {
		for _, item := range items {
			tester(t, item, y, expected)
		}
	}
}

func TestYearRange_String(t *testing.T) {
	y := YearRange{1978, 2000}

	if expected := "1978-2000"; y.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, y.String())
	}
}

func TestYearRange_Match(t *testing.T) {
	y := YearRange{1978, 2000}

	for expected, items := range map[bool][][5]int{
		true: {
			{1978, 9, 30, 0, 8},
		},
		false: {
			{2001, 9, 30, 0, 4},
		},
	} {
		for _, item := range items {
			tester(t, item, y, expected)
		}
	}
}

func TestYearEvery_String(t *testing.T) {
	y := YearEvery(2)

	if expected := "*/2"; y.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, y.String())
	}
}

func TestYearEvery_Match(t *testing.T) {
	y := YearEvery(3)

	// "*/3" is "1900-2099/3": 1900, 1903, ... 2017, 2020 — not the years
	// divisible by 3.
	for expected, items := range map[bool][][5]int{
		true: {
			{2017, 11, 26, 0, 0},
			{2020, 11, 29, 0, 0},
			{2023, 11, 30, 0, 0},
		},
		false: {
			{2016, 12, 1, 0, 0},
			{2018, 11, 27, 0, 0},
			{2019, 11, 25, 0, 0},
			{2021, 11, 30, 0, 0},
			{2022, 11, 28, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, y, expected)
		}
	}
}

func TestYearEveryFrom_String(t *testing.T) {
	y := YearEveryFrom{2001, 2}

	if expected := "2001/2"; y.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, y.String())
	}
}

func TestYearEveryFrom_Match(t *testing.T) {
	y := YearEveryFrom{2001, 2}

	for expected, items := range map[bool][][5]int{
		true: {
			{2001, 11, 26, 0, 0},
			{2003, 11, 28, 0, 0},
			{2005, 11, 30, 0, 0},
		},
		false: {
			{1999, 11, 25, 0, 0},
			{2000, 11, 27, 0, 0},
			{2002, 11, 29, 0, 0},
			{2004, 12, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, y, expected)
		}
	}
}

func TestYearEveryInRange_String(t *testing.T) {
	y := YearEveryInRange{2000, 2005, 2}

	if expected := "2000-2005/2"; y.String() != expected {
		t.Errorf("Expected \"%s\", got \"%s\"", expected, y.String())
	}
}

func TestYearEveryInRange_Match(t *testing.T) {
	y := YearEveryInRange{2000, 2005, 2}

	for expected, items := range map[bool][][5]int{
		true: {
			{2000, 11, 26, 0, 0},
			{2002, 11, 28, 0, 0},
			{2004, 11, 30, 0, 0},
		},
		false: {
			{1999, 11, 25, 0, 0},
			{2001, 11, 27, 0, 0},
			{2005, 11, 29, 0, 0},
			{2006, 12, 1, 0, 0},
		},
	} {
		for _, item := range items {
			tester(t, item, y, expected)
		}
	}
}

/************
 * Due Next *
 ************/

func TestDueNext_Happy(t *testing.T) {
	relativeTo := time.Date(1978, 9, 30, 3, 0, 0, 0, time.UTC)

	testData := []struct {
		schedule Schedule
		expected time.Time
	}{
		{
			Schedule{ // Every day at 6:00 AM
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(6)},
				Day:     nil,
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 9, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // At 2:15 PM on the 1st of every month
				Minute:  Field{Minute(15)},
				Hour:    Field{Hour(14)},
				Day:     Field{Day(1)},
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 10, 1, 14, 15, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every Friday at 6:30 PM
				Minute:  Field{Minute(30)},
				Hour:    Field{Hour(18)},
				Day:     nil,
				Month:   nil,
				Weekday: Field{Weekday{time.Friday, true}},
				Year:    nil,
			}, time.Date(1978, 10, 6, 18, 30, 0, 0, time.UTC),
		},
		{
			Schedule{ // Once a year on Jan 1 at midnight
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(0)},
				Day:     Field{Day(1)},
				Month:   Field{Month{time.January, true}},
				Weekday: nil,
				Year:    nil,
			}, time.Date(1979, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every 10 minutes
				Minute:  Field{MinuteEvery(10)},
				Hour:    nil,
				Day:     nil,
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 9, 30, 3, 10, 0, 0, time.UTC),
		},
		{
			Schedule{ // Weekdays at 12:00 PM
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(12)},
				Day:     nil,
				Month:   nil,
				Weekday: Field{WeekdayEveryInRange{time.Monday, time.Friday, 1}},
				Year:    nil,
			}, time.Date(1978, 10, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // Sundays at 4:05 AM
				Minute:  Field{Minute(5)},
				Hour:    Field{Hour(4)},
				Day:     nil,
				Month:   nil,
				Weekday: Field{Weekday{time.Sunday, true}},
				Year:    nil,
			}, time.Date(1978, 10, 1, 4, 5, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every 3rd hour, on the hour
				Minute:  Field{Minute(0)},
				Hour:    Field{HourEvery(3)},
				Day:     nil,
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 9, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // 11:45 PM on the 28th of each month
				Minute:  Field{Minute(45)},
				Hour:    Field{Hour(23)},
				Day:     Field{Day(28)},
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 10, 28, 23, 45, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every 2 hours between 9 AM and 5 PM daily
				Minute:  Field{Minute(0)},
				Hour:    Field{HourEveryInRange{9, 17, 2}},
				Day:     nil,
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 9, 30, 9, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // At midnight on Saturdays and Sundays
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(0)},
				Day:     nil,
				Month:   nil,
				Weekday: Field{Weekday{time.Saturday, true}, Weekday{time.Sunday, true}},
				Year:    nil,
			}, time.Date(1978, 10, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // Oct 15 at 8:30 AM every year
				Minute:  Field{Minute(30)},
				Hour:    Field{Hour(8)},
				Day:     Field{Day(15)},
				Month:   Field{Month{time.October, true}},
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 10, 15, 8, 30, 0, 0, time.UTC),
		},
		{
			Schedule{ // On Feb 29 at midnight (leap year only)
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(0)},
				Day:     Field{Day(29)},
				Month:   Field{Month{time.February, true}},
				Weekday: nil,
				Year:    nil,
			}, time.Date(1980, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // On Sep 30 at 11:45 PM each year
				Minute:  Field{Minute(45)},
				Hour:    Field{Hour(23)},
				Day:     Field{Day(30)},
				Month:   Field{Month{time.September, true}},
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 9, 30, 23, 45, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every Monday at 10:15 AM
				Minute:  Field{Minute(15)},
				Hour:    Field{Hour(10)},
				Day:     nil,
				Month:   nil,
				Weekday: Field{Weekday{time.Monday, true}},
				Year:    nil,
			}, time.Date(1978, 10, 2, 10, 15, 0, 0, time.UTC),
		},
		{
			Schedule{ // Every day at 1:00 AM
				Minute:  Field{Minute(0)},
				Hour:    Field{Hour(1)},
				Day:     nil,
				Month:   nil,
				Weekday: nil,
				Year:    nil,
			}, time.Date(1978, 10, 1, 1, 0, 0, 0, time.UTC),
		},
		{
			Schedule{ // At Oct 15, 2000 8:30 AM
				Minute:  Field{Minute(30)},
				Hour:    Field{Hour(8)},
				Day:     Field{Day(15)},
				Month:   Field{Month{time.October, true}},
				Weekday: nil,
				Year:    Field{Year(2000)},
			}, time.Date(2000, 10, 15, 8, 30, 0, 0, time.UTC),
		},
	}

	for _, td := range testData {
		dueNext := td.schedule.NextMatching(relativeTo)
		if dueNext == nil {
			t.Errorf("Expected a due next time, got nil.")
		} else if !dueNext.Equal(td.expected) {
			t.Errorf("Expected next due time: %s, got: %s", td.expected.Format("2006-01-02 15:04"), td.schedule.NextMatching(relativeTo).Format("2006-01-02 15:04"))
		}
	}
}

/*************
 * Step form *
 *************/

func TestEveryStep_EquivalentToExplicitRange(t *testing.T) {
	// "*/N" is shorthand for the field's whole range stepped by N, so it has to
	// match exactly what the explicit "min-max/N" form matches. The two forms
	// diverge if a step is ever computed as a plain modulus, which is wrong for
	// every field whose range does not start at 0.
	for _, c := range []struct {
		field string
		star  string
		full  string
	}{
		{"minute", "*/7 * * * *", "0-59/7 * * * *"},
		{"hour", "* */7 * * *", "* 0-23/7 * * *"},
		{"day", "* * */7 * *", "* * 1-31/7 * *"},
		{"month", "* * * */5 *", "* * * 1-12/5 *"},
		{"weekday", "* * * * */2", "* * * * 0-6/2"},
		{"year", "* * * * * */7", "* * * * * 1900-2099/7"},
	} {
		star, err := Parse(c.star)
		if err != nil {
			t.Fatalf("%s: cannot parse %q: %v", c.field, c.star, err)
		}

		full, err := Parse(c.full)
		if err != nil {
			t.Fatalf("%s: cannot parse %q: %v", c.field, c.full, err)
		}

		// A 97 minute step is coprime with 60, so the sweep visits every minute
		// and hour, and spans enough years to exercise the month and year steps.
		for tm := time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC); tm.Year() < 2018; tm = tm.Add(97 * time.Minute) {
			if s, f := star.Match(tm), full.Match(tm); s != f {
				t.Fatalf("%s: %q matched %t but %q matched %t at %s",
					c.field, c.star, s, c.full, f, tm.Format("2006-01-02 15:04"))
			}
		}
	}
}
