package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeInputPattern = regexp.MustCompile(`(?i)^\s*(\d{1,2})[:.](\d{2})(?::(\d{2}))?\s*(am|pm)?(?:\s*(?:wib|wita|wit|utc|gmt)|\s+[a-z]{2,5})?\s*$`)

var indonesianMonths = map[string]time.Month{
	"januari":   time.January,
	"februari":  time.February,
	"maret":     time.March,
	"april":     time.April,
	"mei":       time.May,
	"juni":      time.June,
	"juli":      time.July,
	"agustus":   time.August,
	"september": time.September,
	"oktober":   time.October,
	"november":  time.November,
	"desember":  time.December,
	"jan":       time.January,
	"feb":       time.February,
	"mar":       time.March,
	"apr":       time.April,
	"jun":       time.June,
	"jul":       time.July,
	"agt":       time.August,
	"agu":       time.August,
	"sep":       time.September,
	"okt":       time.October,
	"nov":       time.November,
	"des":       time.December,
}

var wibLocation = time.FixedZone("WIB", 7*3600)

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

// ParseConcertDateTime menggabungkan tanggal dan waktu konser menjadi objek time.Time (WIB UTC+7).
func ParseConcertDateTime(dateStr, timeStr string) (time.Time, error) {
	trimmedDate := strings.TrimSpace(dateStr)
	if trimmedDate == "" {
		return time.Time{}, fmt.Errorf("tanggal konser kosong")
	}

	// Cek apakah dateStr sudah berformat ISO/RFC3339 penuh
	if t, err := time.Parse(time.RFC3339, trimmedDate); err == nil {
		return t.In(wibLocation), nil
	}

	// Parsing jam dan menit
	minutesOfDay, ok := ParseTimeToMinutes(timeStr)
	if !ok {
		minutesOfDay = 19*60 + 30 // Default 19:30 jika waktu tidak terdefinisi
	}
	hour := minutesOfDay / 60
	minute := minutesOfDay % 60

	// Format YYYY-MM-DD
	isoPattern := regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})`)
	if m := isoPattern.FindStringSubmatch(trimmedDate); m != nil {
		year, _ := strconv.Atoi(m[1])
		monthNum, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		return time.Date(year, time.Month(monthNum), day, hour, minute, 0, 0, wibLocation), nil
	}

	// Format teks Indonesia, e.g. "Sabtu, 18 April 2026" atau "18 April 2026"
	cleaned := strings.ToLower(trimmedDate)
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx+1:])
	}
	if idx := strings.Index(cleaned, "@"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	parts := strings.Fields(cleaned)
	if len(parts) >= 3 {
		day, dayErr := strconv.Atoi(parts[0])
		monthStr := parts[1]
		year, yearErr := strconv.Atoi(parts[2])

		month, monthFound := indonesianMonths[monthStr]
		if dayErr == nil && yearErr == nil && monthFound {
			return time.Date(year, month, day, hour, minute, 0, 0, wibLocation), nil
		}
	}

	return time.Time{}, fmt.Errorf("format tanggal konser tidak dikenali: %s", dateStr)
}

// ParseOpenGateDateTime menggabungkan tanggal konser dengan jam open gate (WIB UTC+7).
func ParseOpenGateDateTime(dateStr, openGateStr string) (time.Time, error) {
	return ParseConcertDateTime(dateStr, openGateStr)
}

