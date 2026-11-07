package services

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"path/filepath"
)

//go:embed templates/*.html
var emailTemplatesFS embed.FS

var emailTemplates *template.Template

func init() {
	// Template fungsi bantuan untuk memformat angka mata uang dan nilai numerik dalam template HTML
	funcMap := template.FuncMap{
		"formatNumber": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				return fmt.Sprintf("%.0f", val)
			case float32:
				return fmt.Sprintf("%.0f", val)
			case int:
				return fmt.Sprintf("%d", val)
			case int64:
				return fmt.Sprintf("%d", val)
			default:
				return fmt.Sprintf("%v", val)
			}
		},
		"formatCurrency": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				return fmt.Sprintf("%.2f", val)
			case float32:
				return fmt.Sprintf("%.2f", val)
			default:
				return fmt.Sprintf("%v", val)
			}
		},
	}

	var err error
	emailTemplates, err = template.New("").Funcs(funcMap).ParseFS(emailTemplatesFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("Gagal memuat template email dari embedded FS: %v", err))
	}
}

// renderTemplate mengeksekusi template HTML yang tersimpan dalam binary menggunakan data yang disediakan
func renderTemplate(templateName string, data interface{}) (string, error) {
	if emailTemplates == nil {
		return "", fmt.Errorf("koleksi template email belum diinisialisasi")
	}

	cleanName := filepath.Base(templateName)

	var buf bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&buf, cleanName, data); err != nil {
		return "", fmt.Errorf("gagal merender template %s: %w", cleanName, err)
	}

	return buf.String(), nil
}
