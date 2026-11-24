package services

import (
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Zyrexnn/SymphoniaTic-be/config"
	"github.com/Zyrexnn/SymphoniaTic-be/models"
	qrcode "github.com/skip2/go-qrcode"
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

// EnqueueEmailSafe memasukkan pekerjaan email ke antrean secara aman tanpa risiko panic bila antrean tertutup
func EnqueueEmailSafe(job EmailJob) bool {
	mailerLock.Lock()
	if isClosed {
		mailerLock.Unlock()
		log.Printf("Peringatan: Antrean email telah ditutup, membatalkan pengiriman email ke %s\n", job.To)
		return false
	}
	mailerLock.Unlock()
	enqueueEmail(job)
	return true
}

// sendEmailViaSMTP menyusun payload MIME dan mengirimkan email via protokol SMTP
func sendEmailViaSMTP(job EmailJob) {
	_ = sendEmailViaSMTPWithError(job)
}

func sendEmailViaSMTPWithError(job EmailJob) error {
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

	headerSanitizer := strings.NewReplacer("\r", "", "\n", "")
	cleanTo := headerSanitizer.Replace(strings.TrimSpace(job.To))
	cleanSubject := headerSanitizer.Replace(strings.TrimSpace(job.Subject))

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
	headers["To"] = cleanTo
	headers["Subject"] = cleanSubject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n" + job.Body)

	err := emailSender(addr, nil, senderEmail, []string{cleanTo}, []byte(message.String()))
	if err != nil {
		log.Printf("[MAILPIT-ERROR] Gagal mengirim email ke %s: %v\n", job.To, err)
	} else {
		log.Printf("[MAILPIT-SUCCESS] Email [%s] berhasil dikirim ke %s via %s\n", job.Subject, job.To, addr)
	}
	return err
}

// SendETicketEmail mengirimkan email E-Ticket berformat HTML secara asinkron melalui antrean worker
func SendETicketEmail(order models.OrderRecord) {
	qrPayload := strings.TrimSpace(order.QRCode)
	if qrPayload == "" {
		qrPayload = strings.TrimSpace(order.OrderCode)
	}

	var qrDataURI template.URL
	if qrPayload != "" {
		pngBytes, err := qrcode.Encode(qrPayload, qrcode.Medium, 220)
		if err == nil {
			qrDataURI = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes))
		} else {
			log.Printf("Peringatan: Gagal membuat QR code untuk pesanan %s: %v\n", order.OrderCode, err)
		}
	}

	data := map[string]interface{}{
		"UserName":     order.UserName,
		"EventTitle":   order.EventTitle,
		"OrderCode":    order.OrderCode,
		"QRCode":       qrPayload,
		"QRDataURI":    qrDataURI,
		"Artist":       order.Artist,
		"Venue":        order.Venue,
		"Date":         order.Date,
		"Time":         order.EventTime,
		"OpenGate":     order.OpenGate,
		"VenueAddress": order.VenueAddress,
		"CategoryName": order.CategoryName,
		"SeatNumbers":  order.SeatNumbers,
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
	SendEventReminderEmailWithSchedule(userEmail, userName, eventTitle, venue, address, date, "", "", orderCode)
}

