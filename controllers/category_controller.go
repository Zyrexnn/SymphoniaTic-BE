package controllers

import (
	"errors"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

type CategoryController struct {
	repo repositories.EventRepository
}

func NewCategoryController(repo repositories.EventRepository) *CategoryController {
	return &CategoryController{repo: repo}
}

// CreateTicketCategory godoc
// @Summary Menambahkan kategori tiket baru ke konser (Admin)
// @Description Menambahkan kelas tiket baru (misal: VIP, Regular) dengan kuota dan harga spesifik ke konser.
// @Tags Admin - Ticket Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param payload body models.CreateCategoryInput true "Create Category Payload"
// @Success 201 {object} models.APIResponse{data=models.TicketCategory} "Kategori tiket berhasil ditambahkan"
// @Failure 400 {object} models.APIResponse "Nama, Harga, dan Kuota tiket wajib valid"
// @Router /admin/events/{id}/categories [post]
func (c *CategoryController) CreateTicketCategory(ctx *fiber.Ctx) error {
	eventID := ctx.Params("id")
	var createCategoryReq models.CreateCategoryInput
	if err := ctx.BodyParser(&createCategoryReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	if createCategoryReq.Name == "" || createCategoryReq.Price <= 0 || createCategoryReq.Quota <= 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Nama, Harga, dan Kuota tiket wajib valid",
		})
	}

	createdCat, err := c.repo.AddCategory(ctx.Context(), eventID, &createCategoryReq)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menambah kategori tiket",
			Error:   err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil ditambahkan",
		Data:    createdCat,
	})
}

// UpdateTicketCategory godoc
// @Summary Memperbarui kategori tiket konser (Admin)
// @Description Memperbarui nama, harga, dan kuota dari kategori tiket konser yang sudah ada.
// @Tags Admin - Ticket Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param payload body models.UpdateTicketCategoryRequest true "Update Category Payload"
// @Success 200 {object} models.APIResponse "Kategori tiket berhasil diperbarui"
// @Failure 404 {object} models.APIResponse "Kategori tiket tidak ditemukan"
// @Router /admin/categories/{id} [put]
func (c *CategoryController) UpdateTicketCategory(ctx *fiber.Ctx) error {
	catID := ctx.Params("id")
	var updateCategoryReq models.UpdateTicketCategoryRequest
	if err := ctx.BodyParser(&updateCategoryReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Format request tidak valid",
			Error:   err.Error(),
		})
	}

	err := c.repo.UpdateCategory(ctx.Context(), catID, &updateCategoryReq)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kategori tiket tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal memperbarui kategori tiket",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil diperbarui",
	})
}

// DeleteTicketCategory godoc
// @Summary Menghapus kategori tiket konser (Admin)
// @Description Menghapus satu kategori tiket konser berdasarkan ID kategori.
// @Tags Admin - Ticket Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.APIResponse "Kategori tiket berhasil dihapus"
// @Failure 404 {object} models.APIResponse "Kategori tiket tidak ditemukan"
// @Router /admin/categories/{id} [delete]
func (c *CategoryController) DeleteTicketCategory(ctx *fiber.Ctx) error {
	catID := ctx.Params("id")
	err := c.repo.DeleteCategory(ctx.Context(), catID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ctx.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "Kategori tiket tidak ditemukan",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Gagal menghapus kategori tiket",
			Error:   err.Error(),
		})
	}

	return ctx.JSON(models.APIResponse{
		Success: true,
		Message: "Kategori tiket berhasil dihapus",
	})
}
