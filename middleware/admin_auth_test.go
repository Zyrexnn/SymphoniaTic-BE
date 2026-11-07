package middleware_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
)

func parseResponseBody(t *testing.T, resp *http.Response) models.APIResponse {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Gagal membaca body response: %v", err)
	}

	var apiResp models.APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		t.Fatalf("Gagal unmarshal JSON response (%s): %v", string(body), err)
	}
	return apiResp
}

func setupTestApp(allowedRoles ...string) *fiber.App {
	app := fiber.New()
	app.Get("/admin/test-resource", middleware.RequireAdminAuth(allowedRoles...), func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(models.APIResponse{
			Success: true,
			Message: "Akses resource berhasil",
			Data: fiber.Map{
				"admin_id":    c.Locals("admin_id"),
				"admin_email": c.Locals("admin_email"),
				"admin_role":  c.Locals("admin_role"),
			},
		})
	})
	return app
}

func TestRequireAdminAuth_MissingHeader(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got: %d", resp.StatusCode)
	}

	body := parseResponseBody(t, resp)
	if body.Success {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestRequireAdminAuth_MalformedHeader(t *testing.T) {
	app := setupTestApp()

	malformedHeaders := []string{
		"Token12345",
		"Basic dXNlcjpwYXNz",
		"Bearer",
		"Bearer token extra parts",
	}

	for _, headerVal := range malformedHeaders {
		req := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
		req.Header.Set("Authorization", headerVal)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Gagal eksekusi test request: %v", err)
		}

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Header '%s': expected status 401 Unauthorized, got: %d", headerVal, resp.StatusCode)
		}

		body := parseResponseBody(t, resp)
		if body.Success {
			t.Errorf("Header '%s': expected success to be false, got true", headerVal)
		}
	}
}

func TestRequireAdminAuth_InvalidToken(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got: %d", resp.StatusCode)
	}

	body := parseResponseBody(t, resp)
	if body.Success {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestRequireAdminAuth_UserRoleForbidden(t *testing.T) {
	app := setupTestApp()

	userToken, err := utils.GenerateUserToken("usr-reg-001", "user@symphoniatic.id", "USER")
	if err != nil {
		t.Fatalf("Gagal membuat token user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden for role USER, got: %d", resp.StatusCode)
	}

	body := parseResponseBody(t, resp)
	if body.Success {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestRequireAdminAuth_AdminRoleSuccess(t *testing.T) {
	app := setupTestApp()

	adminToken, err := utils.GenerateUserToken("admin-1", "admin@symphoniatic.id", "ADMIN")
	if err != nil {
		t.Fatalf("Gagal membuat token admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 OK for role ADMIN, got: %d", resp.StatusCode)
	}

	body := parseResponseBody(t, resp)
	if !body.Success {
		t.Errorf("Expected success to be true, got false")
	}

	dataMap, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Data to be map, got: %T", body.Data)
	}
	if dataMap["admin_id"] != "admin-1" || dataMap["admin_email"] != "admin@symphoniatic.id" || dataMap["admin_role"] != "ADMIN" {
		t.Errorf("Locals mismatch: got %v", dataMap)
	}
}

func TestRequireAdminAuth_PetugasRoleWithAllowedRoles(t *testing.T) {
	petugasToken, err := utils.GenerateUserToken("petugas-1", "petugas@symphoniatic.id", "PETUGAS")
	if err != nil {
		t.Fatalf("Gagal membuat token petugas: %v", err)
	}

	// Case 1: Default role (ADMIN only) should deny PETUGAS
	appDefault := setupTestApp()
	reqDenied := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	reqDenied.Header.Set("Authorization", "Bearer "+petugasToken)
	respDenied, err := appDefault.Test(reqDenied)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}
	if respDenied.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden for PETUGAS on default roles, got: %d", respDenied.StatusCode)
	}

	// Case 2: Configured with ADMIN and PETUGAS roles should allow PETUGAS
	appWithPetugas := setupTestApp("ADMIN", "PETUGAS")
	reqAllowed := httptest.NewRequest(http.MethodGet, "/admin/test-resource", nil)
	reqAllowed.Header.Set("Authorization", "Bearer "+petugasToken)
	respAllowed, err := appWithPetugas.Test(reqAllowed)
	if err != nil {
		t.Fatalf("Gagal eksekusi test request: %v", err)
	}
	if respAllowed.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 OK for PETUGAS when role is allowed, got: %d", respAllowed.StatusCode)
	}

	bodyAllowed := parseResponseBody(t, respAllowed)
	if !bodyAllowed.Success {
		t.Errorf("Expected success to be true, got false")
	}

	dataMap, ok := bodyAllowed.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Data to be map, got: %T", bodyAllowed.Data)
	}
	if dataMap["admin_role"] != "PETUGAS" {
		t.Errorf("Expected admin_role PETUGAS, got: %v", dataMap["admin_role"])
	}
}
