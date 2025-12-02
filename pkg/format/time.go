// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package format

import (
	"fmt"
	"strconv"
	"time"

	"github.com/altshiftab/jsonschema/pkg/types"
)

// dateTimeFormat requires a valid date/time.
func dateTimeFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidDateTime(s) {
		return fmt.Errorf("%q is not a valid date-time", s)
	}
	return nil
}

// isValidDateTime reports whether s is a valid RFC3339 date-time.
func isValidDateTime(s string) bool {
	// date-time = full-date "T" full-time
	if len(s) < dateLen {
		return false
	}
	if !isValidDate(s[:dateLen]) {
		return false
	}
	s = s[dateLen:]
	if len(s) == 0 || (s[0] != 'T' && s[0] != 't') {
		return false
	}
	s = s[1:]
	return isValidTime(s)
}

// dateFormat requires a valid date.
func dateFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidDate(s) {
		return fmt.Errorf("%q is not a valid date", s)
	}
	return nil
}

// dateLen is the length of a RFC3339 full-date.
const dateLen = 10

// isValidDate reports whether s is a valid RFC3339 full-date.
// The format is YYYY-MM-DD.
func isValidDate(s string) bool {
	// full-date     = date-fullyear "-" date-month "-" date-mday
	// date-fullyear = 4DIGIT
	// date-month    = 2DIGIT  ; 01-12
	// date-mday     = 2DIGIT  ; 01-28, 01-29, 01-30, 01-31 based on month/year
	if len(s) != dateLen {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}

	year, err := strconv.Atoi(s[:4])
	if err != nil {
		return false
	}
	month, err := strconv.Atoi(s[5:7])
	if err != nil {
		return false
	}
	mday, err := strconv.Atoi(s[8:])
	if err != nil {
		return false
	}

	// Make sure year/month/day are valid.
	if year < 0 || month < 1 || month > 12 || mday < 1 || mday > 31 {
		return false
	}
	dy, dm, dd := time.Date(year, time.Month(month), mday, 0, 0, 0, 0, time.UTC).Date()
	if dy != year || dm != time.Month(month) || dd != mday {
		return false
	}

	return true
}

// timeFormat requires a valid time.
func timeFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidTime(s) {
		return fmt.Errorf("%q is not a valid time", s)
	}
	return nil
}

// isValidTime reports whether s is a valid RFC3339 full-time.
// The format is HH:MM:SS[frac]offset
func isValidTime(s string) bool {
	// time-hour      = 2DIGIT  ; 00-23
	// time-minute    = 2DIGIT  ; 00-59
	// time-second    = 2DIGIT  ; 00-58, 00-59, 00-60 based on leap second rules
	// time-secfrac   = "." 1*DIGIT
	// time-numoffset = ("+" / "-") time-hour ":" time-minute
	// time-offset    = "Z" / time-numoffset
	// partial-time   = time-hour ":" time-minute ":" time-second [time-secfrac]
	// full-time      = partial-time time-offset
	if len(s) < 8 {
		return false
	}
	if s[2] != ':' || s[5] != ':' {
		return false
	}

	hour, err := strconv.Atoi(s[:2])
	if err != nil {
		return false
	}
	minute, err := strconv.Atoi(s[3:5])
	if err != nil {
		return false
	}
	second, err := strconv.Atoi(s[6:8])
	if err != nil {
		return false
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 60 {
		return false
	}

	s = s[8:]
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		if len(s) == 0 {
			return false
		}
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			s = s[1:]
		}
	}

	if len(s) == 0 {
		return false
	}
	negOffset := false
	switch s[0] {
	case 'Z', 'z':
		if second == 60 && (hour != 23 || minute != 59) {
			return false
		}
		return len(s) == 1
	case '+':
		s = s[1:]
	case '-':
		negOffset = true
		s = s[1:]
	default:
		return false
	}

	if len(s) != 5 {
		return false
	}
	if s[2] != ':' {
		return false
	}
	hourOffset, err := strconv.Atoi(s[:2])
	if err != nil {
		return false
	}
	minuteOffset, err := strconv.Atoi(s[3:])
	if err != nil {
		return false
	}
	if hourOffset < 0 || hourOffset > 23 || minuteOffset < 0 || minuteOffset > 59 {
		return false
	}

	if second == 60 {
		// The time zone offset is counted from UTC,
		// and we have local time, so we need to add a negative
		// offset and subtract a positive one.
		if !negOffset {
			hourOffset = -hourOffset
			minuteOffset = -minuteOffset
		}
		if (hour+hourOffset != 23 && hour+hourOffset != 0) || (minute+minuteOffset != 59 && minute+minuteOffset != -1) {
			return false
		}
	}

	return true
}

