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
	var lastOTPCreatedAt time.Time
	cooldownCheckErr := database.DB.QueryRow(`
		SELECT created_at FROM auth_otps 
		WHERE email = $1 AND purpose = $2 AND created_at > $3 
		ORDER BY created_at DESC LIMIT 1
	`, email, purpose, time.Now().Add(-OTPCooldownDuration)).Scan(&lastOTPCreatedAt)
	if cooldownCheckErr == nil {
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
	otpExpiresAt := time.Now().Add(OTPTTL)

	_, err = database.DB.Exec(`
		INSERT INTO auth_otps (id, email, otp_code, purpose, attempts, is_used, expires_at)
		VALUES ($1, $2, $3, $4, 0, FALSE, $5)
	`, otpID, email, otpCode, purpose, otpExpiresAt)
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

	var activeOTPRecord models.AuthOTPRecord
	err := database.DB.QueryRow(`
		SELECT id, email, otp_code, purpose, attempts, is_used, expires_at, created_at
		FROM auth_otps
		WHERE email = $1 AND purpose = $2 AND is_used = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, email, purpose).Scan(&activeOTPRecord.ID, &activeOTPRecord.Email, &activeOTPRecord.OTPCode, &activeOTPRecord.Purpose, &activeOTPRecord.Attempts, &activeOTPRecord.IsUsed, &activeOTPRecord.ExpiresAt, &activeOTPRecord.CreatedAt)

	if err != nil {
		return ErrOTPInvalid
	}

	if activeOTPRecord.Attempts >= MaxOTPAttempts {
		_, _ = database.DB.Exec("UPDATE auth_otps SET is_used = TRUE WHERE id = $1", activeOTPRecord.ID)
		return ErrOTPMaxExceed
	}

	if activeOTPRecord.OTPCode != otpCode {
		newAttemptCount := activeOTPRecord.Attempts + 1
		isOTPNowExhausted := newAttemptCount >= MaxOTPAttempts
		_, _ = database.DB.Exec("UPDATE auth_otps SET attempts = $1, is_used = $2 WHERE id = $3", newAttemptCount, isOTPNowExhausted, activeOTPRecord.ID)

		if isOTPNowExhausted {
			return ErrOTPMaxExceed
		}
		return fmt.Errorf("Sisa percobaan: %d", MaxOTPAttempts-newAttemptCount)
	}

	// Mark as used
	_, _ = database.DB.Exec("UPDATE auth_otps SET is_used = TRUE WHERE id = $1", activeOTPRecord.ID)
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
func RequestRegisterOTP(ctx *fiber.Ctx) error {
	var registerReq models.RegisterRequestOTPInput
	if err := ctx.BodyParser(&registerReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	registerReq.Email = strings.TrimSpace(strings.ToLower(registerReq.Email))
	registerReq.Name = strings.TrimSpace(registerReq.Name)

	if registerReq.Email == "" || registerReq.Name == "" {
		return utils.ResponseBadRequest(ctx, "Email dan Nama wajib diisi.")
	}

	var existingUserCount int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", registerReq.Email).Scan(&existingUserCount)
	if err == nil && existingUserCount > 0 {
		return utils.ResponseBadRequest(ctx, "Email ini sudah terdaftar. Silakan login ke akun Anda.")
	}

	err = IssueOTP(registerReq.Email, registerReq.Name, "REGISTER")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(ctx, "Mohon tunggu 60 detik sebelum meminta kode OTP registrasi baru.")
	} else if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal membuat dan mengirimkan kode OTP registrasi.", err)
	}

	return utils.ResponseOK(ctx, "Kode OTP registrasi berhasil dikirim ke email Anda. Berlaku selama 5 menit.", nil)
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
func VerifyRegisterOTP(ctx *fiber.Ctx) error {
	var registerVerifyReq models.RegisterVerifyOTPInput
	if err := ctx.BodyParser(&registerVerifyReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	registerVerifyReq.Email = strings.TrimSpace(strings.ToLower(registerVerifyReq.Email))
	registerVerifyReq.OTPCode = strings.TrimSpace(registerVerifyReq.OTPCode)

	if registerVerifyReq.Email == "" || registerVerifyReq.OTPCode == "" || registerVerifyReq.Password == "" {
		return utils.ResponseBadRequest(ctx, "Email, Kode OTP, dan Password wajib diisi.")
	}

	if len(registerVerifyReq.Password) < 6 {
		return utils.ResponseBadRequest(ctx, "Password minimal 6 karakter.")
	}

	err := ValidateAndConsumeOTP(registerVerifyReq.Email, "REGISTER", registerVerifyReq.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak valid atau telah kedaluwarsa. Silakan minta OTP baru.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(ctx, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak sesuai. "+err.Error())
	}

	hashedPassword, err := utils.HashPassword(registerVerifyReq.Password)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memproses keamanan kata sandi.", err)
	}

	displayName := strings.TrimSpace(registerVerifyReq.Name)
	if displayName == "" {
		emailParts := strings.Split(registerVerifyReq.Email, "@")
		if len(emailParts) > 0 {
			displayName = strings.Title(emailParts[0])
		} else {
			displayName = registerVerifyReq.Email
		}
	}

	newUserAccount := models.UserRecord{
		ID:         "usr-" + uuid.New().String(),
		Email:      registerVerifyReq.Email,
		Name:       displayName,
		Role:       "USER",
		IsVerified: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = database.DB.Exec(`
		INSERT INTO users (id, email, name, password_hash, role, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, newUserAccount.ID, newUserAccount.Email, newUserAccount.Name, hashedPassword, newUserAccount.Role, newUserAccount.IsVerified)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal membuat akun pengguna.", err)
	}

	jwtToken, err := utils.GenerateUserToken(newUserAccount.ID, newUserAccount.Email, newUserAccount.Role)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Registrasi berhasil, tetapi gagal menerbitkan token akses.", err)
	}

	return utils.ResponseCreated(ctx, "Registrasi akun berhasil!", models.AuthResponseData{
		Token: jwtToken,
		User:  newUserAccount,
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
func PasswordLogin(ctx *fiber.Ctx) error {
	var loginReq models.PasswordLoginInput
	if err := ctx.BodyParser(&loginReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	loginReq.Email = strings.TrimSpace(strings.ToLower(loginReq.Email))

	if loginReq.Email == "" || loginReq.Password == "" {
		return utils.ResponseBadRequest(ctx, "Email dan Password wajib diisi.")
	}

	var userAccount models.UserRecord
	var storedPasswordHash string

	err := database.DB.QueryRow(`
		SELECT id, email, name, phone, password_hash, role, is_verified, created_at, updated_at
		FROM users WHERE email = $1
	`, loginReq.Email).Scan(&userAccount.ID, &userAccount.Email, &userAccount.Name, &userAccount.Phone, &storedPasswordHash, &userAccount.Role, &userAccount.IsVerified, &userAccount.CreatedAt, &userAccount.UpdatedAt)

	if err == sql.ErrNoRows {
		return utils.ResponseUnauthorized(ctx, "Email atau password yang Anda masukkan salah.")
	} else if err != nil {
		return utils.ResponseInternalError(ctx, "Terjadi kesalahan saat memeriksa data akun.", err)
	}

	if !utils.CheckPasswordHash(loginReq.Password, storedPasswordHash) {
		return utils.ResponseUnauthorized(ctx, "Email atau password yang Anda masukkan salah.")
	}

	jwtToken, err := utils.GenerateUserToken(userAccount.ID, userAccount.Email, userAccount.Role)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menerbitkan token sesi login.", err)
	}

	return utils.ResponseOK(ctx, "Login berhasil!", models.AuthResponseData{
		Token: jwtToken,
		User:  userAccount,
	})
}

// RequestLoginOTP handles requesting OTP for OTP-based Login
func RequestLoginOTP(ctx *fiber.Ctx) error {
	var loginOTPReq models.LoginRequestOTPInput
	if err := ctx.BodyParser(&loginOTPReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	loginOTPReq.Email = strings.TrimSpace(strings.ToLower(loginOTPReq.Email))
	if loginOTPReq.Email == "" {
		return utils.ResponseBadRequest(ctx, "Email wajib diisi.")
	}

	var registeredUserName string
	err := database.DB.QueryRow("SELECT name FROM users WHERE email = $1", loginOTPReq.Email).Scan(&registeredUserName)
	if err == sql.ErrNoRows {
		return utils.ResponseNotFound(ctx, "Email tidak terdaftar. Silakan lakukan registrasi terlebih dahulu.")
	}

	err = IssueOTP(loginOTPReq.Email, registeredUserName, "LOGIN")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(ctx, "Mohon tunggu 60 detik sebelum meminta kode OTP login baru.")
	} else if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal membuat dan mengirimkan kode OTP login.", err)
	}

	return utils.ResponseOK(ctx, "Kode OTP login telah dikirim ke email Anda.", nil)
}

// VerifyLoginOTP verifies OTP for login and returns JWT
func VerifyLoginOTP(ctx *fiber.Ctx) error {
	var loginVerifyReq models.LoginVerifyOTPInput
	if err := ctx.BodyParser(&loginVerifyReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	loginVerifyReq.Email = strings.TrimSpace(strings.ToLower(loginVerifyReq.Email))
	loginVerifyReq.OTPCode = strings.TrimSpace(loginVerifyReq.OTPCode)

	if loginVerifyReq.Email == "" || loginVerifyReq.OTPCode == "" {
		return utils.ResponseBadRequest(ctx, "Email dan Kode OTP wajib diisi.")
	}

	err := ValidateAndConsumeOTP(loginVerifyReq.Email, "LOGIN", loginVerifyReq.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak valid atau telah kedaluwarsa.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(ctx, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak sesuai. "+err.Error())
	}

	var userAccount models.UserRecord
	err = database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE email = $1
	`, loginVerifyReq.Email).Scan(&userAccount.ID, &userAccount.Email, &userAccount.Name, &userAccount.Phone, &userAccount.Role, &userAccount.IsVerified, &userAccount.CreatedAt, &userAccount.UpdatedAt)

	if err != nil {
		return utils.ResponseNotFound(ctx, "Data akun tidak ditemukan.")
	}

	jwtToken, err := utils.GenerateUserToken(userAccount.ID, userAccount.Email, userAccount.Role)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal menerbitkan token sesi login.", err)
	}

	return utils.ResponseOK(ctx, "Login via OTP berhasil!", models.AuthResponseData{
		Token: jwtToken,
		User:  userAccount,
	})
}

// ─── 3. FORGOT PASSWORD VIA OTP ───

// RequestForgotPasswordOTP handles requesting OTP to reset password
func RequestForgotPasswordOTP(ctx *fiber.Ctx) error {
	var forgotPwdReq models.ForgotPasswordRequestOTPInput
	if err := ctx.BodyParser(&forgotPwdReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	forgotPwdReq.Email = strings.TrimSpace(strings.ToLower(forgotPwdReq.Email))
	if forgotPwdReq.Email == "" {
		return utils.ResponseBadRequest(ctx, "Email wajib diisi.")
	}

	var registeredUserName string
	err := database.DB.QueryRow("SELECT name FROM users WHERE email = $1", forgotPwdReq.Email).Scan(&registeredUserName)
	if err == sql.ErrNoRows {
		return utils.ResponseNotFound(ctx, "Email tidak terdaftar dalam sistem.")
	}

	err = IssueOTP(forgotPwdReq.Email, registeredUserName, "FORGOT_PASSWORD")
	if err == ErrOTPCooldown {
		return utils.ResponseTooManyRequests(ctx, "Mohon tunggu 60 detik sebelum meminta kode OTP reset kata sandi baru.")
	} else if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal membuat dan mengirimkan kode OTP reset kata sandi.", err)
	}

	return utils.ResponseOK(ctx, "Kode OTP reset kata sandi telah dikirim ke email Anda.", nil)
}

// VerifyForgotPasswordOTP verifies OTP and returns a temporary Password Reset Token
func VerifyForgotPasswordOTP(ctx *fiber.Ctx) error {
	var forgotPwdVerifyReq models.ForgotPasswordVerifyOTPInput
	if err := ctx.BodyParser(&forgotPwdVerifyReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	forgotPwdVerifyReq.Email = strings.TrimSpace(strings.ToLower(forgotPwdVerifyReq.Email))
	forgotPwdVerifyReq.OTPCode = strings.TrimSpace(forgotPwdVerifyReq.OTPCode)

	if forgotPwdVerifyReq.Email == "" || forgotPwdVerifyReq.OTPCode == "" {
		return utils.ResponseBadRequest(ctx, "Email dan Kode OTP wajib diisi.")
	}

	err := ValidateAndConsumeOTP(forgotPwdVerifyReq.Email, "FORGOT_PASSWORD", forgotPwdVerifyReq.OTPCode)
	if err == ErrOTPInvalid {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak valid atau telah kedaluwarsa.")
	} else if err == ErrOTPMaxExceed {
		return utils.ResponseBadRequest(ctx, "Batas percobaan verifikasi OTP (5x) telah terlampaui. Silakan minta OTP baru.")
	} else if err != nil {
		return utils.ResponseBadRequest(ctx, "Kode OTP tidak sesuai. "+err.Error())
	}

	passwordResetToken, err := utils.GeneratePasswordResetToken(forgotPwdVerifyReq.Email)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal memproses permintaan reset kata sandi.", err)
	}

	return utils.ResponseOK(ctx, "Verifikasi OTP berhasil! Gunakan token ini untuk menyetel ulang kata sandi Anda.", models.VerifyOTPResponseData{
		ResetToken: passwordResetToken,
		Message:    "Verifikasi sukses",
	})
}

// ResetPassword sets a new password using the validated Reset Token
func ResetPassword(ctx *fiber.Ctx) error {
	var resetPwdReq models.ResetPasswordInput
	if err := ctx.BodyParser(&resetPwdReq); err != nil {
		return utils.ResponseBadRequest(ctx, "Format payload request tidak valid.")
	}

	if resetPwdReq.ResetToken == "" || resetPwdReq.NewPassword == "" {
		return utils.ResponseBadRequest(ctx, "Reset Token dan Kata Sandi Baru wajib diisi.")
	}

	if len(resetPwdReq.NewPassword) < 6 {
		return utils.ResponseBadRequest(ctx, "Kata sandi baru minimal 6 karakter.")
	}

	tokenClaims, err := utils.ValidatePasswordResetToken(resetPwdReq.ResetToken)
	if err != nil {
		return utils.ResponseUnauthorized(ctx, "Token reset kata sandi tidak valid atau telah kedaluwarsa.")
	}

	newPasswordHash, err := utils.HashPassword(resetPwdReq.NewPassword)
	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengamankan kata sandi baru.", err)
	}

	updateResult, err := database.DB.Exec(`
		UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2
	`, newPasswordHash, tokenClaims.Email)

	if err != nil {
		return utils.ResponseInternalError(ctx, "Gagal mengonfirmasi kata sandi baru.", err)
	}

	affectedRows, _ := updateResult.RowsAffected()
	if affectedRows == 0 {
		return utils.ResponseNotFound(ctx, "Pengguna tidak ditemukan.")
	}

	return utils.ResponseOK(ctx, "Kata sandi akun Anda berhasil diperbarui. Silakan login dengan kata sandi baru Anda.", nil)
}

// ─── 4. PROFIL SAYA (PROTECTED) ───

// GetMyProfile returns the profile of the logged in user
func GetMyProfile(ctx *fiber.Ctx) error {
	authenticatedUserID, ok := ctx.Locals("user_id").(string)
	if !ok || authenticatedUserID == "" {
		return utils.ResponseUnauthorized(ctx, "Sesi tidak valid.")
	}

	var userAccount models.UserRecord
	err := database.DB.QueryRow(`
		SELECT id, email, name, phone, role, is_verified, created_at, updated_at
		FROM users WHERE id = $1
	`, authenticatedUserID).Scan(&userAccount.ID, &userAccount.Email, &userAccount.Name, &userAccount.Phone, &userAccount.Role, &userAccount.IsVerified, &userAccount.CreatedAt, &userAccount.UpdatedAt)

	if err != nil {
		return utils.ResponseNotFound(ctx, "Data profil pengguna tidak ditemukan.")
	}

	return utils.ResponseOK(ctx, "Berhasil mengambil profil pengguna.", userAccount)
}
