package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

type OrderController struct {
	repo repositories.OrderRepository
}

func NewOrderController(repo repositories.OrderRepository) *OrderController {
	return &OrderController{repo: repo}
}

// CreateOrder godoc
// @Summary Membeli tiket / Membuat pesanan
// @Description Membuat pesanan tiket baru untuk event tertentu baik sebagai Guest maupun Logged-in User.
// @Tags Public - Orders
// @Accept json
// @Produce json
// @Param payload body models.CreateOrderRequest true "Order Request Payload"
// @Success 201 {object} models.APIResponse{data=models.OrderRecord} "Pemesanan tiket berhasil!"
// @Failure 400 {object} models.APIResponse "Payload tidak valid / Quota habis"
// @Failure 500 {object} models.APIResponse "Gagal memproses pemesanan"
// @Router /orders [post]
func (c *OrderController) CreateOrder(ctx *fiber.Ctx) error {
	var createOrderReq models.CreateOrderRequest
	if err := ctx.BodyParser(&createOrderReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	var authenticatedUserID string
	rawAuthHeader := ctx.Get("Authorization")
	if rawAuthHeader != "" && strings.HasPrefix(rawAuthHeader, "Bearer ") {
		rawBearerToken := strings.TrimPrefix(rawAuthHeader, "Bearer ")
		claims, err := utils.ValidateUserToken(rawBearerToken)
		if err == nil && claims != nil {
			authenticatedUserID = claims.UserID
			if createOrderReq.UserEmail == "" {
				createOrderReq.UserEmail = claims.Email
			}
		}
	}

	if (createOrderReq.UserName == "" && authenticatedUserID == "") || createOrderReq.UserEmail == "" || createOrderReq.TicketCategoryID == "" {
		return utils.ResponseBadRequest(ctx, "Nama, Email, dan Kategori Tiket wajib diisi")
	}

	if createOrderReq.Quantity < 1 || createOrderReq.Quantity > 4 {
		return utils.ResponseBadRequest(ctx, "Maksimal pemesanan adalah 1 hingga 4 tiket per transaksi")
	}

	orderRecord, err := c.repo.CreateOrder(ctx.Context(), &createOrderReq, authenticatedUserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.ResponseNotFound(ctx, "Kategori tiket tidak ditemukan")
		}
		if errors.Is(err, repositories.ErrEventClosed) {
			return utils.ResponseBadRequest(ctx, "Penjualan tiket untuk pertunjukan ini telah ditutup karena konser sudah dimulai.")
		}
		if errors.Is(err, repositories.ErrQuotaExceeded) {
			return utils.ResponseBadRequest(ctx, err.Error())
		}
		return utils.ResponseInternalError(ctx, "Gagal membuat pesanan tiket", err)
	}

	services.SendETicketEmail(*orderRecord)

	return utils.ResponseCreated(ctx, "Simulasi Pembayaran Sandbox Berhasil & E-Ticket Terbit!", *orderRecord)
}

// GetAllOrders godoc
// @Summary Mengambil semua pesanan tiket (Admin)
// @Description Mengambil riwayat seluruh pesanan dengan filter pencarian teks dan status.
// @Tags Admin - Orders Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param search query string false "Search query (order code, name, email, event)"
// @Param status query string false "Filter by order status"
// @Success 200 {object} models.APIResponse{data=[]models.OrderRecord} "Daftar pesanan berhasil diambil"
// @Router /admin/orders [get]
func (c *OrderController) GetAllOrders(ctx *fiber.Ctx) error {
	searchQuery := ctx.Query("search")
	statusFilter := ctx.Query("status")

	allOrders, err := c.repo.GetAllOrders(ctx.Context(), searchQuery, statusFilter)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal mengambil daftar pesanan",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Daftar pesanan berhasil diambil",
		Data:    allOrders,
	})
}

// UpdateOrderStatus godoc
// @Summary Memperbarui status pesanan tiket (Admin)
// @Description Memperbarui status pesanan tiket (misal: CHECKED_IN, REMINDED, REFUNDED) oleh admin atau petugas.
// @Tags Admin - Orders Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param payload body models.UpdateOrderStatusRequest true "Update Order Status Payload"
// @Success 200 {object} models.APIResponse "Status pesanan berhasil diperbarui"
// @Failure 404 {object} models.APIResponse "Pesanan tidak ditemukan"
// @Router /admin/orders/{id}/status [patch]
func (c *OrderController) UpdateOrderStatus(ctx *fiber.Ctx) error {
	orderID := ctx.Params("id")
	var updateStatusReq models.UpdateOrderStatusRequest
	if err := ctx.BodyParser(&updateStatusReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if updateStatusReq.Status == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Status pesanan wajib diisi",
		})
	}

	err := c.repo.UpdateOrderStatus(ctx.Context(), orderID, updateStatusReq.Status)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Pesanan tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui status pesanan",
			Error:   err.Error(),
		})
	}

	// Mengirimkan notifikasi email pengingat acara jika status diubah menjadi REMINDED atau REMINDER
	if strings.ToUpper(updateStatusReq.Status) == "REMINDED" || strings.ToUpper(updateStatusReq.Status) == "REMINDER" {
		ord, fetchErr := c.repo.GetOrderByID(ctx.Context(), orderID)
		if fetchErr == nil && ord != nil {
			services.SendEventReminderEmail(ord.UserEmail, ord.UserName, ord.EventTitle, ord.Venue, "", ord.Date, ord.OrderCode)
		}
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Status pesanan berhasil diperbarui menjadi %s", updateStatusReq.Status),
	})
}