// durationFormat requires a valid duration.
func durationFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidDuration(s) {
		return fmt.Errorf("%q is not a valid duration", s)
	}
	return nil
}

// isValidDuration reports whether s is a valid RFC3339 duration.
func isValidDuration(s string) bool {
	isChar := func(s string, ch1, ch2 byte) bool {
		if len(s) == 0 {
			return false
		}
		return s[0] == ch1 || s[0] == ch2
	}

	isDigit := func(s string) bool {
		if len(s) == 0 {
			return false
		}
		return s[0] >= '0' && s[0] <= '9'
	}

	skipDigits := func(s string) (string, bool) {
		if !isDigit(s) {
			return "", false
		}
		s = s[1:]
		for {
			if !isDigit(s) {
				return s, true
			}
			s = s[1:]
		}
	}

	// duration = "P" (dur-date / dur-time / dur-week)

	if !isChar(s, 'P', 'p') {
		return false
	}
	s = s[1:]

	// dur-second  = 1*DIGIT "S"
	// dur-minute  = 1*DIGIT "M" [dur-second]
	// dur-hour    = 1*DIGIT "H" [dur-minute]
	// dur-time    = "T" (dur-hour / dur-minute / dur-second)
	validDurTime := func(s string) bool {
		if !isChar(s, 'T', 't') {
			return false
		}
		s = s[1:]
		s, ok := skipDigits(s)
		if !ok {
			return false
		}
		if isChar(s, 'H', 'h') {
			s = s[1:]
			if len(s) == 0 {
				return true
			}
			s, ok = skipDigits(s)
			if !ok {
				return false
			}
			if !isChar(s, 'M', 'm') {
				return false
			}
		}
		if isChar(s, 'M', 'm') {
			s = s[1:]
			if len(s) == 0 {
				return true
			}
			s, ok = skipDigits(s)
			if !ok {
				return false
			}
		}
		return isChar(s, 'S', 's')
	}

	// dur-day   = 1*DIGIT "D"
	// dur-week  = 1*DIGIT "W"
	// dur-month = 1*DIGIT "M" [dur-day]
	// dur-year  = 1*DIGIT "Y" [dur-month]
	// dur-date  = (dur-day / dur-month / dur-year) [dur-time]
	validDurDateOrWeek := func(s string) bool {
		s, ok := skipDigits(s)
		if !ok {
			return false
		}
		if isChar(s, 'W', 'w') {
			s = s[1:]
			if len(s) == 0 {
				return true
			}
			return validDurTime(s)
		}
		if isChar(s, 'Y', 'y') {
			s = s[1:]
			if len(s) == 0 {
				return true
			}
			if isChar(s, 'T', 't') {
				return validDurTime(s)
			}
			s, ok = skipDigits(s)
			if !ok {
				return false
			}
			if !isChar(s, 'M', 'm') {
				return false
			}
		}
		if isChar(s, 'M', 'm') {
			s = s[1:]
			if len(s) == 0 {
				return true
			}
			if isChar(s, 'T', 't') {
				return validDurTime(s)
			}
			s, ok = skipDigits(s)
			if !ok {
				return false
			}
			if !isChar(s, 'D', 'd') {
				return false
			}
		}
		if !isChar(s, 'D', 'd') {
			return false
		}
		s = s[1:]
		if len(s) == 0 {
			return true
		}
		return validDurTime(s)
	}

	if isChar(s, 'T', 't') {
		return validDurTime(s)
	}
	return validDurDateOrWeek(s)
}
