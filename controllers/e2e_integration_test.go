package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func setupTestApp() *fiber.App {
	_ = godotenv.Load("../.env")
	database.ConnectDB()

	app := fiber.New()
	api := app.Group("/api/v1")

	// Public Endpoints
	api.Get("/events", controllers.GetEvents)
	api.Get("/events/:id", controllers.GetEventByID)
	api.Post("/orders", controllers.CreateOrder)
	api.Get("/tickets/lookup", controllers.LookupTicketByCode)

	// Auth Endpoints
	auth := api.Group("/auth")
	auth.Post("/register/request-otp", controllers.RequestRegisterOTP)
	auth.Post("/register/verify-otp", controllers.VerifyRegisterOTP)
	auth.Post("/login/password", controllers.PasswordLogin)
	auth.Post("/login/request-otp", controllers.RequestLoginOTP)
	auth.Post("/login/verify-otp", controllers.VerifyLoginOTP)
	auth.Post("/forgot-password/request-otp", controllers.RequestForgotPasswordOTP)
	auth.Post("/forgot-password/verify-otp", controllers.VerifyForgotPasswordOTP)
	auth.Post("/forgot-password/reset", controllers.ResetPassword)
	auth.Get("/me", middleware.RequireUserAuth, controllers.GetMyProfile)

	// User Endpoints
	userGroup := api.Group("/user", middleware.RequireUserAuth)
	userGroup.Get("/orders", controllers.GetUserOrders)
	userGroup.Get("/orders/:orderCode", controllers.GetUserOrderByCode)
	userGroup.Put("/profile", controllers.UpdateUserProfile)
	userGroup.Post("/change-password", controllers.ChangeUserPassword)
	userGroup.Get("/dashboard-summary", controllers.GetUserDashboardSummary)
	userGroup.Get("/refunds", controllers.GetUserRefunds)

	return app
}

