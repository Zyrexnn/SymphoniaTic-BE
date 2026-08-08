package utils

import (
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/gofiber/fiber/v2"
)

// SendJSON sends a structured APIResponse JSON.
func SendJSON(ctx *fiber.Ctx, statusCode int, success bool, message string, data interface{}, errMsg string) error {
	return ctx.Status(statusCode).JSON(models.APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   errMsg,
	})
}

// ResponseOK sends HTTP 200 OK.
func ResponseOK(ctx *fiber.Ctx, message string, data interface{}) error {
	return SendJSON(ctx, fiber.StatusOK, true, message, data, "")
}

// ResponseCreated sends HTTP 201 Created.
func ResponseCreated(ctx *fiber.Ctx, message string, data interface{}) error {
	return SendJSON(ctx, fiber.StatusCreated, true, message, data, "")
}

// ResponseBadRequest sends HTTP 400 Bad Request.
func ResponseBadRequest(ctx *fiber.Ctx, message string) error {
	return SendJSON(ctx, fiber.StatusBadRequest, false, message, nil, "")
}

// ResponseUnauthorized sends HTTP 401 Unauthorized.
func ResponseUnauthorized(ctx *fiber.Ctx, message string) error {
	return SendJSON(ctx, fiber.StatusUnauthorized, false, message, nil, "")
}

// ResponseNotFound sends HTTP 404 Not Found.
func ResponseNotFound(ctx *fiber.Ctx, message string) error {
	return SendJSON(ctx, fiber.StatusNotFound, false, message, nil, "")
}

// ResponseTooManyRequests sends HTTP 429 Too Many Requests.
func ResponseTooManyRequests(ctx *fiber.Ctx, message string) error {
	return SendJSON(ctx, fiber.StatusTooManyRequests, false, message, nil, "")
}

// ResponseInternalError sends HTTP 500 Internal Server Error.
func ResponseInternalError(ctx *fiber.Ctx, message string, optionalErrors ...error) error {
	errMsg := ""
	if len(optionalErrors) > 0 && optionalErrors[0] != nil {
		errMsg = optionalErrors[0].Error()
	}
	return SendJSON(ctx, fiber.StatusInternalServerError, false, message, nil, errMsg)
}
