package controllers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/database"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	"github.com/Zyrexnn/SymphoniaTic-be/services"
	"github.com/Zyrexnn/SymphoniaTic-be/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	OTPCooldownDuration = 60 * time.Second
	OTPTTL              = 5 * time.Minute
	MaxOTPAttempts      = 5
)

// Helper error types for OTP validation
var (
	ErrOTPCooldown  = fmt.Errorf("COOLDOWN")
	ErrOTPInvalid   = fmt.Errorf("INVALID_OR_EXPIRED")
	ErrOTPMaxExceed = fmt.Errorf("MAX_ATTEMPTS_EXCEEDED")
)

// IssueOTP encapsulates cooldown check, invalidating old OTPs, generating crypto OTP, saving DB, and sending Mailpit email.
func IssueOTP(email, name, purpose string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	// Check Cooldown (60 seconds)
	var createdAt time.Time
	err := database.DB.QueryRow(`
		SELECT created_at FROM auth_otps 
		WHERE email = $1 AND purpose = $2 AND created_at > $3 
		ORDER BY created_at DESC LIMIT 1
	`, email, purpose, time.Now().Add(-OTPCooldownDuration)).Scan(&createdAt)
	if err == nil {
		return ErrOTPCooldown
	}

	// Invalidate older active OTPs
	_, _ = database.DB.Exec(`
		UPDATE auth_otps SET is_used = TRUE 
		WHERE email = $1 AND purpose = $2 AND is_used = FALSE
	`, email, purpose)

	// Generate 6-digit crypto OTP
	otpCode, err := utils.GenerateCryptoOTP()
	if err != nil {
		return err
	}

	otpID := "otp-" + uuid.New().String()
	expiresAt := time.Now().Add(OTPTTL)

	_, err = database.DB.Exec(`
		INSERT INTO auth_otps (id, email, otp_code, purpose, attempts, is_used, expires_at)
		VALUES ($1, $2, $3, $4, 0, FALSE, $5)
	`, otpID, email, otpCode, purpose, expiresAt)
	if err != nil {
		return err
	}

	// Dispatch email via Mailpit asynchronously
	services.SendAuthOTPEmail(email, name, otpCode, purpose)
	return nil
}

