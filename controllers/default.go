package controllers

import (
	"database/sql"
	"sync"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
)

var (
	defaultMu           sync.RWMutex
	defaultEventCtrl    *EventController
	defaultCategoryCtrl *CategoryController
	defaultOrderCtrl    *OrderController
	defaultTicketCtrl   *TicketController
	defaultRefundCtrl   *RefundController
	defaultAdminCtrl    *AdminController
)

func InitDefaultControllers(db *sql.DB, authCfg *config.AuthConfig) {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	eventRepo := repositories.NewEventRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	refundRepo := repositories.NewRefundRepository(db)

	defaultEventCtrl = NewEventController(eventRepo)
	defaultCategoryCtrl = NewCategoryController(eventRepo)
	defaultOrderCtrl = NewOrderController(orderRepo)
	defaultTicketCtrl = NewTicketController(orderRepo)
	defaultRefundCtrl = NewRefundControllerWithOrderRepo(refundRepo, orderRepo)
	defaultAdminCtrl = NewAdminController(orderRepo, authCfg)
}

func ensureDefaultControllers() {
	defaultMu.RLock()
	if defaultEventCtrl != nil {
		defaultMu.RUnlock()
		return
	}
	defaultMu.RUnlock()

	if database.DB != nil {
		InitDefaultControllers(database.DB, nil)
	}
}

func GetEvents(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.GetEvents(ctx)
}

func GetEventByID(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.GetEventByID(ctx)
}

func CreateEvent(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.CreateEvent(ctx)
}

func UpdateEvent(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.UpdateEvent(ctx)
}

func DeleteEvent(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.DeleteEvent(ctx)
}

func ToggleEventClose(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultEventCtrl.ToggleEventClose(ctx)
}

func CreateTicketCategory(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultCategoryCtrl.CreateTicketCategory(ctx)
}

func UpdateTicketCategory(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultCategoryCtrl.UpdateTicketCategory(ctx)
}

func DeleteTicketCategory(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultCategoryCtrl.DeleteTicketCategory(ctx)
}

func CreateOrder(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultOrderCtrl.CreateOrder(ctx)
}

func GetAllOrders(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultOrderCtrl.GetAllOrders(ctx)
}

func UpdateOrderStatus(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultOrderCtrl.UpdateOrderStatus(ctx)
}

func LookupTicketByCode(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultTicketCtrl.LookupTicketByCode(ctx)
}

func RequestRefundOTP(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultRefundCtrl.RequestRefundOTP(ctx)
}

func SubmitRefund(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultRefundCtrl.SubmitRefund(ctx)
}

func GetRefundStatus(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultRefundCtrl.GetRefundStatus(ctx)
}

func AdminGetAllRefunds(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultRefundCtrl.AdminGetAllRefunds(ctx)
}

func AdminUpdateRefundStatus(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultRefundCtrl.AdminUpdateRefundStatus(ctx)
}

func AdminLogin(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultAdminCtrl.AdminLogin(ctx)
}

func GetAdminDashboardMetrics(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultAdminCtrl.GetAdminDashboardMetrics(ctx)
}

func UploadImage(ctx *fiber.Ctx) error {
	ensureDefaultControllers()
	return defaultAdminCtrl.UploadImage(ctx)
}
