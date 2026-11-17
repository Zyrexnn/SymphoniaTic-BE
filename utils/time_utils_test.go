package utils_test

import (
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/utils"
)

func TestParseTimeToMinutes(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected int
		ok       bool
	}{
		{"24h standar dengan WIB", "14:21 WIB", 14*60 + 21, true},
		{"Notasi titik Indonesia dengan WIB", "14.21 WIB", 14*60 + 21, true},
		{"Jam satu digit", "9:00", 9 * 60, true},
		{"Jam dua digit tanpa zona", "09:00", 9 * 60, true},
		{"Format 12 jam PM dengan spasi", "02:21 PM", 14*60 + 21, true},
		{"Format 12 jam pm tanpa spasi", "2:21pm", 14*60 + 21, true},
		{"Format 12 jam tengah malam", "12:00 AM", 0, true},
		{"Format 12 jam tengah hari", "12:00 PM", 12 * 60, true},
		{"Format 24h dengan detik dan WITA", "19:30:00 WITA", 19*60 + 30, true},
		{"Format 24h batas akhir hari", "23:59 WIT", 23*60 + 59, true},
		{"String kosong", "", 0, false},
		{"Teks non waktu", "Sesuai Jadwal", 0, false},
		{"Jam di luar batas 24h", "25:00", 0, false},
		{"Menit di luar batas", "14:60", 0, false},
		{"Jam 12h tidak valid (13 PM)", "13:00 PM", 0, false},
		{"Jam 12h tidak valid (0 AM)", "0:30 AM", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := utils.ParseTimeToMinutes(tc.input)
			if ok != tc.ok {
				t.Fatalf("ParseTimeToMinutes(%q) ok = %v, ekspektasi %v", tc.input, ok, tc.ok)
			}
			if got != tc.expected {
				t.Fatalf("ParseTimeToMinutes(%q) = %d, ekspektasi %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsTimeBefore(t *testing.T) {
	cases := []struct {
		name         string
		t1           string
		t2           string
		expectValid  bool
		expectBefore bool
	}{
		{"Open gate lebih awal dari konser", "13:21 WIB", "14:21 WIB", true, true},
		{"Open gate format titik lebih awal dari format 12 jam", "13.00 WIB", "02:21 PM", true, true},
		{"Open gate sama dengan waktu konser", "14:21 WIB", "14:21", true, false},
		{"Open gate lebih lambat dari waktu konser", "18:00 WIB", "14:21 WIB", true, false},
		{"Salah satu tidak dapat diparsing", "Sesuai Jadwal", "14:21 WIB", false, false},
		{"Keduanya kosong", "", "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, before := utils.IsTimeBefore(tc.t1, tc.t2)
			if valid != tc.expectValid || before != tc.expectBefore {
				t.Fatalf("IsTimeBefore(%q, %q) = (%v, %v), ekspektasi (%v, %v)",
					tc.t1, tc.t2, valid, before, tc.expectValid, tc.expectBefore)
			}
		})
	}
}

func TestParseConcertDateTime(t *testing.T) {
	cases := []struct {
		name       string
		dateStr    string
		timeStr    string
		expectYear int
		expectMonth int
		expectDay  int
		expectHour int
		expectMin  int
		expectErr  bool
	}{
		{
			name:        "Tanggal Indonesia lengkap dengan hari",
			dateStr:     "Sabtu, 18 April 2026",
			timeStr:     "19:30 WIB",
			expectYear:  2026,
			expectMonth: 4,
			expectDay:   18,
			expectHour:  19,
			expectMin:   30,
			expectErr:   false,
		},
		{
			name:        "Tanggal tanpa nama hari",
			dateStr:     "22 Agustus 2026",
			timeStr:     "20:00",
			expectYear:  2026,
			expectMonth: 8,
			expectDay:   22,
			expectHour:  20,
			expectMin:   0,
			expectErr:   false,
		},
		{
			name:        "Format ISO yyyy-mm-dd",
			dateStr:     "2026-05-02",
			timeStr:     "17:30 WIB",
			expectYear:  2026,
			expectMonth: 5,
			expectDay:   2,
			expectHour:  17,
			expectMin:   30,
			expectErr:   false,
		},
		{
			name:        "Format tanggal tidak valid",
			dateStr:     "Tanggal Rahasia",
			timeStr:     "19:30",
			expectErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dt, err := utils.ParseConcertDateTime(tc.dateStr, tc.timeStr)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("Ekspektasi error untuk (%q, %q), tapi nil", tc.dateStr, tc.timeStr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConcertDateTime(%q, %q) gagal: %v", tc.dateStr, tc.timeStr, err)
			}
			if dt.Year() != tc.expectYear || int(dt.Month()) != tc.expectMonth || dt.Day() != tc.expectDay ||
				dt.Hour() != tc.expectHour || dt.Minute() != tc.expectMin {
				t.Fatalf("Hasil tidak sesuai: %v", dt)
			}
		})
	}
}

