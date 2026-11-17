package utils

import (
	"regexp"
	"strconv"
	"strings"
)

var timeInputPattern = regexp.MustCompile(`(?i)^\s*(\d{1,2})[:.](\d{2})(?::(\d{2}))?\s*(am|pm)?(?:\s*(?:wib|wita|wit|utc|gmt)|\s+[a-z]{2,5})?\s*$`)

// ParseTimeToMinutes mengonversi string waktu multi-format menjadi jumlah menit sejak tengah malam (0..1439).
func ParseTimeToMinutes(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}

	matches := timeInputPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return 0, false
	}

	hours, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	minutes, err := strconv.Atoi(matches[2])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, false
	}

	if matches[3] != "" {
		seconds, secErr := strconv.Atoi(matches[3])
		if secErr != nil || seconds < 0 || seconds > 59 {
			return 0, false
		}
	}

	meridiem := strings.ToLower(matches[4])
	if meridiem != "" {
		if hours < 1 || hours > 12 {
			return 0, false
		}
		if meridiem == "am" && hours == 12 {
			hours = 0
		} else if meridiem == "pm" && hours != 12 {
			hours += 12
		}
	} else {
		if hours < 0 || hours > 23 {
			return 0, false
		}
	}

	return hours*60 + minutes, true
}

// IsTimeBefore membandingkan apakah t1 lebih awal secara kronologis daripada t2 apabila keduanya dapat diparsing.
func IsTimeBefore(t1, t2 string) (validComparison bool, isBefore bool) {
	m1, ok1 := ParseTimeToMinutes(t1)
	m2, ok2 := ParseTimeToMinutes(t2)
	if !ok1 || !ok2 {
		return false, false
	}
	return true, m1 < m2
}
