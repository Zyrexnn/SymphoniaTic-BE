package utils

import (
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

func TestGenerateHallSeats_StandardAndOverflow(t *testing.T) {
	categories := []models.TicketCategory{
		{ID: "cat-fest", EventID: "evt-1", Name: "Festival Stalls", Price: 300000, Quota: 50},
		{ID: "cat-vip", EventID: "evt-1", Name: "VIP Orchestral Pit", Price: 750000, Quota: 14},
		{ID: "cat-cat1", EventID: "evt-1", Name: "CAT 1 Grand Tier", Price: 450000, Quota: 24},
	}

	seats := GenerateHallSeats("evt-1", categories)
	expectedTotal := 50 + 14 + 24
	if len(seats) != expectedTotal {
		t.Fatalf("ekspektasi %d kursi, didapat %d", expectedTotal, len(seats))
	}

	seenCodes := make(map[string]bool)
	countsByCat := make(map[string]int)
	vipHasFrontRow := false

	for _, s := range seats {
		if seenCodes[s.SeatCode] {
			t.Fatalf("ditemukan kode kursi duplikat: %s", s.SeatCode)
		}
		seenCodes[s.SeatCode] = true
		countsByCat[s.CategoryID]++
		if s.Status != "AVAILABLE" {
			t.Fatalf("status awal kursi wajib AVAILABLE, didapat: %s", s.Status)
		}
		if s.CategoryID == "cat-vip" && (s.RowLabel == "A" || s.RowLabel == "B" || s.RowLabel == "C" || s.RowLabel == "D") {
			vipHasFrontRow = true
		}
	}

	if countsByCat["cat-vip"] != 14 || countsByCat["cat-cat1"] != 24 || countsByCat["cat-fest"] != 50 {
		t.Fatalf("distribusi jumlah kursi per kategori tidak sesuai kuota: %+v", countsByCat)
	}
	if !vipHasFrontRow {
		t.Fatal("kursi VIP harus ditempatkan pada baris depan (A-D)")
	}
}

func TestGenerateHallSeats_OverflowBeyondStandardHall(t *testing.T) {
	categories := []models.TicketCategory{
		{ID: "cat-vip", EventID: "evt-big", Name: "VIP", Price: 1000000, Quota: 20},
		{ID: "cat-reg", EventID: "evt-big", Name: "Regular", Price: 400000, Quota: 150},
	}

	seats := GenerateHallSeats("evt-big", categories)
	if len(seats) != 170 {
		t.Fatalf("ekspektasi 170 kursi untuk hall overflow, didapat %d", len(seats))
	}

	hasRowLOrHigher := false
	seen := make(map[string]bool)
	for _, s := range seats {
		if seen[s.SeatCode] {
			t.Fatalf("kode kursi duplikat pada overflow: %s", s.SeatCode)
		}
		seen[s.SeatCode] = true
		if s.RowLabel == "L" || s.RowLabel == "M" {
			hasRowLOrHigher = true
		}
	}

	if !hasRowLOrHigher {
		t.Fatal("kursi overflow > 138 harus menggunakan label baris tambahan (L, M, ...)")
	}
}

func TestGenerateAdditionalCategorySeats_SkipsExisting(t *testing.T) {
	existing := []string{"A-1", "A-2", "B-1", "B-2"}
	added := GenerateAdditionalCategorySeats("evt-1", "cat-new", 5, existing)

	if len(added) != 5 {
		t.Fatalf("ekspektasi 5 kursi tambahan, didapat %d", len(added))
	}

	existingSet := map[string]bool{"A-1": true, "A-2": true, "B-1": true, "B-2": true}
	for _, s := range added {
		if existingSet[s.SeatCode] {
			t.Fatalf("kursi tambahan tidak boleh menabrak kursi yang sudah ada: %s", s.SeatCode)
		}
	}
}