func TestE2ECompleteUserFlow(t *testing.T) {
	app := setupTestApp()

	testEmail := fmt.Sprintf("testuser_%d@symphoniatic.com", time.Now().UnixNano())
	testName := "E2E Test User"
	testPassword := "InitialPass123!"

	// 1. GET /events (Verify batch loading works and returns 200)
	reqEvents := httptest.NewRequest("GET", "/api/v1/events", nil)
	respEvents, err := app.Test(reqEvents, -1)
	if err != nil || respEvents.StatusCode != http.StatusOK {
		t.Fatalf("GET /events failed with status %d: %v", respEvents.StatusCode, err)
	}

	// 2. Request OTP Registration
	regBody, _ := json.Marshal(map[string]string{
		"email": testEmail,
		"name":  testName,
	})
	reqRegOTP := httptest.NewRequest("POST", "/api/v1/auth/register/request-otp", bytes.NewBuffer(regBody))
	reqRegOTP.Header.Set("Content-Type", "application/json")
	respRegOTP, _ := app.Test(reqRegOTP, -1)
	if respRegOTP.StatusCode != http.StatusOK {
		t.Fatalf("RequestRegisterOTP failed with status: %d", respRegOTP.StatusCode)
	}

	// Query created OTP code directly from DB for verification test
	var otpCode string
	err = database.DB.QueryRow("SELECT otp_code FROM auth_otps WHERE email = $1 AND purpose = 'REGISTER' AND is_used = FALSE ORDER BY created_at DESC LIMIT 1", strings.ToLower(testEmail)).Scan(&otpCode)
	if err != nil || otpCode == "" {
		t.Fatalf("Failed to fetch OTP from database: %v", err)
	}

	// 3. Verify OTP & Register Account
	verifyBody, _ := json.Marshal(map[string]string{
		"email":    testEmail,
		"name":     testName,
		"otpCode":  otpCode,
		"password": testPassword,
	})
	reqVerify := httptest.NewRequest("POST", "/api/v1/auth/register/verify-otp", bytes.NewBuffer(verifyBody))
	reqVerify.Header.Set("Content-Type", "application/json")
	respVerify, _ := app.Test(reqVerify, -1)
	if respVerify.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(respVerify.Body)
		t.Fatalf("VerifyRegisterOTP failed with status %d: %s", respVerify.StatusCode, string(bodyBytes))
	}

	// Extract Auth Token
	bodyVerifyBytes, _ := io.ReadAll(respVerify.Body)
	var verifyRes models.APIResponse
	_ = json.Unmarshal(bodyVerifyBytes, &verifyRes)
	dataMap := verifyRes.Data.(map[string]interface{})
	token := dataMap["token"].(string)

	if token == "" {
		t.Fatalf("JWT Token is empty in register response")
	}

	// 4. Test GET /auth/me (Protected Route)
	reqMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	respMe, _ := app.Test(reqMe, -1)
	if respMe.StatusCode != http.StatusOK {
		t.Fatalf("GET /auth/me failed with status %d", respMe.StatusCode)
	}

	// 5. Test Password Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    testEmail,
		"password": testPassword,
	})
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login/password", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	respLogin, _ := app.Test(reqLogin, -1)
	if respLogin.StatusCode != http.StatusOK {
		t.Fatalf("PasswordLogin failed with status %d", respLogin.StatusCode)
	}

	// 6. Test Autofill Checkout (POST /orders with Bearer token, omitting name/email)
	orderBody, _ := json.Marshal(map[string]interface{}{
		"eventId":          "evt-1",
		"ticketCategoryId": "c1-vip",
		"quantity":         2,
	})
	reqOrder := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewBuffer(orderBody))
	reqOrder.Header.Set("Content-Type", "application/json")
	reqOrder.Header.Set("Authorization", "Bearer "+token)
	respOrder, _ := app.Test(reqOrder, -1)
	if respOrder.StatusCode != http.StatusCreated {
		orderBytes, _ := io.ReadAll(respOrder.Body)
		t.Fatalf("Autofill CreateOrder failed with status %d: %s", respOrder.StatusCode, string(orderBytes))
	}

	// Extract OrderCode
	orderRespBytes, _ := io.ReadAll(respOrder.Body)
	var orderAPIRes models.APIResponse
	_ = json.Unmarshal(orderRespBytes, &orderAPIRes)
	orderDataMap := orderAPIRes.Data.(map[string]interface{})
	orderCode := orderDataMap["orderCode"].(string)

	if orderCode == "" {
		t.Fatalf("OrderCode is empty")
	}

	// 7. Test User Order History (GET /user/orders)
	reqUserOrders := httptest.NewRequest("GET", "/api/v1/user/orders", nil)
	reqUserOrders.Header.Set("Authorization", "Bearer "+token)
	respUserOrders, _ := app.Test(reqUserOrders, -1)
	if respUserOrders.StatusCode != http.StatusOK {
		t.Fatalf("GET /user/orders failed with status %d", respUserOrders.StatusCode)
	}

	// 8. Test User Order Detail (GET /user/orders/:orderCode)
	reqUserOrderDetail := httptest.NewRequest("GET", "/api/v1/user/orders/"+orderCode, nil)
	reqUserOrderDetail.Header.Set("Authorization", "Bearer "+token)
	respUserOrderDetail, _ := app.Test(reqUserOrderDetail, -1)
	if respUserOrderDetail.StatusCode != http.StatusOK {
		t.Fatalf("GET /user/orders/%s failed with status %d", orderCode, respUserOrderDetail.StatusCode)
	}

	// 9. Test User Dashboard Summary (GET /user/dashboard-summary)
	reqSummary := httptest.NewRequest("GET", "/api/v1/user/dashboard-summary", nil)
	reqSummary.Header.Set("Authorization", "Bearer "+token)
	respSummary, _ := app.Test(reqSummary, -1)
	if respSummary.StatusCode != http.StatusOK {
		t.Fatalf("GET /user/dashboard-summary failed with status %d", respSummary.StatusCode)
	}

	// 10. Test Update Profile (PUT /user/profile)
	updateProfileBody, _ := json.Marshal(map[string]string{
		"name":  "E2E User Updated Name",
		"phone": "081299998888",
	})
	reqUpdateProf := httptest.NewRequest("PUT", "/api/v1/user/profile", bytes.NewBuffer(updateProfileBody))
	reqUpdateProf.Header.Set("Content-Type", "application/json")
	reqUpdateProf.Header.Set("Authorization", "Bearer "+token)
	respUpdateProf, _ := app.Test(reqUpdateProf, -1)
	if respUpdateProf.StatusCode != http.StatusOK {
		t.Fatalf("PUT /user/profile failed with status %d", respUpdateProf.StatusCode)
	}

	// 11. Test Change Password (POST /user/change-password)
	newPassword := "NewPassUltra789!"
	changePassBody, _ := json.Marshal(map[string]string{
		"oldPassword": testPassword,
		"newPassword": newPassword,
	})
	reqChangePass := httptest.NewRequest("POST", "/api/v1/user/change-password", bytes.NewBuffer(changePassBody))
	reqChangePass.Header.Set("Content-Type", "application/json")
	reqChangePass.Header.Set("Authorization", "Bearer "+token)
	respChangePass, _ := app.Test(reqChangePass, -1)
	if respChangePass.StatusCode != http.StatusOK {
		t.Fatalf("POST /user/change-password failed with status %d", respChangePass.StatusCode)
	}

	// Verify login with new password
	loginNewBody, _ := json.Marshal(map[string]string{
		"email":    testEmail,
		"password": newPassword,
	})
	reqLoginNew := httptest.NewRequest("POST", "/api/v1/auth/login/password", bytes.NewBuffer(loginNewBody))
	reqLoginNew.Header.Set("Content-Type", "application/json")
	respLoginNew, _ := app.Test(reqLoginNew, -1)
	if respLoginNew.StatusCode != http.StatusOK {
		t.Fatalf("Login with new password failed with status %d", respLoginNew.StatusCode)
	}

	t.Log("✅ E2E Complete User Flow Test PASSED successfully with 0 errors!")
}
