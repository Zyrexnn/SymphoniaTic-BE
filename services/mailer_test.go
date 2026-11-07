package services

import (
	"fmt"
	"net/smtp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

// resetMailerForTest memastikan isolasi state worker pool dan mock sender pada setiap test case
func resetMailerForTest(t *testing.T) {
	t.Helper()
	ShutdownMailer()

	mailerLock.Lock()
	origSender := emailSender
	origConfig := smtpConfig
	emailQueue = nil
	isClosed = false
	mailerLock.Unlock()

	t.Cleanup(func() {
		ShutdownMailer()
		mailerLock.Lock()
		emailSender = origSender
		smtpConfig = origConfig
		emailQueue = nil
		isClosed = false
		mailerLock.Unlock()
	})
}

func TestRenderTemplate_AllTemplates(t *testing.T) {
	testCases := []struct {
		name         string
		templateName string
		data         interface{}
		mustContain  []string
	}{
		{
			name:         "Render E-Ticket Template",
			templateName: "eticket.html",
			data: map[string]interface{}{
				"UserName":     "Ahmad Dhani",
				"EventTitle":   "Simfoni Mahakarya 2026",
				"OrderCode":    "ORD-TEST-999",
				"Artist":       "Twilite Orchestra",
				"Venue":        "Aula Simfonia Jakarta",
				"Date":         "2026-12-15 19:30",
				"CategoryName": "VVIP Diamond",
				"Quantity":     2,
				"TotalPrice":   1500000.0,
			},
			mustContain: []string{
				"Ahmad Dhani",
				"Simfoni Mahakarya 2026",
				"ORD-TEST-999",
				"Twilite Orchestra",
				"Aula Simfonia Jakarta",
				"VVIP Diamond",
				"2 Tiket",
				"Rp 1500000",
			},
		},
		{
			name:         "Render Pengingat Acara Template",
			templateName: "reminder.html",
			data: map[string]interface{}{
				"UserName":   "Dewi Lestari",
				"EventTitle": "Konser Resital Piano",
				"OrderCode":  "ORD-REMIND-001",
				"Date":       "2026-11-20 20:00",
				"Venue":      "Teater Besar TIM",
				"Address":    "Jl. Cikini Raya No. 73",
				"MapLink":    "https://maps.google.com/?q=Teater+Besar+TIM",
			},
			mustContain: []string{
				"Dewi Lestari",
				"Konser Resital Piano",
				"ORD-REMIND-001",
				"2026-11-20 20:00",
				"Teater Besar TIM",
				"Jl. Cikini Raya No. 73",
				"https://maps.google.com/?q=Teater",
			},
		},
		{
			name:         "Render Refund OTP Template",
			templateName: "refund_otp.html",
			data: map[string]interface{}{
				"OrderCode": "ORD-REFUND-777",
				"OTPCode":   "482910",
			},
			mustContain: []string{
				"ORD-REFUND-777",
				"482910",
				"10 menit",
			},
		},
		{
			name:         "Render Refund Status Template",
			templateName: "refund_status.html",
			data: map[string]interface{}{
				"OrderCode":   "ORD-STATUS-555",
				"StatusTitle": "DISETUJUI",
				"BadgeColor":  "#10b981",
				"Amount":      350000.0,
				"AdminNote":   "Pengembalian dana diproses ke rekening pemohon",
			},
			mustContain: []string{
				"ORD-STATUS-555",
				"STATUS: DISETUJUI",
				"#10b981",
				"Rp 350000.00",
				"Pengembalian dana diproses ke rekening pemohon",
			},
		},
		{
			name:         "Render Auth OTP Template",
			templateName: "auth_otp.html",
			data: map[string]interface{}{
				"Title":       "VERIFIKASI PENDAFTARAN AKUN",
				"Subtitle":    "Kode OTP Pendaftaran",
				"NameDisplay": "Budi Santoso",
				"Warning":     "Gunakan kode OTP ini untuk verifikasi pendaftaran.",
				"OTPCode":     "112233",
			},
			mustContain: []string{
				"VERIFIKASI PENDAFTARAN AKUN",
				"Kode OTP Pendaftaran",
				"Budi Santoso",
				"112233",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := renderTemplate(tc.templateName, tc.data)
			if err != nil {
				t.Fatalf("Gagal merender template %s: %v", tc.templateName, err)
			}

			for _, expected := range tc.mustContain {
				if !strings.Contains(rendered, expected) {
					t.Errorf("Hasil render template %s tidak mengandung string yang diharapkan: %q", tc.templateName, expected)
				}
			}
		})
	}
}

func TestRenderTemplate_NotFound(t *testing.T) {
	_, err := renderTemplate("template_yang_tidak_ada.html", nil)
	if err == nil {
		t.Fatal("Diharapkan error saat template tidak ditemukan, namun mendapatkan nil")
	}
}

func TestInitMailerWorkerPool_And_Shutdown(t *testing.T) {
	resetMailerForTest(t)

	cfg := &config.SMTPConfig{
		Host:        "localhost",
		Port:        "1025",
		SenderEmail: "test@symphoniatic.com",
		SenderName:  "Test Mailer",
	}

	var sentCount int32
	var wg sync.WaitGroup
	wg.Add(5)

	emailSender = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		atomic.AddInt32(&sentCount, 1)
		wg.Done()
		return nil
	}

	InitMailerWorkerPool(cfg, 3, 20)

	for i := 0; i < 5; i++ {
		enqueueEmail(EmailJob{
			To:      fmt.Sprintf("user%d@example.com", i),
			Subject: fmt.Sprintf("Subjek Pengujian %d", i),
			Body:    "<p>Konten Pengujian</p>",
		})
	}

	// Menunggu seluruh worker menyelesaikan pemrosesan antrean
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout menunggu pemrosesan email pada worker pool")
	}

	if atomic.LoadInt32(&sentCount) != 5 {
		t.Fatalf("Jumlah email terkirim tidak sesuai: diharapkan 5, didapat %d", sentCount)
	}

	// Memverifikasi graceful shutdown menutup channel tanpa menimbulkan race condition atau panic
	ShutdownMailer()

	// Pemanggilan berikutnya setelah shutdown harus ditolak dengan aman tanpa panic
	enqueueEmail(EmailJob{
		To:      "after_shutdown@example.com",
		Subject: "Uji Pasca Shutdown",
		Body:    "<p>Tidak terkirim</p>",
	})

	if atomic.LoadInt32(&sentCount) != 5 {
		t.Fatalf("Pekerjaan pasca shutdown seharusnya tidak diproses, didapat %d", sentCount)
	}
}

