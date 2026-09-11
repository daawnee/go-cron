package cron

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	baseField = struct {
		minute     string
		hour       string
		dom        string
		month      string
		monthNames string
		dow        string
		dowNames   string
		year       string
	}{
		"[0-5]?\\d",
		"[01]?\\d|2[0-3]",
		"0?[1-9]|[12]\\d|3[01]",
		"0?[1-9]|1[0-2]",
		"JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC",
		"[0-6]",
		"MON|TUE|WED|THU|FRI|SAT|SUN",
		"(?:19|20)\\d{2}",
	}

	period = "\\/([1-9]\\d?)"

	fullField = struct {
		minute string
		hour   string
		dom    string
		month  string
		dow    string
		year   string
	}{
		"(?:\\*" + period + ")|(?:(" + baseField.minute + ")(?:-(" + baseField.minute + "))?(?:" + period + ")?)",
		"(?:\\*" + period + ")|(?:(" + baseField.hour + ")(?:-(" + baseField.hour + "))?(?:" + period + ")?)",
		"(?:\\*" + period + ")|(?:(" + baseField.dom + ")(?:-(" + baseField.dom + "))?(?:" + period + ")?)|(?:(L|" + baseField.dom + ")(W)?)",
		"(?:\\*" + period + ")|(?:(" + baseField.monthNames + ")(?:-(" + baseField.monthNames + "))?|(?:(" + baseField.month + ")(?:-(" + baseField.month + "))?)(?:" + period + ")?)",
		"(?:\\*" + period + ")|(?:(" + baseField.dowNames + ")(?:-(" + baseField.dowNames + "))?|(?:(" + baseField.dow + ")(?:-(" + baseField.dow + "))?)(?:" + period + ")?)|(?:(?:(" + baseField.dow + ")|(" + baseField.dowNames + "))(L|#([1-5]))?)",
		"(?:\\*" + period + ")|(?:(" + baseField.year + ")(?:-(" + baseField.year + "))?(?:" + period + ")?)",
	}

	fieldsCheck = regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)(?:\s+(\S+))?\s*$`)
	minuteCheck = regexp.MustCompile("^(?:" + fullField.minute + ")$")
	hourCheck   = regexp.MustCompile("^(?:" + fullField.hour + ")$")
	domCheck    = regexp.MustCompile("^(?:" + fullField.dom + ")$")
	monthCheck  = regexp.MustCompile("^(?:" + fullField.month + ")$")
	dowCheck    = regexp.MustCompile("^(?:" + fullField.dow + ")$")
	yearCheck   = regexp.MustCompile("^(?:" + fullField.year + ")$")
)

func toInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Parse parses a cron expression into a Schedule. The expression has five
// mandatory fields (minute, hour, day-of-month, month, day-of-week) and an
// optional sixth year field, so standard five-field cron expressions are
// accepted. An omitted year matches every year; writing "*" there is rejected,
// because it says the same thing and because accepting it would let a six field
// Quartz expression parse as an unrelated schedule. It returns an error if the
// expression has the wrong number of fields or any field is malformed or out of
// range.
func Parse(cronExpression string) (schedule Schedule, err error) {
	fields := fieldsCheck.FindStringSubmatch(cronExpression)
	if fields == nil {
		err = errors.New("invalid number of cron expression fields")
		return
	}

	// "*" as a year means exactly what leaving the field out means. Accepting it
	// is also what lets a six field Quartz expression — whose sixth field is the
	// day of week, not the year — parse here as a different schedule rather than
	// fail. Rejecting it keeps that mistake loud.
	if fields[6] == "*" {
		err = errors.New(`invalid year expression: "*" (omit the year field to match every year)`)
		return
	}

	var result Schedule

	for i, v := range []struct {
		fld *Field
		fnc func(string) (Field, error)
	}{
		{&result.Minute, parseMinute},
		{&result.Hour, parseHour},
		{&result.Day, parseDoM},
		{&result.Month, parseMonth},
		{&result.Weekday, parseDoW},
		{&result.Year, parseYear},
	} {
		*v.fld, err = v.fnc(fields[i+1])
		if err != nil {
			return
		}
	}

	return result, nil
}

func parseMinute(expression string) (minute Field, err error) {
	if expression == "*" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid minute expression: %q", field)

		matches := minuteCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			minute = append(minute, MinuteEvery(toInt(matches[1])))
		} else if matches[3] != "" {
			from := toInt(matches[2])
			to := toInt(matches[3])

			if from > to {
				return
			}

			if matches[4] != "" {
				minute = append(minute, MinuteEveryInRange{from, to, toInt(matches[4])})
			} else {
				minute = append(minute, MinuteRange{from, to})
			}
		} else {
			if matches[4] != "" {
				minute = append(minute, MinuteEveryFrom{toInt(matches[2]), toInt(matches[4])})
			} else {
				minute = append(minute, Minute(toInt(matches[2])))
			}
		}
	}

	return minute, nil
}

func parseHour(expression string) (hour Field, err error) {
	if expression == "*" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid hour expression: %q", field)

		matches := hourCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			hour = append(hour, HourEvery(toInt(matches[1])))
		} else if matches[3] != "" {
			from := toInt(matches[2])
			to := toInt(matches[3])

			if from > to {
				return
			}

			if matches[4] != "" {
				hour = append(hour, HourEveryInRange{from, to, toInt(matches[4])})
			} else {
				hour = append(hour, HourRange{from, to})
			}
		} else {
			if matches[4] != "" {
				hour = append(hour, HourEveryFrom{toInt(matches[2]), toInt(matches[4])})
			} else {
				hour = append(hour, Hour(toInt(matches[2])))
			}
		}
	}

	return hour, nil
}

func parseDoM(expression string) (dom Field, err error) {
	if expression == "*" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid day expression: %q", field)

		matches := domCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			dom = append(dom, DayEvery(toInt(matches[1])))
		} else if matches[3] != "" {
			from := toInt(matches[2])
			to := toInt(matches[3])

			if from > to {
				return
			}

			if matches[4] != "" {
				dom = append(dom, DayEveryInRange{from, to, toInt(matches[4])})
			} else {
				dom = append(dom, DayRange{from, to})
			}
		} else if matches[2] != "" {
			if matches[4] != "" {
				dom = append(dom, DayEveryFrom{toInt(matches[2]), toInt(matches[4])})
			} else {
				dom = append(dom, Day(toInt(matches[2])))
			}
		} else if matches[6] != "" {
			if matches[5] == "L" {
				dom = append(dom, DayLastWeekday{})
			} else {
				dom = append(dom, DayOnWeekday(toInt(matches[5])))
			}
		} else {
			if matches[5] == "L" {
				dom = append(dom, DayLast{})
			} else {
				dom = append(dom, Day(toInt(matches[5])))
			}
		}
	}

	return dom, nil
}

var monthToNum = map[string]time.Month{
	"JAN": 1, "FEB": 2, "MAR": 3,
	"APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9,
	"OCT": 10, "NOV": 11, "DEC": 12,
}

func parseMonth(expression string) (month Field, err error) {
	if expression == "*" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid month expression: %q", field)

		matches := monthCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			month = append(month, MonthEvery(toInt(matches[1])))
		} else if matches[2] != "" {
			if matches[3] != "" {
				from := monthToNum[matches[2]]
				to := monthToNum[matches[3]]

				if from > to {
					return
				}

				month = append(month, MonthRange{from, to, false})
			} else {
				month = append(month, Month{monthToNum[matches[2]], false})
			}
		} else if matches[5] != "" {
			from := toInt(matches[4])
			to := toInt(matches[5])

			if from > to {
				return
			}

			if matches[6] != "" {
				month = append(month, MonthEveryInRange{time.Month(from), time.Month(to), toInt(matches[6])})
			} else {
				month = append(month, MonthRange{time.Month(from), time.Month(to), true})
			}
		} else {
			if matches[6] != "" {
				month = append(month, MonthEveryFrom{time.Month(toInt(matches[4])), toInt(matches[6])})
			} else {
				month = append(month, Month{time.Month(toInt(matches[4])), true})
			}
		}
	}

	return month, nil
}

var weekdayToType = map[string]time.Weekday{
	"SUN": time.Sunday,
	"MON": time.Monday,
	"TUE": time.Tuesday,
	"WED": time.Wednesday,
	"THU": time.Thursday,
	"FRI": time.Friday,
	"SAT": time.Saturday}

func parseDoW(expression string) (dow Field, err error) {
	if expression == "*" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid weekday expression: %q", field)

		matches := dowCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			dow = append(dow, WeekdayEvery(toInt(matches[1])))
		} else if matches[2] != "" {
			if matches[3] != "" {
				from := weekdayToType[matches[2]]
				to := weekdayToType[matches[3]]

				if from > to {
					return
				}

				dow = append(dow, WeekdayRange{from, to, false})
			} else {
				dow = append(dow, Weekday{weekdayToType[matches[2]], false})
			}
		} else if matches[4] != "" {
			if matches[5] != "" {
				from := time.Weekday(toInt(matches[4]))
				to := time.Weekday(toInt(matches[5]))

				if from > to {
					return
				}

				if matches[6] != "" {
					dow = append(dow, WeekdayEveryInRange{from, to, toInt(matches[6])})
				} else {
					dow = append(dow, WeekdayRange{from, to, true})
				}
			} else {
				if matches[6] != "" {
					dow = append(dow, WeekdayEveryFrom{time.Weekday(toInt(matches[4])), toInt(matches[6])})
				} else {
					dow = append(dow, Weekday{time.Weekday(toInt(matches[4])), true})
				}
			}
		} else {
			var at time.Weekday
			var numeric bool

			if matches[7] != "" {
				at = time.Weekday(toInt(matches[7]))
				numeric = true
			} else {
				at = weekdayToType[matches[8]]
				numeric = false
			}

			if matches[9] == "" {
				dow = append(dow, Weekday{at, numeric})
			} else if matches[9] == "L" {
				dow = append(dow, WeekdayLast{at, numeric})
			} else {
				dow = append(dow, WeekdayNth{at, toInt(matches[10]), numeric})
			}
		}
	}

	return dow, nil
}

func parseYear(expression string) (year Field, err error) {
	// An omitted year field matches every year.
	if expression == "" {
		return
	}

	for field := range strings.SplitSeq(expression, ",") {
		err = fmt.Errorf("invalid year expression: %q", field)

		matches := yearCheck.FindStringSubmatch(field)
		if matches == nil {
			return
		}

		if matches[1] != "" {
			year = append(year, YearEvery(toInt(matches[1])))
		} else if matches[3] != "" {
			from := toInt(matches[2])
			to := toInt(matches[3])

			if from > to {
				return
			}

			if matches[4] != "" {
				year = append(year, YearEveryInRange{from, to, toInt(matches[4])})
			} else {
				year = append(year, YearRange{from, to})
			}
		} else {
			if matches[4] != "" {
				year = append(year, YearEveryFrom{toInt(matches[2]), toInt(matches[4])})
			} else {
				year = append(year, Year(toInt(matches[2])))
			}
		}
	}

	return year, nil
}
