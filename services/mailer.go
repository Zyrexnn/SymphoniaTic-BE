package services

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"sync"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

// EmailJob merepresentasikan muatan pekerjaan pengiriman email asinkron
type EmailJob struct {
	To      string
	Subject string
	Body    string
}

type smtpSenderFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

var (
	emailQueue  chan EmailJob
	workerWG    sync.WaitGroup
	mailerLock  sync.Mutex
	smtpConfig  *config.SMTPConfig
	isClosed    bool
	emailSender smtpSenderFunc = smtp.SendMail
)

// InitMailerWorkerPool menginisialisasi bounded pool goroutine untuk pengiriman email
func InitMailerWorkerPool(cfg *config.SMTPConfig, workerCount, queueSize int) {
	mailerLock.Lock()
	defer mailerLock.Unlock()

	// Mencegah kebocoran goroutine jika worker pool diinisialisasi ulang
	if emailQueue != nil && !isClosed {
		close(emailQueue)
		isClosed = true
		mailerLock.Unlock()
		workerWG.Wait()
		mailerLock.Lock()
	}

	if workerCount <= 0 {
		workerCount = 5
	}
	if queueSize <= 0 {
		queueSize = 200
	}

	smtpConfig = cfg
	emailQueue = make(chan EmailJob, queueSize)
	isClosed = false

	for i := 0; i < workerCount; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for job := range emailQueue {
				sendEmailViaSMTP(job)
			}
		}()
	}
}

// ShutdownMailer menutup antrean email dan menunggu seluruh worker menyelesaikan pekerjaan tersisa
func ShutdownMailer() {
	mailerLock.Lock()
	if isClosed || emailQueue == nil {
		mailerLock.Unlock()
		return
	}
	isClosed = true
	close(emailQueue)
	mailerLock.Unlock()

	workerWG.Wait()
}

// enqueueEmail memasukkan pekerjaan ke antrean secara non-blocking dengan fallback mandiri saat antrean penuh
func enqueueEmail(job EmailJob) {
	mailerLock.Lock()
	if isClosed {
		mailerLock.Unlock()
		log.Printf("Peringatan: Antrean email telah ditutup, membatalkan pengiriman email ke %s\n", job.To)
		return
	}

	// Alokasi instan jika pool belum dipanggil melalui InitMailerWorkerPool agar pengiriman ad-hoc tetap berjalan
	if emailQueue == nil {
		emailQueue = make(chan EmailJob, 200)
		isClosed = false
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for j := range emailQueue {
				sendEmailViaSMTP(j)
			}
		}()
	}

	select {
	case emailQueue <- job:
		mailerLock.Unlock()
	default:
		// Melepaskan lock sebelum fallback goroutine agar konkurensi request lain tidak terblokir
		mailerLock.Unlock()
		log.Printf("Peringatan: Antrean email penuh, menjalankan worker cadangan untuk %s\n", job.To)
		go sendEmailViaSMTP(job)
	}
}

// sendEmailViaSMTP menyusun payload MIME dan mengirimkan email via protokol SMTP
func sendEmailViaSMTP(job EmailJob) {
	smtpHost := "localhost"
	smtpPort := "1025"
	senderEmail := "noreply@symphoniatic.com"
	senderName := "SymphoniaTic Notification System"

	mailerLock.Lock()
	cfg := smtpConfig
	mailerLock.Unlock()

	if cfg != nil {
		if cfg.Host != "" {
			smtpHost = cfg.Host
		}
		if cfg.Port != "" {
			smtpPort = cfg.Port
		}
		if cfg.SenderEmail != "" {
			senderEmail = cfg.SenderEmail
		}
		if cfg.SenderName != "" {
			senderName = cfg.SenderName
		}
	} else {
		if h := os.Getenv("SMTP_HOST"); h != "" {
			smtpHost = h
		}
		if p := os.Getenv("SMTP_PORT"); p != "" {
			smtpPort = p
		}
		if se := os.Getenv("SMTP_SENDER_EMAIL"); se != "" {
			senderEmail = se
		}
		if sn := os.Getenv("SMTP_SENDER_NAME"); sn != "" {
			senderName = sn
		}
	}

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
	headers["To"] = job.To
	headers["Subject"] = job.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n" + job.Body)

	err := emailSender(addr, nil, senderEmail, []string{job.To}, []byte(message.String()))
	if err != nil {
		log.Printf("[MAILPIT-ERROR] Gagal mengirim email ke %s: %v\n", job.To, err)
	} else {
		log.Printf("[MAILPIT-SUCCESS] Email [%s] berhasil dikirim ke %s via %s\n", job.Subject, job.To, addr)
	}
}

// SendETicketEmail mengirimkan email E-Ticket berformat HTML secara asinkron melalui antrean worker
func SendETicketEmail(order models.OrderRecord) {
	data := map[string]interface{}{
		"UserName":     order.UserName,
		"EventTitle":   order.EventTitle,
		"OrderCode":    order.OrderCode,
		"Artist":       order.Artist,
		"Venue":        order.Venue,
		"Date":         order.Date,
		"CategoryName": order.CategoryName,
		"Quantity":     order.Quantity,
		"TotalPrice":   order.TotalPrice,
	}

	body, err := renderTemplate("eticket.html", data)
	if err != nil {
		log.Printf("Gagal merender template e-ticket untuk %s: %v\n", order.UserEmail, err)
		return
	}

	subject := fmt.Sprintf("E-TICKET RESMI [%s] - %s", order.OrderCode, order.EventTitle)
	enqueueEmail(EmailJob{
		To:      order.UserEmail,
		Subject: subject,
		Body:    body,
	})
}