// SendEventReminderEmailWithSchedule mengirimkan pengingat H-1 konser beserta rincian waktu mulai dan open gate
func SendEventReminderEmailWithSchedule(userEmail, userName, eventTitle, venue, address, date, eventTime, openGate, orderCode string) {
	mapLink := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", fmt.Sprintf("%s %s", eventTitle, venue))
	data := map[string]interface{}{
		"UserName":   userName,
		"EventTitle": eventTitle,
		"OrderCode":  orderCode,
		"Date":       date,
		"Time":       eventTime,
		"OpenGate":   openGate,
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

func replaceBroadcastPlaceholders(raw string, recipient models.BroadcastRecipient, eventTitle string) string {
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		name = "Penikmat Musik"
	}
	r := strings.NewReplacer(
		"{{name}}", name,
		"{{email}}", strings.TrimSpace(recipient.Email),
		"{{orderCode}}", strings.TrimSpace(recipient.OrderCode),
		"{{eventTitle}}", strings.TrimSpace(eventTitle),
	)
	return r.Replace(raw)
}

// RenderDynamicBroadcastHTML merender email broadcast dinamis dengan HTML escaping ketat
func RenderDynamicBroadcastHTML(req models.CreateBroadcastRequest, eventTitle string, recipient models.BroadcastRecipient) (string, string) {
	subject := replaceBroadcastPlaceholders(strings.TrimSpace(req.Subject), recipient, eventTitle)
	headline := html.EscapeString(replaceBroadcastPlaceholders(strings.TrimSpace(req.Headline), recipient, eventTitle))
	rawBody := replaceBroadcastPlaceholders(strings.TrimSpace(req.BodyText), recipient, eventTitle)
	escapedBody := html.EscapeString(rawBody)
	formattedBody := strings.ReplaceAll(escapedBody, "\n", "<br/>")

	badgeLabel := "PENGUMUMAN"
	badgeBg := "#D4FF2A"
	badgeColor := "#141413"
	switch strings.ToUpper(strings.TrimSpace(req.Category)) {
	case "PROMO":
		badgeLabel = "PENAWARAN PROMO"
	case "EVENT_CANCELLED":
		badgeLabel = "PEMBATALAN KONSER"
		badgeBg = "#C81E1E"
		badgeColor = "#FFFFFF"
	case "EVENT_RESCHEDULED":
		badgeLabel = "PERUBAHAN JADWAL"
		badgeBg = "#8A5A00"
		badgeColor = "#FFFFFF"
	}

	promoHTML := ""
	if code := strings.TrimSpace(req.PromoCode); code != "" {
		promoHTML = fmt.Sprintf(`
		<div style="margin:24px 0;padding:16px 20px;background-color:#F4F3EF;border:1px solid #141413;">
			<div style="font-size:11px;letter-spacing:1.5px;text-transform:uppercase;color:#52514D;font-weight:700;">KODE PROMO EKSKLUSIF</div>
			<div style="margin-top:6px;font-size:22px;font-weight:800;letter-spacing:2px;color:#141413;font-family:'JetBrains Mono',ui-monospace,monospace;">%s</div>
		</div>`, html.EscapeString(strings.ToUpper(code)))
	}

	ctaHTML := ""
	if label, url := strings.TrimSpace(req.CTALabel), strings.TrimSpace(req.CTAURL); label != "" && url != "" {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "/") {
			ctaHTML = fmt.Sprintf(`
			<div style="margin-top:28px;">
				<a href="%s" style="display:inline-block;background-color:#141413;color:#D4FF2A;text-decoration:none;font-weight:700;font-size:13px;letter-spacing:1px;text-transform:uppercase;padding:14px 24px;border:1px solid #141413;">%s</a>
			</div>`, html.EscapeString(url), html.EscapeString(label))
		}
	}

	unsubscribeHTML := ""
	if strings.EqualFold(strings.TrimSpace(req.Category), "PROMO") && strings.TrimSpace(recipient.UnsubscribeToken) != "" {
		unsubHref := "/unsubscribe?token=" + html.EscapeString(strings.TrimSpace(recipient.UnsubscribeToken))
		unsubscribeHTML = fmt.Sprintf(`<div style="margin-top:8px;"><a href="%s" style="color:#52514D;text-decoration:underline;">Berhenti berlangganan email promo</a></div>`, unsubHref)
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width,initial-scale=1.0" /></head>
<body style="margin:0;padding:24px 12px;background-color:#F4F3EF;font-family:'Plus Jakarta Sans','Archivo',Arial,Helvetica,sans-serif;color:#141413;">
	<div style="max-width:600px;margin:0 auto;background-color:#FFFFFF;border:1px solid #141413;">
		<div style="background-color:#141413;padding:22px 28px;border-bottom:1px solid #141413;">
			<div style="font-size:18px;font-weight:800;letter-spacing:2px;color:#F4F3EF;">SYMPHONIATIC.</div>
			<div style="margin-top:10px;">
				<span style="display:inline-block;background-color:%s;color:%s;font-size:11px;font-weight:800;letter-spacing:1.2px;padding:4px 10px;font-family:'JetBrains Mono',ui-monospace,monospace;">%s</span>
			</div>
		</div>
		<div style="padding:28px;">
			<h1 style="margin:0 0 16px 0;font-size:22px;line-height:1.25;color:#141413;font-weight:800;">%s</h1>
			<div style="font-size:14px;line-height:1.65;color:#52514D;">%s</div>
			%s
			%s
		</div>
		<div style="padding:18px 28px;background-color:#F4F3EF;border-top:1px solid #DFDDD6;font-size:11px;color:#6E6D68;">
			Pesan ini dikirim secara resmi oleh sistem informasi SymphoniaTic.
			%s
		</div>
	</div>
</body>
</html>`, badgeBg, badgeColor, badgeLabel, headline, formattedBody, promoHTML, ctaHTML, unsubscribeHTML)

	return subject, htmlBody
}

// SendBroadcastBatchAsync mengirim email broadcast ke daftar penerima secara paralel terbatas dan memanggil onComplete setelah selesai
func SendBroadcastBatchAsync(req models.CreateBroadcastRequest, eventTitle string, recipients []models.BroadcastRecipient, onComplete func(sent, failed int)) {
	SendBroadcastBatchWithTrackerAsync(req, eventTitle, recipients, nil, onComplete)
}

// SendBroadcastBatchWithTrackerAsync mengirim email broadcast dengan pelacakan status per penerima
func SendBroadcastBatchWithTrackerAsync(
	req models.CreateBroadcastRequest,
	eventTitle string,
	recipients []models.BroadcastRecipient,
	onRecipientResult func(email, status, errMsg string),
	onComplete func(sent, failed int),
) {
	go func() {
		var sentCount int64
		var failedCount int64

		jobs := make(chan models.BroadcastRecipient, len(recipients))
		for _, r := range recipients {
			jobs <- r
		}
		close(jobs)

		workerCount := 5
		if len(recipients) < workerCount {
			workerCount = len(recipients)
		}
		if workerCount == 0 {
			if onComplete != nil {
				onComplete(0, 0)
			}
			return
		}

		var wg sync.WaitGroup
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for recipient := range jobs {
					subject, body := RenderDynamicBroadcastHTML(req, eventTitle, recipient)
					err := sendEmailViaSMTPWithError(EmailJob{
						To:      recipient.Email,
						Subject: subject,
						Body:    body,
					})
					if err != nil {
						atomic.AddInt64(&failedCount, 1)
						if onRecipientResult != nil {
							onRecipientResult(recipient.Email, "FAILED", err.Error())
						}
					} else {
						atomic.AddInt64(&sentCount, 1)
						if onRecipientResult != nil {
							onRecipientResult(recipient.Email, "SENT", "")
						}
					}
				}
			}()
		}
		wg.Wait()

		if onComplete != nil {
			onComplete(int(sentCount), int(failedCount))
		}
	}()
}
