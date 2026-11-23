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

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/middleware"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/repositories"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func setupTestApp(t *testing.T) *fiber.App {
	_ = godotenv.Load("../.env")
	if err := database.TryConnectDB(); err != nil {
		t.Skip("PostgreSQL database tidak tersedia, melewati E2E test")
		return nil
	}
	_, _ = database.DB.Exec("UPDATE ticket_categories SET remaining_quota = quota WHERE id = 'c1-vip'")
	_, _ = database.DB.Exec("UPDATE seats SET status = 'AVAILABLE', order_id = NULL WHERE category_id = 'c1-vip'")

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
	if testing.Short() {
		t.Skip("Melewati E2E integration test pada mode short (tanpa koneksi live PostgreSQL)")
	}

	app := setupTestApp(t)
	if app == nil {
		return
	}

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

func TestE2ESuperAdminStaffAndAuditFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Melewati E2E Super Admin integration test pada mode short (tanpa koneksi live PostgreSQL)")
	}

	_ = godotenv.Load("../.env")
	cfg, err := config.LoadConfig("../.env")
	if err != nil {
		t.Fatalf("Gagal memuat konfigurasi: %v", err)
	}
	if err := database.TryConnectDB(); err != nil {
		t.Skip("PostgreSQL database tidak tersedia, melewati E2E test Super Admin")
		return
	}

	orderRepo := repositories.NewOrderRepository(database.DB)
	staffRepo := repositories.NewAdminUserRepository(database.DB)
	secRepo := repositories.NewSecurityRepository(database.DB)
	blockMgr := middleware.NewIPBlocklistManager(secRepo, staffRepo)
	adminCtrl := controllers.NewAdminControllerWithStaffRepo(orderRepo, staffRepo, &cfg.Auth)
	staffCtrl := controllers.NewStaffController(staffRepo)
	securityCtrl := controllers.NewSecurityController(secRepo, staffRepo, blockMgr)

	t.Setenv("APP_ENV", "development")

	app := fiber.New()
	api := app.Group("/api/v1")
	api.Use(blockMgr.BlocklistMiddleware())
	api.Use(blockMgr.SecurityAuditMiddleware())

	admin := api.Group("/admin")
	admin.Post("/login", adminCtrl.AdminLogin)

	guardSuper := middleware.RequireAdminAuth("SUPER_ADMIN")
	admin.Get("/staff", guardSuper, staffCtrl.GetAllStaff)
	admin.Post("/staff", guardSuper, staffCtrl.CreateStaff)
	admin.Put("/staff/:id", guardSuper, staffCtrl.UpdateStaff)
	admin.Patch("/staff/:id/status", guardSuper, staffCtrl.ToggleStaffStatus)
	admin.Post("/staff/:id/reset-password", guardSuper, staffCtrl.ResetStaffPassword)
	admin.Delete("/staff/:id", guardSuper, staffCtrl.DeleteStaff)
	admin.Get("/audit-logs", guardSuper, staffCtrl.GetAuditLogs)
	admin.Get("/audit-logs/search", guardSuper, securityCtrl.GetFilteredAuditLogs)
	admin.Get("/security/blocked-ips", guardSuper, securityCtrl.ListBlockedIPs)
	admin.Post("/security/blocked-ips", guardSuper, securityCtrl.BlockIP)
	admin.Delete("/security/blocked-ips/:id", guardSuper, securityCtrl.UnblockIP)

	// 1. Login sebagai SUPER_ADMIN seeded
	loginSuperBody, _ := json.Marshal(models.AdminLoginRequest{Username: "superadmin", Password: "123"})
	reqLoginSuper := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginSuperBody))
	reqLoginSuper.Header.Set("Content-Type", "application/json")
	respLoginSuper, err := app.Test(reqLoginSuper, -1)
	if err != nil || respLoginSuper.StatusCode != http.StatusOK {
		t.Fatalf("Login superadmin gagal: status=%v err=%v", respLoginSuper, err)
	}
	var loginSuperResp models.APIResponse
	_ = json.NewDecoder(respLoginSuper.Body).Decode(&loginSuperResp)
	superData := loginSuperResp.Data.(map[string]interface{})
	superToken := superData["token"].(string)

	// 2. Buat akun staf sementara (PETUGAS)
	tempUsername := fmt.Sprintf("e2e_staff_%d", time.Now().UnixNano())
	createBody, _ := json.Marshal(models.CreateStaffRequest{
		Username: tempUsername,
		Name:     "E2E Temp Staff",
		Email:    tempUsername + "@symphoniatic.id",
		Password: "TempPassword123!",
		Role:     "PETUGAS",
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff", bytes.NewReader(createBody))
	reqCreate.Header.Set("Authorization", "Bearer "+superToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, err := app.Test(reqCreate, -1)
	if err != nil || respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("CreateStaff gagal: status=%v err=%v", respCreate, err)
	}
	var createResp models.APIResponse
	_ = json.NewDecoder(respCreate.Body).Decode(&createResp)
	createdMap := createResp.Data.(map[string]interface{})
	staffID := createdMap["id"].(string)

	// 3. Login dengan akun staf baru -> verifikasi PETUGAS ditolak 403 di /admin/staff
	loginTempBody, _ := json.Marshal(models.AdminLoginRequest{Username: tempUsername, Password: "TempPassword123!"})
	reqLoginTemp := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginTempBody))
	reqLoginTemp.Header.Set("Content-Type", "application/json")
	respLoginTemp, err := app.Test(reqLoginTemp, -1)
	if err != nil || respLoginTemp.StatusCode != http.StatusOK {
		t.Fatalf("Login staf baru gagal: status=%v err=%v", respLoginTemp, err)
	}
	var loginTempResp models.APIResponse
	_ = json.NewDecoder(respLoginTemp.Body).Decode(&loginTempResp)
	tempToken := loginTempResp.Data.(map[string]interface{})["token"].(string)

	reqForbidden := httptest.NewRequest(http.MethodGet, "/api/v1/admin/staff", nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+tempToken)
	respForbidden, _ := app.Test(reqForbidden, -1)
	if respForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("PETUGAS harus ditolak 403 pada /admin/staff, dapat: %d", respForbidden.StatusCode)
	}

	// 4. Update role staf menjadi ADMIN
	updateBody, _ := json.Marshal(models.UpdateStaffRequest{
		Name:  "E2E Temp Staff Updated",
		Email: tempUsername + "@symphoniatic.id",
		Role:  "ADMIN",
	})
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/admin/staff/"+staffID, bytes.NewReader(updateBody))
	reqUpdate.Header.Set("Authorization", "Bearer "+superToken)
	reqUpdate.Header.Set("Content-Type", "application/json")
	respUpdate, _ := app.Test(reqUpdate, -1)
	if respUpdate.StatusCode != http.StatusOK {
		t.Fatalf("UpdateStaff gagal: status=%d", respUpdate.StatusCode)
	}

	// 5. Nonaktifkan staf -> verifikasi login ditolak 403
	reqToggle := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/staff/"+staffID+"/status", nil)
	reqToggle.Header.Set("Authorization", "Bearer "+superToken)
	respToggle, _ := app.Test(reqToggle, -1)
	if respToggle.StatusCode != http.StatusOK {
		t.Fatalf("ToggleStaffStatus (nonaktif) gagal: status=%d", respToggle.StatusCode)
	}

	reqInactiveLogin := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginTempBody))
	reqInactiveLogin.Header.Set("Content-Type", "application/json")
	respInactiveLogin, _ := app.Test(reqInactiveLogin, -1)
	if respInactiveLogin.StatusCode != http.StatusForbidden {
		t.Fatalf("Login akun nonaktif harus ditolak 403, dapat: %d", respInactiveLogin.StatusCode)
	}

	// 6. Aktifkan kembali & reset password -> login dengan sandi baru berhasil
	reqReenable := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/staff/"+staffID+"/status", nil)
	reqReenable.Header.Set("Authorization", "Bearer "+superToken)
	_, _ = app.Test(reqReenable, -1)

	resetBody, _ := json.Marshal(models.ResetStaffPasswordRequest{NewPassword: "NewResetPass456!"})
	reqReset := httptest.NewRequest(http.MethodPost, "/api/v1/admin/staff/"+staffID+"/reset-password", bytes.NewReader(resetBody))
	reqReset.Header.Set("Authorization", "Bearer "+superToken)
	reqReset.Header.Set("Content-Type", "application/json")
	respReset, _ := app.Test(reqReset, -1)
	if respReset.StatusCode != http.StatusOK {
		t.Fatalf("ResetStaffPassword gagal: status=%d", respReset.StatusCode)
	}

	loginNewPassBody, _ := json.Marshal(models.AdminLoginRequest{Username: tempUsername, Password: "NewResetPass456!"})
	reqNewPass := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginNewPassBody))
	reqNewPass.Header.Set("Content-Type", "application/json")
	respNewPass, _ := app.Test(reqNewPass, -1)
	if respNewPass.StatusCode != http.StatusOK {
		t.Fatalf("Login dengan kata sandi reset gagal: status=%d", respNewPass.StatusCode)
	}

	// 7. Hapus akun staf sementara
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/staff/"+staffID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+superToken)
	respDel, _ := app.Test(reqDel, -1)
	if respDel.StatusCode != http.StatusOK {
		t.Fatalf("DeleteStaff gagal: status=%d", respDel.StatusCode)
	}

	// 8. Verifikasi audit logs tercatat di PostgreSQL
	reqLogs := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?search="+tempUsername, nil)
	reqLogs.Header.Set("Authorization", "Bearer "+superToken)
	respLogs, err := app.Test(reqLogs, -1)
	if err != nil || respLogs.StatusCode != http.StatusOK {
		t.Fatalf("GetAuditLogs gagal: status=%v err=%v", respLogs, err)
	}
	var logsResp models.APIResponse
	_ = json.NewDecoder(respLogs.Body).Decode(&logsResp)
	logsSlice := logsResp.Data.([]interface{})
	if len(logsSlice) < 4 {
		t.Fatalf("Ekspektasi minimal 4 entri audit log untuk siklus hidup staf, dapat %d", len(logsSlice))
	}

	// 9. Simulasi probe keamanan dari IP eksternal -> verifikasi tercatat & bisa diblokir lalu dibuka kembali
	attackerIP := "203.0.113.88"
	reqProbe := httptest.NewRequest(http.MethodGet, "/api/v1/admin/.env", nil)
	reqProbe.Header.Set("X-Test-Client-IP", attackerIP)
	_, _ = app.Test(reqProbe, -1)

	reqSecSearch := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/search?category=SECURITY&ip="+attackerIP, nil)
	reqSecSearch.Header.Set("Authorization", "Bearer "+superToken)
	respSecSearch, err := app.Test(reqSecSearch, -1)
	if err != nil || respSecSearch.StatusCode != http.StatusOK {
		t.Fatalf("GetFilteredAuditLogs gagal: status=%v err=%v", respSecSearch, err)
	}
	var secSearchResp models.APIResponse
	_ = json.NewDecoder(respSecSearch.Body).Decode(&secSearchResp)
	secPage := secSearchResp.Data.(map[string]interface{})
	if secPage["total"].(float64) < 1 {
		t.Fatalf("Ekspektasi minimal 1 audit log SECURITY untuk IP %s, dapat %v", attackerIP, secPage["total"])
	}

	blockBody, _ := json.Marshal(models.BlockIPRequest{
		IPAddress:       attackerIP,
		Reason:          "E2E test security probe",
		DurationMinutes: 15,
	})
	reqBlock := httptest.NewRequest(http.MethodPost, "/api/v1/admin/security/blocked-ips", bytes.NewReader(blockBody))
	reqBlock.Header.Set("Authorization", "Bearer "+superToken)
	reqBlock.Header.Set("Content-Type", "application/json")
	respBlock, err := app.Test(reqBlock, -1)
	if err != nil || respBlock.StatusCode != http.StatusCreated {
		t.Fatalf("BlockIP gagal: status=%v err=%v", respBlock, err)
	}

	reqBlockedAttempt := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginSuperBody))
	reqBlockedAttempt.Header.Set("X-Test-Client-IP", attackerIP)
	reqBlockedAttempt.Header.Set("Content-Type", "application/json")
	respBlockedAttempt, _ := app.Test(reqBlockedAttempt, -1)
	if respBlockedAttempt.StatusCode != http.StatusForbidden {
		t.Fatalf("IP yang diblokir harus ditolak 403, dapat: %d", respBlockedAttempt.StatusCode)
	}

	reqUnblock := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/security/blocked-ips/"+attackerIP, nil)
	reqUnblock.Header.Set("Authorization", "Bearer "+superToken)
	respUnblock, err := app.Test(reqUnblock, -1)
	if err != nil || respUnblock.StatusCode != http.StatusOK {
		t.Fatalf("UnblockIP gagal: status=%v err=%v", respUnblock, err)
	}
}
