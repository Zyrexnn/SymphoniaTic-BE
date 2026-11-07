package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

type SMTPConfig struct {
	Host        string
	Port        string
	SenderEmail string
	SenderName  string
}

type AuthConfig struct {
	JWTSecret       string
	AdminUsername   string
	AdminPassword   string
	PetugasUsername string
	PetugasPassword string
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	SMTP     SMTPConfig
	Auth     AuthConfig
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// LoadConfig memuat konfigurasi dari variabel lingkungan dengan toleransi jika berkas .env tidak ada
func LoadConfig(filenames ...string) (*Config, error) {
	if len(filenames) > 0 {
		_ = godotenv.Load(filenames...)
	} else {
		_ = godotenv.Load()
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8082"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		SMTP: SMTPConfig{
			Host:        getEnv("SMTP_HOST", "localhost"),
			Port:        getEnv("SMTP_PORT", "1025"),
			SenderEmail: getEnv("SMTP_SENDER_EMAIL", "noreply@symphoniatic.com"),
			SenderName:  getEnv("SMTP_SENDER_NAME", "SymphoniaTic E-Ticket System"),
		},
		Auth: AuthConfig{
			JWTSecret:       os.Getenv("JWT_SECRET"),
			AdminUsername:   os.Getenv("ADMIN_USERNAME"),
			AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
			PetugasUsername: os.Getenv("PETUGAS_USERNAME"),
			PetugasPassword: os.Getenv("PETUGAS_PASSWORD"),
		},
	}

	return cfg, nil
}