// ValidateAndConsumeOTP verifies an incoming OTP code, checks max attempts (5x), and marks it used.
func ValidateAndConsumeOTP(email, purpose, otpCode string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	otpCode = strings.TrimSpace(otpCode)

	var otp models.AuthOTPRecord
	err := database.DB.QueryRow(`
		SELECT id, email, otp_code, purpose, attempts, is_used, expires_at, created_at
		FROM auth_otps
		WHERE email = $1 AND purpose = $2 AND is_used = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, email, purpose).Scan(&otp.ID, &otp.Email, &otp.OTPCode, &otp.Purpose, &otp.Attempts, &otp.IsUsed, &otp.ExpiresAt, &otp.CreatedAt)

	if err != nil {
		return ErrOTPInvalid
	}

	if otp.Attempts >= MaxOTPAttempts {
		_, _ = database.DB.Exec("UPDATE auth_otps SET is_used = TRUE WHERE id = $1", otp.ID)
		return ErrOTPMaxExceed
	}

	if otp.OTPCode != otpCode {
		newAttempts := otp.Attempts + 1
		isUsed := newAttempts >= MaxOTPAttempts
		_, _ = database.DB.Exec("UPDATE auth_otps SET attempts = $1, is_used = $2 WHERE id = $3", newAttempts, isUsed, otp.ID)

		if isUsed {
			return ErrOTPMaxExceed
		}
		return fmt.Errorf("Sisa percobaan: %d", MaxOTPAttempts-newAttempts)
	}

	// Mark as used
	_, _ = database.DB.Exec("UPDATE auth_otps SET is_used = TRUE WHERE id = $1", otp.ID)
	return nil
}

// ─── 1. REGISTRASI VIA OTP ───

// RequestRegisterOTP godoc
// @Summary Meminta kode OTP untuk registrasi
// @Description Mengirimkan 6 digit kode OTP registrasi ke email pengguna.
// @Tags Auth - User
// @Accept json
// @Produce json
// @Param payload body models.RegisterRequestOTPInput true "Register Request OTP Payload"
// @Success 200 {object} models.APIResponse "Kode OTP registrasi berhasil dikirim"
// @Failure 400 {object} models.APIResponse "Payload tidak valid / Email terdaftar"
// @Failure 429 {object} models.APIResponse "Cooldown OTP (tunggu 60s)"
// @Router /auth/register/request-otp [post]
func RequestRegisterOTP(c *fiber.Ctx) error {
	var req models.RegisterRequestOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Name == "" {
		return utils.ResponseBadRequest(c, "Email dan Nama wajib diisi.")
	}

	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", req.Email).Scan(&count)
	if err == nil && count > 0 {
		return utils.ResponseBadRequest(c, "Email ini sudah terdaftar. Silakan login ke akun Anda.")
	}

	err = IssueOTP(req.Email, req.Name, "REGISTER")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(c, "Mohon tunggu 60 detik sebelum meminta kode OTP registrasi baru.")
	} else if err != nil {
		return utils.ResponseInternalError(c, "Gagal membuat dan mengirimkan kode OTP registrasi.", err)
	}

	return utils.ResponseOK(c, "Kode OTP registrasi berhasil dikirim ke email Anda. Berlaku selama 5 menit.", nil)
}

// VerifyRegisterOTP godoc
// @Summary Verifikasi OTP & Buat Akun Pengguna
// @Description Memverifikasi kode OTP registrasi dan membuat akun baru beserta JWT token.
// @Tags Auth - User
// @Accept json
// @Produce json
// @Param payload body models.RegisterVerifyOTPInput true "Register Verify OTP Payload"
// @Success 201 {object} models.APIResponse{data=models.AuthResponseData} "Registrasi akun berhasil!"
// @Failure 400 {object} models.APIResponse "OTP tidak sesuai / expired"
// @Router /auth/register/verify-otp [post]
func VerifyRegisterOTP(c *fiber.Ctx) error {
	var req models.RegisterVerifyOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.OTPCode = strings.TrimSpace(req.OTPCode)

	if req.Email == "" || req.OTPCode == "" || req.Password == "" {
		return utils.ResponseBadRequest(c, "Email, Kode OTP, dan Password wajib diisi.")
	}

	if len(req.Password) < 6 {
		return utils.ResponseBadRequest(c, "Password minimal 6 karakter.")
	}

	err := ValidateAndConsumeOTP(req.Email, "REGISTER", req.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(c, "Kode OTP tidak valid atau telah kedaluwarsa. Silakan minta OTP baru.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(c, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(c, "Kode OTP tidak sesuai. "+err.Error())
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memproses keamanan kata sandi.", err)
	}

	userName := strings.TrimSpace(req.Name)
	if userName == "" {
		parts := strings.Split(req.Email, "@")
		if len(parts) > 0 {
			userName = strings.Title(parts[0])
		} else {
			userName = req.Email
		}
	}

	user := models.UserRecord{
		ID:         "usr-" + uuid.New().String(),
		Email:      req.Email,
		Name:       userName,
		Role:       "USER",
		IsVerified: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = database.DB.Exec(`
		INSERT INTO users (id, email, name, password_hash, role, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, user.ID, user.Email, user.Name, hashedPassword, user.Role, user.IsVerified)

	if err != nil {
		return utils.ResponseInternalError(c, "Gagal membuat akun pengguna.", err)
	}

	token, err := utils.GenerateUserToken(user.ID, user.Email, user.Role)
	if err != nil {
		return utils.ResponseInternalError(c, "Registrasi berhasil, tetapi gagal menerbitkan token akses.", err)
	}

	return utils.ResponseCreated(c, "Registrasi akun berhasil!", models.AuthResponseData{
		Token: token,
		User:  user,
	})
}

// ─── 2. LOGIN (PASSWORD & OTP) ───

// PasswordLogin godoc
// @Summary Login dengan Email & Password
// @Description Autentikasi user menggunakan email dan password tradisional.
// @Tags Auth - User
// @Accept json
// @Produce json
// @Param payload body models.PasswordLoginInput true "Login Payload"
// @Success 200 {object} models.APIResponse{data=models.AuthResponseData} "Login berhasil!"
// @Failure 401 {object} models.APIResponse "Email atau password salah"
// @Router /auth/login/password [post]
func PasswordLogin(c *fiber.Ctx) error {
	var req models.PasswordLoginInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		return utils.ResponseBadRequest(c, "Email dan Password wajib diisi.")
	}

	var user models.UserRecord
	var passwordHash string

	err := database.DB.QueryRow(`
		SELECT id, email, name, phone, password_hash, role, is_verified, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Name, &user.Phone, &passwordHash, &user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return utils.ResponseUnauthorized(c, "Email atau password yang Anda masukkan salah.")
	} else if err != nil {
		return utils.ResponseInternalError(c, "Terjadi kesalahan saat memeriksa data akun.", err)
	}

	if !utils.CheckPasswordHash(req.Password, passwordHash) {
		return utils.ResponseUnauthorized(c, "Email atau password yang Anda masukkan salah.")
	}

	token, err := utils.GenerateUserToken(user.ID, user.Email, user.Role)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal menerbitkan token sesi login.", err)
	}

	return utils.ResponseOK(c, "Login berhasil!", models.AuthResponseData{
		Token: token,
		User:  user,
	})
}

// RequestLoginOTP handles requesting OTP for OTP-based Login
func RequestLoginOTP(c *fiber.Ctx) error {
	var req models.LoginRequestOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return utils.ResponseBadRequest(c, "Email wajib diisi.")
	}

	var userName string
	err := database.DB.QueryRow("SELECT name FROM users WHERE email = $1", req.Email).Scan(&userName)
	if err == sql.ErrNoRows {
		return utils.ResponseNotFound(c, "Email tidak terdaftar. Silakan lakukan registrasi terlebih dahulu.")
	}

	err = IssueOTP(req.Email, userName, "LOGIN")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(c, "Mohon tunggu 60 detik sebelum meminta kode OTP login baru.")
	} else if err != nil {
		return utils.ResponseInternalError(c, "Gagal membuat dan mengirimkan kode OTP login.", err)
	}

	return utils.ResponseOK(c, "Kode OTP login telah dikirim ke email Anda.", nil)
}

// VerifyLoginOTP verifies OTP for login and returns JWT
func VerifyLoginOTP(c *fiber.Ctx) error {
	var req models.LoginVerifyOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.OTPCode = strings.TrimSpace(req.OTPCode)

	if req.Email == "" || req.OTPCode == "" {
		return utils.ResponseBadRequest(c, "Email dan Kode OTP wajib diisi.")
	}

	err := ValidateAndConsumeOTP(req.Email, "LOGIN", req.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(c, "Kode OTP tidak valid atau telah kedaluwarsa.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(c, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(c, "Kode OTP tidak sesuai. "+err.Error())
	}

	var user models.UserRecord
	err = database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Name, &user.Phone, &user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return utils.ResponseNotFound(c, "Data akun tidak ditemukan.")
	}

	token, err := utils.GenerateUserToken(user.ID, user.Email, user.Role)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal menerbitkan token sesi login.", err)
	}

	return utils.ResponseOK(c, "Login via OTP berhasil!", models.AuthResponseData{
		Token: token,
		User:  user,
	})
}

// ─── 3. FORGOT PASSWORD VIA OTP ───

// RequestForgotPasswordOTP handles requesting OTP to reset password
func RequestForgotPasswordOTP(c *fiber.Ctx) error {
	var req models.ForgotPasswordRequestOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return utils.ResponseBadRequest(c, "Email wajib diisi.")
	}

	var userName string
	err := database.DB.QueryRow("SELECT name FROM users WHERE email = $1", req.Email).Scan(&userName)
	if err == sql.ErrNoRows {
		return utils.ResponseNotFound(c, "Email tidak terdaftar dalam sistem.")
	}

	err = IssueOTP(req.Email, userName, "FORGOT_PASSWORD")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(c, "Mohon tunggu 60 detik sebelum meminta kode OTP reset kata sandi baru.")
	} else if err != nil {
		return utils.ResponseInternalError(c, "Gagal membuat dan mengirimkan kode OTP reset kata sandi.", err)
	}

	return utils.ResponseOK(c, "Kode OTP reset kata sandi telah dikirim ke email Anda.", nil)
}

// VerifyForgotPasswordOTP verifies OTP and returns a temporary Password Reset Token
func VerifyForgotPasswordOTP(c *fiber.Ctx) error {
	var req models.ForgotPasswordVerifyOTPInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.OTPCode = strings.TrimSpace(req.OTPCode)

	if req.Email == "" || req.OTPCode == "" {
		return utils.ResponseBadRequest(c, "Email dan Kode OTP wajib diisi.")
	}

	err := ValidateAndConsumeOTP(req.Email, "FORGOT_PASSWORD", req.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(c, "Kode OTP tidak valid atau telah kedaluwarsa.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(c, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(c, "Kode OTP tidak sesuai. "+err.Error())
	}

	resetToken, err := utils.GeneratePasswordResetToken(req.Email)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal memproses permintaan reset kata sandi.", err)
	}

	return utils.ResponseOK(c, "Verifikasi OTP berhasil! Gunakan token ini untuk menyetel ulang kata sandi Anda.", models.VerifyOTPResponseData{
		ResetToken: resetToken,
		Message:    "Verifikasi sukses",
	})
}

// ResetPassword sets a new password using the validated Reset Token
func ResetPassword(c *fiber.Ctx) error {
	var req models.ResetPasswordInput
	if err := c.BodyParser(&req); err != nil {
		return utils.ResponseBadRequest(c, "Format payload request tidak valid.")
	}

	if req.ResetToken == "" || req.NewPassword == "" {
		return utils.ResponseBadRequest(c, "Reset Token dan Kata Sandi Baru wajib diisi.")
	}

	if len(req.NewPassword) < 6 {
		return utils.ResponseBadRequest(c, "Kata sandi baru minimal 6 karakter.")
	}

	claims, err := utils.ValidatePasswordResetToken(req.ResetToken)
	if err != nil {
		return utils.ResponseUnauthorized(c, "Token reset kata sandi tidak valid atau telah kedaluwarsa.")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.ResponseInternalError(c, "Gagal mengamankan kata sandi baru.", err)
	}

	result, err := database.DB.Exec(`
		UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2
	`, hashedPassword, claims.Email)

	if err != nil {
		return utils.ResponseInternalError(c, "Gagal mengonfirmasi kata sandi baru.", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return utils.ResponseNotFound(c, "Pengguna tidak ditemukan.")
	}

	return utils.ResponseOK(c, "Kata sandi akun Anda berhasil diperbarui. Silakan login dengan kata sandi baru Anda.", nil)
}

// ─── 4. PROFIL SAYA (PROTECTED) ───

// GetMyProfile returns the profile of the logged in user
func GetMyProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return utils.ResponseUnauthorized(c, "Sesi tidak valid.")
	}

	var user models.UserRecord
	err := database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.Phone, &user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return utils.ResponseNotFound(c, "Data profil pengguna tidak ditemukan.")
	}

	return utils.ResponseOK(c, "Berhasil mengambil profil pengguna.", user)
}
