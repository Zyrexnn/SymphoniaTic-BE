package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeInputPattern = regexp.MustCompile(`(?i)^\s*(\d{1,2})[:.](\d{2})(?::(\d{2}))?\s*(am|pm)?(?:\s*(wib|wita|wit|utc|gmt)|\s+[a-z]{2,5})?\s*$`)

var monthAliases = map[string]time.Month{
	"januari":   time.January,
	"jan":       time.January,
	"january":   time.January,
	"februari":  time.February,
	"feb":       time.February,
	"february":  time.February,
	"peb":       time.February,
	"pebruari":  time.February,
	"maret":     time.March,
	"mar":       time.March,
	"march":     time.March,
	"april":     time.April,
	"apr":       time.April,
	"mei":       time.May,
	"may":       time.May,
	"juni":      time.June,
	"jun":       time.June,
	"june":      time.June,
	"juli":      time.July,
	"jul":       time.July,
	"july":      time.July,
	"agustus":   time.August,
	"agu":       time.August,
	"agt":       time.August,
	"aug":       time.August,
	"august":    time.August,
	"september": time.September,
	"sep":       time.September,
	"sept":      time.September,
	"oktober":   time.October,
	"okt":       time.October,
	"oct":       time.October,
	"october":   time.October,
	"november":  time.November,
	"nov":       time.November,
	"nop":       time.November,
	"nopember":  time.November,
	"desember":  time.December,
	"des":       time.December,
	"dec":       time.December,
	"december":  time.December,
}

var indonesianWeekdays = [...]string{
	"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu",
}

var indonesianMonthNames = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var zoneLocations = map[string]*time.Location{
	"WIB":  time.FixedZone("WIB", 7*3600),
	"WITA": time.FixedZone("WITA", 8*3600),
	"WIT":  time.FixedZone("WIT", 9*3600),
	"UTC":  time.FixedZone("UTC", 0),
	"GMT":  time.FixedZone("GMT", 0),
}

var wibLocation = zoneLocations["WIB"]

func locationForZone(tz string) *time.Location {
	if loc, ok := zoneLocations[strings.ToUpper(strings.TrimSpace(tz))]; ok {
		return loc
	}
	return wibLocation
}

// ParseTimeWithZone mengonversi string waktu menjadi menit sejak tengah malam, zona waktu, dan status validitas.
func ParseTimeWithZone(raw string) (minutes int, tz string, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, "", false
	}

	matches := timeInputPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return 0, "", false
	}

	hours, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", false
	}

	mins, err := strconv.Atoi(matches[2])
	if err != nil || mins < 0 || mins > 59 {
		return 0, "", false
	}

	if matches[3] != "" {
		seconds, secErr := strconv.Atoi(matches[3])
		if secErr != nil || seconds < 0 || seconds > 59 {
			return 0, "", false
		}
	}

	meridiem := strings.ToLower(matches[4])
	if meridiem != "" {
		if hours < 1 || hours > 12 {
			return 0, "", false
		}
		if meridiem == "am" && hours == 12 {
			hours = 0
		} else if meridiem == "pm" && hours != 12 {
			hours += 12
		}
	} else {
		if hours < 0 || hours > 23 {
			return 0, "", false
		}
	}

	zone := strings.ToUpper(matches[5])
	if zone == "" {
		zone = "WIB"
	}

	return hours*60 + mins, zone, true
}

// ParseTimeToMinutes mengonversi string waktu multi-format menjadi jumlah menit sejak tengah malam (0..1439).
func ParseTimeToMinutes(raw string) (int, bool) {
	mins, _, ok := ParseTimeWithZone(raw)
	return mins, ok
}

// NormalizeTimeString menormalkan string waktu ke format kanonik "HH:MM TZ".
func NormalizeTimeString(raw string) (string, error) {
	mins, tz, ok := ParseTimeWithZone(raw)
	if !ok {
		return "", fmt.Errorf("format waktu tidak valid: %s", raw)
	}
	return fmt.Sprintf("%02d:%02d %s", mins/60, mins%60, tz), nil
}