func TestPublicSendFunctions(t *testing.T) {
	resetMailerForTest(t)

	cfg := &config.SMTPConfig{
		Host:        "127.0.0.1",
		Port:        "1025",
		SenderEmail: "system@symphoniatic.com",
		SenderName:  "SymphoniaTic System",
	}

	type recordedMail struct {
		addr string
		from string
		to   []string
		msg  string
	}

	var mu sync.Mutex
	var sentMails []recordedMail
	var wg sync.WaitGroup
	wg.Add(5)

	emailSender = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		mu.Lock()
		sentMails = append(sentMails, recordedMail{
			addr: addr,
			from: from,
			to:   to,
			msg:  string(msg),
		})
		mu.Unlock()
		wg.Done()
		return nil
	}

	InitMailerWorkerPool(cfg, 2, 50)

	// 1. SendETicketEmail
	order := models.OrderRecord{
		OrderCode:    "ORD-PUB-101",
		UserEmail:    "buyer@example.com",
		UserName:     "Pembeli Resmi",
		EventTitle:   "Konser Simfoni No. 5",
		Artist:       "Ludwig Orchestra",
		Venue:        "Gedung Kesenian Jakarta",
		Date:         "2026-12-01 19:00",
		CategoryName: "VIP Gold",
		Quantity:     1,
		TotalPrice:   350000,
	}
	SendETicketEmail(order)

	// 2. SendEventReminderEmail
	SendEventReminderEmail(
		"buyer@example.com",
		"Pembeli Resmi",
		"Konser Simfoni No. 5",
		"Gedung Kesenian Jakarta",
		"Jl. Pasar Baru No. 1",
		"2026-12-01 19:00",
		"ORD-PUB-101",
	)

	// 3. SendRefundOTPEmail
	SendRefundOTPEmail("buyer@example.com", "ORD-PUB-101", "882910")

	// 4. SendRefundStatusNotificationEmail
	SendRefundStatusNotificationEmail("buyer@example.com", "ORD-PUB-101", "APPROVED", "Disetujui otomatis", 350000)

	// 5. SendAuthOTPEmail
	SendAuthOTPEmail("buyer@example.com", "Pembeli Resmi", "445566", "REGISTER")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout menunggu pemrosesan fungsi publik email")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(sentMails) != 5 {
		t.Fatalf("Jumlah pengiriman email publik salah: diharapkan 5, didapat %d", len(sentMails))
	}

	var (
		foundETicket   bool
		foundReminder  bool
		foundRefundOTP bool
		foundRefund    bool
		foundAuthOTP   bool
	)

	for _, mail := range sentMails {
		if strings.Contains(mail.msg, "E-TICKET RESMI") && strings.Contains(mail.msg, "ORD-PUB-101") {
			foundETicket = true
		}
		if strings.Contains(mail.msg, "PENGINGAT") && strings.Contains(mail.msg, "Gedung Kesenian Jakarta") {
			foundReminder = true
		}
		if strings.Contains(mail.msg, "KODE OTP REFUND") && strings.Contains(mail.msg, "882910") {
			foundRefundOTP = true
		}
		if strings.Contains(mail.msg, "UPDATE STATUS REFUND") && strings.Contains(mail.msg, "DISETUJUI") {
			foundRefund = true
		}
		if strings.Contains(mail.msg, "REGISTER") && strings.Contains(mail.msg, "445566") {
			foundAuthOTP = true
		}
	}

	if !foundETicket {
		t.Error("Pesan e-ticket tidak ditemukan dalam daftar terkirim")
	}
	if !foundReminder {
		t.Error("Pesan pengingat acara tidak ditemukan dalam daftar terkirim")
	}
	if !foundRefundOTP {
		t.Error("Pesan OTP refund tidak ditemukan dalam daftar terkirim")
	}
	if !foundRefund {
		t.Error("Pesan status refund tidak ditemukan dalam daftar terkirim")
	}
	if !foundAuthOTP {
		t.Error("Pesan auth OTP tidak ditemukan dalam daftar terkirim")
	}
}