// SendEventReminderEmail mengirimkan pengingat H-1 konser dengan peta lokasi dan petunjuk pelaksanaan
func SendEventReminderEmail(userEmail, userName, eventTitle, venue, address, date, orderCode string) {
	mapLink := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", fmt.Sprintf("%s %s", eventTitle, venue))
	data := map[string]interface{}{
		"UserName":   userName,
		"EventTitle": eventTitle,
		"OrderCode":  orderCode,
		"Date":       date,
		"Venue":      venue,
		"Address":    address,
		"MapLink":    mapLink,
	}

	body, err := renderTemplate("reminder.html", data)
	if err != nil {
		log.Printf("Gagal merender template pengingat acara untuk %s: %v\n", userEmail, err)
		return
	}

	subject := fmt.Sprintf("⏰ PENGINGAT H-1 KONSER: %s [%s]", eventTitle, orderCode)
	enqueueEmail(EmailJob{
		To:      userEmail,
		Subject: subject,
		Body:    body,
	})
}

// SendRefundOTPEmail mengirimkan kode verifikasi OTP 6-digit untuk otentikasi pengajuan refund
func SendRefundOTPEmail(userEmail, orderCode, otpCode string) {
	data := map[string]interface{}{
		"OrderCode": orderCode,
		"OTPCode":   otpCode,
	}

	body, err := renderTemplate("refund_otp.html", data)
	if err != nil {
		log.Printf("Gagal merender template OTP refund untuk %s: %v\n", userEmail, err)
		return
	}

	subject := fmt.Sprintf("🔐 KODE OTP REFUND TIKET [%s] - %s", orderCode, otpCode)
	enqueueEmail(EmailJob{
		To:      userEmail,
		Subject: subject,
		Body:    body,
	})
}

// SendRefundStatusNotificationEmail mengirimkan notifikasi keputusan status pengajuan refund beserta catatan admin
func SendRefundStatusNotificationEmail(userEmail, orderCode, status, adminNote string, amount float64) {
	statusTitle := "DISETUJUI"
	badgeColor := "#10b981"
	if status == "REJECTED" {
		statusTitle = "DITOLAK"
		badgeColor = "#ef4444"
	}

	data := map[string]interface{}{
		"OrderCode":   orderCode,
		"StatusTitle": statusTitle,
		"BadgeColor":  badgeColor,
		"Amount":      amount,
		"AdminNote":   adminNote,
	}

	body, err := renderTemplate("refund_status.html", data)
	if err != nil {
		log.Printf("Gagal merender template status refund untuk %s: %v\n", userEmail, err)
		return
	}

	subject := fmt.Sprintf("📢 UPDATE STATUS REFUND TIKET [%s] - %s", orderCode, statusTitle)
	enqueueEmail(EmailJob{
		To:      userEmail,
		Subject: subject,
		Body:    body,
	})
}

// SendAuthOTPEmail mengirimkan email OTP otentikasi pendaftaran, login, atau pengaturan ulang kata sandi
func SendAuthOTPEmail(userEmail, userName, otpCode, purpose string) {
	var title, subtitle, warning string
	switch purpose {
	case "REGISTER":
		title = "VERIFIKASI PENDAFTARAN AKUN"
		subtitle = "Kode OTP untuk Menyelesaikan Registrasi Akun SymphoniaTic"
		warning = "Gunakan kode OTP di bawah ini untuk memverifikasi pendaftaran akun Anda."
	case "LOGIN":
		title = "KODE OTP MASUK AKUN"
		subtitle = "Kode OTP Keamanan Login SymphoniaTic"
		warning = "Gunakan kode OTP di bawah ini untuk masuk ke akun Anda."
	case "FORGOT_PASSWORD":
		title = "RESET KATA SANDI AKUN"
		subtitle = "Kode OTP Pemulihan Kata Sandi SymphoniaTic"
		warning = "Gunakan kode OTP di bawah ini untuk memulihkan kata sandi akun Anda."
	default:
		title = "VERIFIKASI KEAMANAN"
		subtitle = "Kode OTP Keamanan SymphoniaTic"
		warning = "Gunakan kode OTP di bawah ini untuk memverifikasi identitas Anda."
	}

	nameDisplay := userName
	if nameDisplay == "" {
		nameDisplay = userEmail
	}

	data := map[string]interface{}{
		"Title":       title,
		"Subtitle":    subtitle,
		"NameDisplay": nameDisplay,
		"Warning":     warning,
		"OTPCode":     otpCode,
	}

	body, err := renderTemplate("auth_otp.html", data)
	if err != nil {
		log.Printf("Gagal merender template OTP otentikasi untuk %s: %v\n", userEmail, err)
		return
	}

	subject := fmt.Sprintf("🔐 [%s] KODE OTP: %s - SYMPHONIATIC", purpose, otpCode)
	enqueueEmail(EmailJob{
		To:      userEmail,
		Subject: subject,
		Body:    body,
	})
}
