package controllers_test

import (
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

func TestPromotion_CalculateDiscount(t *testing.T) {
	t.Run("Menghitung diskon persentase dengan dan tanpa batas maksimal", func(t *testing.T) {
		promo := models.Promotion{
			Code:         "SYMPHONIA10",
			DiscountType: "PERCENTAGE",
			DiscountVal:  10,
			MaxDiscount:  50000,
		}
		if got := promo.CalculateDiscount(400000); got != 40000 {
			t.Fatalf("expected discount 40000, got %v", got)
		}
		if got := promo.CalculateDiscount(1000000); got != 50000 {
			t.Fatalf("expected capped discount 50000, got %v", got)
		}
	})

	t.Run("Menghitung diskon nominal tetap dan membatasi tidak melebihi subtotal", func(t *testing.T) {
		promo := models.Promotion{
			Code:         "HARMONY50K",
			DiscountType: "FIXED",
			DiscountVal:  50000,
		}
		if got := promo.CalculateDiscount(250000); got != 50000 {
			t.Fatalf("expected fixed discount 50000, got %v", got)
		}
		if got := promo.CalculateDiscount(30000); got != 30000 {
			t.Fatalf("expected discount capped to subtotal 30000, got %v", got)
		}
	})
}
