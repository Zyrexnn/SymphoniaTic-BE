package utils

import (
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/gofiber/fiber/v2"
)

// SendJSON sends a structured APIResponse JSON.
func SendJSON(c *fiber.Ctx, statusCode int, success bool, message string, data interface{}, err string) error {
	return c.Status(statusCode).JSON(models.APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   err,
	})
}

// ResponseOK sends HTTP 200 OK.
func ResponseOK(c *fiber.Ctx, message string, data interface{}) error {
	return SendJSON(c, fiber.StatusOK, true, message, data, "")
}

// ResponseCreated sends HTTP 201 Created.
func ResponseCreated(c *fiber.Ctx, message string, data interface{}) error {
	return SendJSON(c, fiber.StatusCreated, true, message, data, "")
}

// ResponseBadRequest sends HTTP 400 Bad Request.
func ResponseBadRequest(c *fiber.Ctx, message string) error {
	return SendJSON(c, fiber.StatusBadRequest, false, message, nil, "")
}

// ResponseUnauthorized sends HTTP 401 Unauthorized.
func ResponseUnauthorized(c *fiber.Ctx, message string) error {
	return SendJSON(c, fiber.StatusUnauthorized, false, message, nil, "")
}

// ResponseNotFound sends HTTP 404 Not Found.
func ResponseNotFound(c *fiber.Ctx, message string) error {
	return SendJSON(c, fiber.StatusNotFound, false, message, nil, "")
}

// ResponseTooManyRequests sends HTTP 429 Too Many Requests.
func ResponseTooManyRequests(c *fiber.Ctx, message string) error {
	return SendJSON(c, fiber.StatusTooManyRequests, false, message, nil, "")
}

// ResponseInternalError sends HTTP 500 Internal Server Error.
func ResponseInternalError(c *fiber.Ctx, message string, optErr ...error) error {
	errMsg := ""
	if len(optErr) > 0 && optErr[0] != nil {
		errMsg = optErr[0].Error()
	}
	return SendJSON(c, fiber.StatusInternalServerError, false, message, nil, errMsg)
}
