package cron

import (
	"reflect"
	"testing"
	"time"
)

var happyTestData = []struct {
	cronExpr string
	expected Schedule
}{
	{
		// Standard five-field cron; an omitted year matches every year.
		cronExpr: "* * * * *",
		expected: Schedule{},
	},
	{
		// Five-field with values and no year.
		cronExpr: "30 8 1 1 *",
		expected: Schedule{
			Minute: Field{Minute(30)},
			Hour:   Field{Hour(8)},
			Day:    Field{Day(1)},
			Month:  Field{Month{time.January, true}},
		},
	},
	{
		cronExpr: "1,*/2,3/4,5-20/3 * * * *",
		expected: Schedule{Minute: Field{
			Minute(1),
			MinuteEvery(2),
			MinuteEveryFrom{3, 4},
			MinuteEveryInRange{5, 20, 3},
		}},
	},
	{
		cronExpr: "1-5 * * * *",
		expected: Schedule{Minute: Field{
			MinuteRange{1, 5},
		}},
	},
	{
		cronExpr: "1-5,8 * * * *",
		expected: Schedule{Minute: Field{
			MinuteRange{1, 5},
			Minute(8),
		}},
	},
	{
		cronExpr: "* 5,*/4,2/3,10-15/2 * * *",
		expected: Schedule{Hour: Field{
			Hour(5),
			HourEvery(4),
			HourEveryFrom{2, 3},
			HourEveryInRange{10, 15, 2},
		}},
	},
	{
		cronExpr: "* 5,8-10 * * *",
		expected: Schedule{Hour: Field{
			Hour(5),
			HourRange{8, 10}}},
	},
	{
		cronExpr: "* * */3,1/2,5-10/2,1,5W,L,LW,5-10 * *",
		expected: Schedule{Day: Field{
			DayEvery(3),
			DayEveryFrom{1, 2},
			DayEveryInRange{5, 10, 2},
			Day(1),
			DayOnWeekday(5),
			DayLast{},
			DayLastWeekday{},
			DayRange{5, 10},
		}},
	},
	{
		cronExpr: "* * * */2,3/2,2-6/2 *",
		expected: Schedule{Month: Field{
			MonthEvery(2),
			MonthEveryFrom{time.March, 2},
			MonthEveryInRange{time.February, time.June, 2},
		}},
	},
	{
		cronExpr: "* * * 1,FEB *",
		expected: Schedule{Month: Field{
			Month{time.January, true},
			Month{time.February, false},
		}},
	},
	{
		cronExpr: "* * * 1-5 *",
		expected: Schedule{Month: Field{
			MonthRange{time.January, time.May, true},
		}},
	},
	{
		cronExpr: "* * * FEB-DEC *",
		expected: Schedule{Month: Field{
			MonthRange{time.February, time.December, false},
		}},
	},
	{
		cronExpr: "* * * * 1,SUN",
		expected: Schedule{Weekday: Field{
			Weekday{time.Monday, true},
			Weekday{time.Sunday, false},
		}},
	},
	{
		cronExpr: "* * * * 1L,SUNL",
		expected: Schedule{Weekday: Field{
			WeekdayLast{time.Monday, true},
			WeekdayLast{time.Sunday, false},
		}},
	},
	{
		cronExpr: "* * * * 1#2,SUN#3",
		expected: Schedule{Weekday: Field{
			WeekdayNth{time.Monday, 2, true},
			WeekdayNth{time.Sunday, 3, false},
		}},
	},
	{
		cronExpr: "* * * * 1-2,TUE-FRI",
		expected: Schedule{Weekday: Field{
			WeekdayRange{time.Monday, time.Tuesday, true},
			WeekdayRange{time.Tuesday, time.Friday, false},
		}},
	},
	{
		cronExpr: "* * * * */2,1/2,1-5/2",
		expected: Schedule{Weekday: Field{
			WeekdayEvery(2),
			WeekdayEveryFrom{1, 2},
			WeekdayEveryInRange{1, 5, 2},
		}},
	},
	{
		cronExpr: "* * * * * 1978",
		expected: Schedule{Year: Field{
			Year(1978),
		}},
	},
	{
		cronExpr: "* * * * * 1978-2000,2018",
		expected: Schedule{Year: Field{
			YearRange{1978, 2000},
			Year(2018),
		}},
	},
	{
		cronExpr: "* * * * * */4,2000/3,1978-2000/2",
		expected: Schedule{Year: Field{
			YearEvery(4),
			YearEveryFrom{2000, 3},
			YearEveryInRange{1978, 2000, 2},
		}},
	},
}

func TestParseHappy(t *testing.T) {
	for _, d := range happyTestData {
		got, err := Parse(d.cronExpr)

		if err != nil {
			t.Errorf("Unexpected parsing error.")
		}

		if !reflect.DeepEqual(got, d.expected) {
			t.Errorf("Unexpected parsing result.\nParsed  : %s\nExpected: %v <=> %#v\nGot     : %v <=> %#v",
				d.cronExpr,
				d.expected,
				d.expected,
				got,
				got)
		}
	}
}

var sadTestData = []string{
	// Invalid gibberish
	"",
	"abc",
	// Too few fields (minimum is five)
	"* * * *",
	"* * *",
	// Too many fields (maximum is six)
	"* * * * * * *",
	// A "*" year says nothing an omitted year does not, and accepting it would
	// silently reinterpret six-field Quartz expressions, whose sixth field is
	// the day of week.
	"* * * * * *",
	"0 0 12 * * *", // Quartz for noon daily; here it would read as the 12th
	"30 8 1 1 * *",
	// Invalid minutes
	"60 * * * *",
	"20-5 * * * *",
	// Invalid hours
	"* 24 * * *",
	"* 23, 1 * * * *",
	"* 24,1 * * *",
	"* 20-5 * * *",
	"* 5-24 * * *",
	// Invalid days
	"* * 32 * *",
	"* * 31L * *",
	"* * 31LW * *",
	"* * L/2 * *",
	"* * LW/2 * *",
	"* * 1-32 * *",
	"* * 12-2 * *",
	// Invalid months
	"* * * 13 *",
	"* * * Feb *",
	"* * * February *",
	// Invalid weekdays
	"* * * * SUNL#1",
	"* * * * SUNL-MONL",
	"* * * * 1#2-2L",
	"* * * * 1L/2",
	"* * * * 1#2/2",
	// Invalid years
	"* * * * * 0",
	"* * * * * 1978-79",
	"* * * * * 1800",
	"* * * * * 2100",
	"* * * * * */*",
}

func TestParseSad(t *testing.T) {
	for _, d := range sadTestData {
		_, err := Parse(d)

		if err == nil {
			t.Errorf("Parsing error should have occured.")
		}
	}
}

// TestParseRoundTrip checks that Schedule.String renders an expression Parse
// accepts again, producing the same schedule. It is what keeps String honest
// about the year field, which is written only when the schedule restricts it.
func TestParseRoundTrip(t *testing.T) {
	for _, d := range happyTestData {
		first, err := Parse(d.cronExpr)
		if err != nil {
			t.Errorf("Unexpected parsing error for %q: %v", d.cronExpr, err)

			continue
		}

		rendered := first.String()

		second, err := Parse(rendered)
		if err != nil {
			t.Errorf("%q rendered as %q, which does not parse: %v", d.cronExpr, rendered, err)

			continue
		}

		if !reflect.DeepEqual(first, second) {
			t.Errorf("%q rendered as %q, which parses to a different schedule.\nFirst : %#v\nSecond: %#v",
				d.cronExpr, rendered, first, second)
		}
	}
}
