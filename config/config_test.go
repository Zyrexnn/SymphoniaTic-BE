package config_test

import (
	"testing"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Kosongkan variabel lingkungan untuk memastikan nilai default yang digunakan
	envVars := []string{
		"PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"SMTP_HOST", "SMTP_PORT", "SMTP_SENDER_EMAIL", "SMTP_SENDER_NAME",
		"JWT_SECRET", "ADMIN_USERNAME", "ADMIN_PASSWORD", "PETUGAS_USERNAME", "PETUGAS_PASSWORD",
	}
	for _, env := range envVars {
		t.Setenv(env, "")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig menghasilkan error tidak terduga: %v", err)
	}

	if cfg.Server.Port != "8082" {
		t.Errorf("Port server default salah: diharapkan '8082', didapat '%s'", cfg.Server.Port)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("DB Host default salah: diharapkan 'localhost', didapat '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("DB Port default salah: diharapkan '5432', didapat '%s'", cfg.Database.Port)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Errorf("DB SSLMode default salah: diharapkan 'disable', didapat '%s'", cfg.Database.SSLMode)
	}

	if cfg.SMTP.Host != "localhost" {
		t.Errorf("SMTP Host default salah: diharapkan 'localhost', didapat '%s'", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != "1025" {
		t.Errorf("SMTP Port default salah: diharapkan '1025', didapat '%s'", cfg.SMTP.Port)
	}

	expectedDSN := "host=localhost port=5432 user= password= dbname= sslmode=disable"
	if dsn := cfg.Database.DSN(); dsn != expectedDSN {
		t.Errorf("DSN default salah: diharapkan '%s', didapat '%s'", expectedDSN, dsn)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_HOST", "postgres.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("SMTP_HOST", "smtp.internal")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_SENDER_EMAIL", "noreply@custom.com")
	t.Setenv("SMTP_SENDER_NAME", "Custom Mailer")
	t.Setenv("JWT_SECRET", "supersecretjwt")
	t.Setenv("ADMIN_USERNAME", "superadmin")
	t.Setenv("ADMIN_PASSWORD", "adminpassword")
	t.Setenv("PETUGAS_USERNAME", "superpetugas")
	t.Setenv("PETUGAS_PASSWORD", "petugaspassword")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig menghasilkan error tidak terduga: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Port server kustom salah: diharapkan '9090', didapat '%s'", cfg.Server.Port)
	}
	if cfg.Database.Host != "postgres.internal" {
		t.Errorf("DB Host kustom salah: diharapkan 'postgres.internal', didapat '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != "5433" {
		t.Errorf("DB Port kustom salah: diharapkan '5433', didapat '%s'", cfg.Database.Port)
	}
	if cfg.Database.User != "testuser" {
		t.Errorf("DB User kustom salah: diharapkan 'testuser', didapat '%s'", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("DB Password kustom salah: diharapkan 'testpass', didapat '%s'", cfg.Database.Password)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("DB Name kustom salah: diharapkan 'testdb', didapat '%s'", cfg.Database.Name)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("DB SSLMode kustom salah: diharapkan 'require', didapat '%s'", cfg.Database.SSLMode)
	}

	if cfg.SMTP.Host != "smtp.internal" {
		t.Errorf("SMTP Host kustom salah: diharapkan 'smtp.internal', didapat '%s'", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != "2525" {
		t.Errorf("SMTP Port kustom salah: diharapkan '2525', didapat '%s'", cfg.SMTP.Port)
	}
	if cfg.SMTP.SenderEmail != "noreply@custom.com" {
		t.Errorf("SMTP SenderEmail kustom salah: diharapkan 'noreply@custom.com', didapat '%s'", cfg.SMTP.SenderEmail)
	}
	if cfg.SMTP.SenderName != "Custom Mailer" {
		t.Errorf("SMTP SenderName kustom salah: diharapkan 'Custom Mailer', didapat '%s'", cfg.SMTP.SenderName)
	}

	if cfg.Auth.JWTSecret != "supersecretjwt" {
		t.Errorf("Auth JWTSecret kustom salah: diharapkan 'supersecretjwt', didapat '%s'", cfg.Auth.JWTSecret)
	}
	if cfg.Auth.AdminUsername != "superadmin" {
		t.Errorf("Auth AdminUsername kustom salah: diharapkan 'superadmin', didapat '%s'", cfg.Auth.AdminUsername)
	}
	if cfg.Auth.AdminPassword != "adminpassword" {
		t.Errorf("Auth AdminPassword kustom salah: diharapkan 'adminpassword', didapat '%s'", cfg.Auth.AdminPassword)
	}
	if cfg.Auth.PetugasUsername != "superpetugas" {
		t.Errorf("Auth PetugasUsername kustom salah: diharapkan 'superpetugas', didapat '%s'", cfg.Auth.PetugasUsername)
	}
	if cfg.Auth.PetugasPassword != "petugaspassword" {
		t.Errorf("Auth PetugasPassword kustom salah: diharapkan 'petugaspassword', didapat '%s'", cfg.Auth.PetugasPassword)
	}

	expectedDSN := "host=postgres.internal port=5433 user=testuser password=testpass dbname=testdb sslmode=require"
	if dsn := cfg.Database.DSN(); dsn != expectedDSN {
		t.Errorf("DSN kustom salah: diharapkan '%s', didapat '%s'", expectedDSN, dsn)
	}
}