func parseDateParts(dateStr string) (year int, month time.Month, day int, err error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return 0, 0, 0, fmt.Errorf("tanggal konser kosong")
	}

	if t, rfcErr := time.Parse(time.RFC3339, trimmed); rfcErr == nil {
		local := t.In(wibLocation)
		return local.Year(), local.Month(), local.Day(), nil
	}

	isoPattern := regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})(?:$|T|\s)`)
	if m := isoPattern.FindStringSubmatch(trimmed); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if err := validateCalendarDate(y, time.Month(mo), d); err != nil {
			return 0, 0, 0, err
		}
		return y, time.Month(mo), d, nil
	}

	dmyPattern := regexp.MustCompile(`^(\d{1,2})[\/\-](\d{1,2})[\/\-](\d{4})$`)
	if m := dmyPattern.FindStringSubmatch(trimmed); m != nil {
		d, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[3])
		if err := validateCalendarDate(y, time.Month(mo), d); err != nil {
			return 0, 0, 0, err
		}
		return y, time.Month(mo), d, nil
	}

	cleaned := strings.ToLower(trimmed)
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[idx+1:])
	}
	if idx := strings.Index(cleaned, "@"); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}

	parts := strings.Fields(cleaned)
	if len(parts) >= 3 {
		d, dayErr := strconv.Atoi(parts[0])
		moStr := strings.TrimSuffix(parts[1], ",")
		y, yearErr := strconv.Atoi(parts[2])
		mo, monthFound := monthAliases[moStr]
		if dayErr == nil && yearErr == nil && monthFound {
			if err := validateCalendarDate(y, mo, d); err != nil {
				return 0, 0, 0, err
			}
			return y, mo, d, nil
		}
	}

	return 0, 0, 0, fmt.Errorf("format tanggal konser tidak dikenali: %s", dateStr)
}

func validateCalendarDate(year int, month time.Month, day int) error {
	if year < 1900 || year > 2100 || month < time.January || month > time.December || day < 1 || day > 31 {
		return fmt.Errorf("tanggal di luar rentang kalender yang valid")
	}
	dt := time.Date(year, month, day, 0, 0, 0, 0, wibLocation)
	if dt.Year() != year || dt.Month() != month || dt.Day() != day {
		return fmt.Errorf("tanggal tidak ada dalam kalender")
	}
	return nil
}

// NormalizeDateString menormalkan tanggal ke format kanonik "<Hari>, <D> <Bulan> <YYYY>" dengan menghitung ulang nama hari.
func NormalizeDateString(raw string) (string, error) {
	y, m, d, err := parseDateParts(raw)
	if err != nil {
		return "", err
	}
	dt := time.Date(y, m, d, 0, 0, 0, 0, wibLocation)
	weekdayName := indonesianWeekdays[dt.Weekday()]
	monthName := indonesianMonthNames[m]
	return fmt.Sprintf("%s, %d %s %d", weekdayName, d, monthName, y), nil
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

// ParseConcertDateTime menggabungkan tanggal dan waktu konser menjadi objek time.Time sesuai zona waktu pada timeStr.
func ParseConcertDateTime(dateStr, timeStr string) (time.Time, error) {
	trimmedDate := strings.TrimSpace(dateStr)
	if trimmedDate == "" {
		return time.Time{}, fmt.Errorf("tanggal konser kosong")
	}

	if t, err := time.Parse(time.RFC3339, trimmedDate); err == nil {
		return t.In(wibLocation), nil
	}

	y, m, d, err := parseDateParts(trimmedDate)
	if err != nil {
		return time.Time{}, err
	}

	minutesOfDay, tz, ok := ParseTimeWithZone(timeStr)
	if !ok {
		minutesOfDay = 19*60 + 30
		tz = "WIB"
	}
	hour := minutesOfDay / 60
	minute := minutesOfDay % 60
	loc := locationForZone(tz)

	return time.Date(y, m, d, hour, minute, 0, 0, loc), nil
}

// ParseOpenGateDateTime menggabungkan tanggal konser dengan jam open gate (WIB UTC+7).
func ParseOpenGateDateTime(dateStr, openGateStr string) (time.Time, error) {
	return ParseConcertDateTime(dateStr, openGateStr)
}