func TestEnqueueEmail_FullQueueFallback(t *testing.T) {
	resetMailerForTest(t)

	var sentCount int32
	var wg sync.WaitGroup
	wg.Add(4)

	emailSender = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		atomic.AddInt32(&sentCount, 1)
		wg.Done()
		return nil
	}

	// Inisialisasi pool dengan kapasitas buffer hanya 1 slot dan tanpa worker aktif
	// untuk memaksa kondisi antrean penuh sehingga fallback non-blocking terpicu
	mailerLock.Lock()
	emailQueue = make(chan EmailJob, 1)
	isClosed = false
	// Isi slot antrean agar slot penuh
	emailQueue <- EmailJob{To: "filled@example.com", Subject: "Slot Penuh", Body: "Buffer"}
	mailerLock.Unlock()

	// 3 pekerjaan berikutnya akan memicu fallback default select tanpa blocking
	for i := 0; i < 3; i++ {
		enqueueEmail(EmailJob{
			To:      fmt.Sprintf("fallback%d@example.com", i),
			Subject: "Fallback Job",
			Body:    "Body",
		})
	}

	// Jalankan satu worker sementara untuk mengonsumsi 1 slot awal
	go func() {
		job := <-emailQueue
		sendEmailViaSMTP(job)
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout menunggu pemrosesan fallback antrean penuh")
	}

	if atomic.LoadInt32(&sentCount) != 4 {
		t.Fatalf("Jumlah pengiriman email fallback tidak sesuai: diharapkan 4, didapat %d", sentCount)
	}
}
