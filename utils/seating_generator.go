package utils

import (
	"fmt"
	"math"
	"sort"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/google/uuid"
)

const (
	hallStageCX        = 230.0
	hallStageCY        = 308.0
	hallStageClearance = 96.0
)

var standardRowLabels = []string{"K", "J", "I", "H", "G", "F", "E", "D", "C", "B", "A"}

type hallSlot struct {
	SeatCode   string
	RowLabel   string
	SeatNumber int
	X          float64
	Y          float64
	Dist       float64
	Score      float64
	ColIdx     int
}

func rowLabelForExtraIndex(extraIdx int) string {
	// Baris tambahan dimulai dari 'L' (indeks 0 -> 'L', 1 -> 'M', dst.) di belakang baris 'K'
	code := int('L') + extraIdx
	if code <= int('Z') {
		return string(rune(code))
	}
	offset := code - int('Z') - 1
	return fmt.Sprintf("R%d", offset+1)
}

func buildOrderedHallSlots(minCapacity int) []hallSlot {
	slots := make([]hallSlot, 0, maxInt(minCapacity, 138))

	for rowIdx := 0; rowIdx < len(standardRowLabels); rowIdx++ {
		y := 32.0 + float64(rowIdx)*26.0
		rowLabel := standardRowLabels[rowIdx]
		rowFromFront := len(standardRowLabels) - 1 - rowIdx
		seatCounter := 0

		for colIdx := 0; colIdx < 14; colIdx++ {
			var x float64
			if colIdx < 7 {
				x = 209.0 - float64(6-colIdx)*26.0
			} else {
				x = 251.0 + float64(colIdx-7)*26.0
			}

			dist := math.Hypot(x-hallStageCX, y-hallStageCY)
			if dist < hallStageClearance {
				continue
			}

			seatCounter++
			score := dist + float64(rowFromFront)*10.0
			slots = append(slots, hallSlot{
				SeatCode:   fmt.Sprintf("%s-%d", rowLabel, seatCounter),
				RowLabel:   rowLabel,
				SeatNumber: seatCounter,
				X:          x,
				Y:          y,
				Dist:       dist,
				Score:      score,
				ColIdx:     colIdx,
			})
		}
	}

	extraRowIdx := 0
	for len(slots) < minCapacity {
		y := 32.0 - float64(extraRowIdx+1)*26.0
		rowLabel := rowLabelForExtraIndex(extraRowIdx)
		rowFromFront := len(standardRowLabels) + extraRowIdx

		for colIdx := 0; colIdx < 14; colIdx++ {
			var x float64
			if colIdx < 7 {
				x = 209.0 - float64(6-colIdx)*26.0
			} else {
				x = 251.0 + float64(colIdx-7)*26.0
			}
			dist := math.Hypot(x-hallStageCX, y-hallStageCY)
			score := dist + float64(rowFromFront)*10.0
			seatNum := colIdx + 1
			slots = append(slots, hallSlot{
				SeatCode:   fmt.Sprintf("%s-%d", rowLabel, seatNum),
				RowLabel:   rowLabel,
				SeatNumber: seatNum,
				X:          x,
				Y:          y,
				Dist:       dist,
				Score:      score,
				ColIdx:     colIdx,
			})
		}
		extraRowIdx++
	}

	sort.SliceStable(slots, func(i, j int) bool {
		si := math.Round(slots[i].Score * 100)
		sj := math.Round(slots[j].Score * 100)
		if si != sj {
			return si < sj
		}
		ci := math.Abs(slots[i].X - hallStageCX)
		cj := math.Abs(slots[j].X - hallStageCX)
		if math.Abs(ci-cj) > 0.01 {
			return ci < cj
		}
		return slots[i].ColIdx < slots[j].ColIdx
	})

	return slots
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GenerateHallSeats memetakan seluruh kuota kategori tiket sebuah pertunjukan ke nomor kursi fisik secara berurutan dari kelas tertinggi (terdekat ke panggung).
func GenerateHallSeats(eventID string, categories []models.TicketCategory) []models.SeatItem {
	if len(categories) == 0 {
		return nil
	}

	sortedCats := make([]models.TicketCategory, len(categories))
	copy(sortedCats, categories)
	sort.SliceStable(sortedCats, func(i, j int) bool {
		if sortedCats[i].Price != sortedCats[j].Price {
			return sortedCats[i].Price > sortedCats[j].Price
		}
		return sortedCats[i].ID < sortedCats[j].ID
	})

	totalQuota := 0
	for _, cat := range sortedCats {
		if cat.Quota > 0 {
			totalQuota += cat.Quota
		}
	}
	if totalQuota == 0 {
		return nil
	}

	slots := buildOrderedHallSlots(totalQuota)
	seats := make([]models.SeatItem, 0, totalQuota)
	cursor := 0

	for _, cat := range sortedCats {
		for q := 0; q < cat.Quota && cursor < len(slots); q++ {
			slot := slots[cursor]
			cursor++
			seats = append(seats, models.SeatItem{
				ID:         uuid.New().String(),
				EventID:    eventID,
				CategoryID: cat.ID,
				SeatCode:   slot.SeatCode,
				RowLabel:   slot.RowLabel,
				SeatNumber: slot.SeatNumber,
				Status:     "AVAILABLE",
			})
		}
	}

	return seats
}

// GenerateAdditionalCategorySeats membuat kursi baru dengan menghindari kode kursi yang sudah dipakai pada konser terkait.
func GenerateAdditionalCategorySeats(eventID, categoryID string, count int, existingSeatCodes []string) []models.SeatItem {
	if count <= 0 {
		return nil
	}

	existingSet := make(map[string]bool, len(existingSeatCodes))
	for _, code := range existingSeatCodes {
		existingSet[code] = true
	}

	slots := buildOrderedHallSlots(len(existingSeatCodes) + count + 14)
	added := make([]models.SeatItem, 0, count)

	for _, slot := range slots {
		if existingSet[slot.SeatCode] {
			continue
		}
		added = append(added, models.SeatItem{
			ID:         uuid.New().String(),
			EventID:    eventID,
			CategoryID: categoryID,
			SeatCode:   slot.SeatCode,
			RowLabel:   slot.RowLabel,
			SeatNumber: slot.SeatNumber,
			Status:     "AVAILABLE",
		})
		if len(added) == count {
			break
		}
	}

	return added
}
