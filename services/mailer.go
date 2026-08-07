package services

import (
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

// SendETicketEmail sends an HTML formatted E-Ticket email via SMTP (Mailpit compatible) asynchronously.
func SendETicketEmail(order models.OrderRecord) {
	go func() {
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
		if senderEmail == "" {
			senderEmail = "noreply@symphoniatic.com"
		}
		senderName := os.Getenv("SMTP_SENDER_NAME")
		if senderName == "" {
			senderName = "SymphoniaTic E-Ticket System"
		}

		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

		subject := fmt.Sprintf("E-TICKET RESMI [%s] - %s", order.OrderCode, order.EventTitle)

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>E-Ticket SymphoniaTic</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0b0f19; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #131b2e; border-radius: 16px; border: 1px solid #1e293b; overflow: hidden; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
        .header { background: linear-gradient(135deg, #1e1b4b 0%%, #312e81 100%%); padding: 30px 20px; text-align: center; border-bottom: 2px solid #6366f1; }
        .header h1 { margin: 0; color: #fbbf24; font-size: 24px; letter-spacing: 2px; }
        .header p { margin: 5px 0 0 0; color: #94a3b8; font-size: 14px; }
        .content { padding: 30px 25px; }
        .ticket-box { background: #0f172a; border: 1px dashed #6366f1; border-radius: 12px; padding: 20px; margin-bottom: 25px; }
        .ticket-code { font-size: 22px; font-weight: bold; color: #38bdf8; text-align: center; letter-spacing: 3px; margin-bottom: 15px; }
        .detail-row { display: flex; justify-content: space-between; margin-bottom: 10px; font-size: 14px; border-bottom: 1px solid #1e293b; padding-bottom: 6px; }
        .label { color: #94a3b8; }
        .value { color: #f8fafc; font-weight: 600; text-align: right; }
        .qr-section { text-align: center; margin: 25px 0; background: #ffffff; padding: 15px; border-radius: 12px; display: inline-block; }
        .qr-section img { max-width: 160px; height: auto; }
        .footer { background: #090d16; padding: 20px; text-align: center; font-size: 12px; color: #64748b; }
        .badge { background: #059669; color: #ffffff; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>SYMPHONIATIC</h1>
            <p>E-Ticket Resmi Konser & Pertunjukan</p>
        </div>
        <div class="content">
            <p>Halo <strong>%s</strong>,</p>
            <p>Pembayaran transaksi Anda telah terverifikasi. Berikut adalah e-ticket resmi Anda untuk menyaksikan pertunjukan <strong>%s</strong>.</p>
            
            <div class="ticket-box">
                <div class="ticket-code">%s</div>
                <div style="text-align: center; margin-bottom: 15px;">
                    <span class="badge">STATUS: VERIFIED</span>
                </div>
                
                <div class="detail-row">
                    <span class="label">Konser / Event:</span>
                    <span class="value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="label">Artis / Performa:</span>
                    <span class="value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="label">Lokasi (Venue):</span>
                    <span class="value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="label">Waktu:</span>
                    <span class="value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="label">Kategori Tiket:</span>
                    <span class="value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="label">Jumlah Tiket:</span>
                    <span class="value">%d Tiket</span>
                </div>
                <div class="detail-row" style="border-bottom: none;">
                    <span class="label">Total Bayar:</span>
                    <span class="value" style="color: #fbbf24;">Rp %.0f</span>
                </div>
            </div>

            <div style="text-align: center;">
                <div class="qr-section">
                    <img src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=%s" alt="QR Code E-Ticket" />
                    <div style="color: #000; font-size: 11px; margin-top: 5px; font-weight: bold;">Tunjukkan QR ini saat masuk venue</div>
                </div>
            </div>

            <p style="font-size: 13px; color: #94a3b8; text-align: center;">
                Harap simpan email ini atau unduh kode QR di atas untuk verifikasi di lokasi konser.
            </p>
        </div>
        <div class="footer">
            &copy; 2026 SymphoniaTic. Seluruh Hak Cipta Dilindungi.<br/>
            Email dikirim otomatis oleh Sistem SymphoniaTic.
        </div>
    </div>
</body>
</html>`,
			order.UserName,
			order.EventTitle,
			order.OrderCode,
			order.EventTitle,
			order.Artist,
			order.Venue,
			order.Date,
			order.CategoryName,
			order.Quantity,
			order.TotalPrice,
			order.QRCode,
		)

		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = order.UserEmail
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"

		message := ""
		for k, v := range headers {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		message += "\r\n" + htmlBody

		err := smtp.SendMail(addr, nil, senderEmail, []string{order.UserEmail}, []byte(message))
		if err != nil {
			log.Printf("[MAILPIT-ERROR] Gagal mengirim e-ticket ke %s: %v\n", order.UserEmail, err)
		} else {
			log.Printf("[MAILPIT-SUCCESS] E-Ticket email [%s] berhasil dikirim ke Mailpit (%s) untuk %s!\n", order.OrderCode, addr, order.UserEmail)
		}
	}()
}

// SendEventReminderEmail sends an automated H-1 event reminder with map location & rundown via Mailpit.
func SendEventReminderEmail(userEmail, userName, eventTitle, venue, address, date, orderCode string) {
	go func() {
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
		if senderEmail == "" {
			senderEmail = "noreply@symphoniatic.com"
		}
		senderName := os.Getenv("SMTP_SENDER_NAME")
		if senderName == "" {
			senderName = "SymphoniaTic Reminder System"
		}

		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
		subject := fmt.Sprintf("⏰ PENGINGAT H-1 KONSER: %s [%s]", eventTitle, orderCode)

		mapLink := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", fmt.Sprintf("%s %s", eventTitle, venue))

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>Pengingat H-1 Konser SymphoniaTic</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0b0f19; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #131b2e; border-radius: 16px; border: 1px solid #1e293b; overflow: hidden; }
        .header { background: linear-gradient(135deg, #312e81 0%%, #4338ca 100%%); padding: 25px 20px; text-align: center; }
        .header h1 { margin: 0; color: #fbbf24; font-size: 22px; }
        .content { padding: 25px; }
        .box { background: #0f172a; border: 1px solid #334155; border-radius: 10px; padding: 15px; margin: 15px 0; }
        .btn { display: inline-block; background: #6366f1; color: #ffffff; padding: 10px 20px; text-decoration: none; border-radius: 8px; font-weight: bold; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>PENGINGAT KONSER H-1</h1>
            <p style="margin: 5px 0 0 0; color: #cbd5e1;">Besok adalah Hari Pertunjukan!</p>
        </div>
        <div class="content">
            <p>Halo <strong>%s</strong>,</p>
            <p>Konser <strong>%s</strong> yang Anda pesan (%s) akan dilaksanakan besok pada <strong>%s</strong>.</p>
            
            <div class="box">
                <p style="margin:0; font-weight:bold; color:#38bdf8;">📍 Petunjuk Lokasi &amp; Venue:</p>
                <p style="margin:5px 0 0 0; color:#f8fafc;">%s (%s)</p>
                <a href="%s" class="btn" target="_blank">Buka Petunjuk Peta Google Maps</a>
            </div>

            <p style="font-size:13px; color:#94a3b8; text-align:center;">
                Pastikan Anda menyiapkan QR Code E-Ticket pada smartphone Anda untuk proses registrasi di Open Gate. Sampai jumpa di konser!
            </p>
        </div>
    </div>
</body>
</html>`, userName, eventTitle, orderCode, date, venue, address, mapLink)

		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = userEmail
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"

		message := ""
		for k, v := range headers {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		message += "\r\n" + htmlBody

		_ = smtp.SendMail(addr, nil, senderEmail, []string{userEmail}, []byte(message))
		log.Printf("[MAILPIT-REMINDER] Email Pengingat H-1 [%s] berhasil dikirim ke %s\n", orderCode, userEmail)
	}()
}

// SendRefundOTPEmail sends a 6-digit OTP code for refund request verification.
func SendRefundOTPEmail(userEmail, orderCode, otpCode string) {
	go func() {
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
		if senderEmail == "" {
			senderEmail = "noreply@symphoniatic.com"
		}
		senderName := os.Getenv("SMTP_SENDER_NAME")
		if senderName == "" {
			senderName = "SymphoniaTic Refund Security"
		}

		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
		subject := fmt.Sprintf("🔐 KODE OTP REFUND TIKET [%s] - %s", orderCode, otpCode)

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>Kode OTP Refund SymphoniaTic</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0b0f19; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 550px; margin: 0 auto; background: #131b2e; border-radius: 16px; border: 1px solid #1e293b; overflow: hidden; }
        .header { background: linear-gradient(135deg, #1e1b4b 0%%, #312e81 100%%); padding: 25px 20px; text-align: center; border-bottom: 2px solid #6366f1; }
        .header h1 { margin: 0; color: #fbbf24; font-size: 22px; }
        .content { padding: 25px; text-align: center; }
        .otp-box { background: #0f172a; border: 2px dashed #6366f1; border-radius: 12px; padding: 20px; margin: 20px 0; display: inline-block; }
        .otp-code { font-size: 32px; font-weight: bold; color: #38bdf8; letter-spacing: 8px; }
        .warning { font-size: 13px; color: #f59e0b; margin-top: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>SYMPHONIATIC REFUND</h1>
            <p style="margin:5px 0 0 0; color:#94a3b8; font-size:14px;">Kode Verifikasi Keamanan Pengajuan Refund Tiket</p>
        </div>
        <div class="content">
            <p style="color:#cbd5e1; text-align:left;">Anda (atau seseorang) baru saja mengajukan permintaan refund untuk Kode Pesanan: <strong>%s</strong>.</p>
            <p style="color:#cbd5e1; text-align:left;">Gunakan kode OTP di bawah ini untuk melanjutkan formulir pengajuan refund:</p>
            
            <div class="otp-box">
                <div class="otp-code">%s</div>
            </div>

            <p style="font-size:13px; color:#94a3b8;">Kode OTP ini berlaku selama <strong>10 menit</strong>. Jangan bagikan kode ini kepada siapapun demi keamanan transaksi Anda.</p>
            <p class="warning">⚠️ Jika Anda tidak pernah merasa mengajukan refund, abaikan email ini.</p>
        </div>
    </div>
</body>
</html>`, orderCode, otpCode)

		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = userEmail
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"

		message := ""
		for k, v := range headers {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		message += "\r\n" + htmlBody

		_ = smtp.SendMail(addr, nil, senderEmail, []string{userEmail}, []byte(message))
		log.Printf("[MAILPIT-REFUND-OTP] Email OTP Refund [%s] berhasil dikirim ke %s\n", orderCode, userEmail)
	}()
}

// SendRefundStatusNotificationEmail notifies user about their refund status (APPROVED/REJECTED).
func SendRefundStatusNotificationEmail(userEmail, orderCode, status, adminNote string, amount float64) {
	go func() {
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
		if senderEmail == "" {
			senderEmail = "noreply@symphoniatic.com"
		}
		senderName := os.Getenv("SMTP_SENDER_NAME")
		if senderName == "" {
			senderName = "SymphoniaTic Finance"
		}

		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
		statusTitle := "DISETUJUI"
		badgeColor := "#10b981"
		if status == "REJECTED" {
			statusTitle = "DITOLAK"
			badgeColor = "#ef4444"
		}

		subject := fmt.Sprintf("📢 UPDATE STATUS REFUND TIKET [%s] - %s", orderCode, statusTitle)

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>Update Status Refund SymphoniaTic</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0b0f19; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 550px; margin: 0 auto; background: #131b2e; border-radius: 16px; border: 1px solid #1e293b; overflow: hidden; }
        .header { background: linear-gradient(135deg, #1e1b4b 0%%, #312e81 100%%); padding: 25px 20px; text-align: center; border-bottom: 2px solid #6366f1; }
        .header h1 { margin: 0; color: #fbbf24; font-size: 22px; }
        .content { padding: 25px; }
        .status-badge { display: inline-block; background-color: %s; color: #ffffff; padding: 6px 16px; border-radius: 20px; font-weight: bold; font-size: 14px; margin: 15px 0; }
        .box { background: #0f172a; border: 1px solid #334155; border-radius: 10px; padding: 15px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>SYMPHONIATIC REFUND UPDATE</h1>
            <p style="margin:5px 0 0 0; color:#94a3b8; font-size:14px;">Kode Pesanan: %s</p>
        </div>
        <div class="content">
            <p>Halo,</p>
            <p>Pengajuan refund tiket Anda untuk pesanan <strong>%s</strong> telah diproses oleh tim finance kami.</p>
            
            <div style="text-align:center;">
                <span class="status-badge">STATUS: %s</span>
            </div>

            <div class="box">
                <p style="margin:0; font-size:13px; color:#94a3b8;">Nominal Refund:</p>
                <p style="margin:4px 0 10px 0; font-size:18px; font-weight:bold; color:#38bdf8;">Rp %.2f</p>
                <p style="margin:0; font-size:13px; color:#94a3b8;">Catatan Admin:</p>
                <p style="margin:4px 0 0 0; font-size:14px; color:#f8fafc;">%s</p>
            </div>

            <p style="font-size:13px; color:#94a3b8; text-align:center;">Jika Anda memiliki pertanyaan lebih lanjut, silakan hubungi tim bantuan kami di support@symphoniatic.id.</p>
        </div>
    </div>
</body>
</html>`, badgeColor, orderCode, orderCode, statusTitle, amount, adminNote)

		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = userEmail
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"

		message := ""
		for k, v := range headers {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		message += "\r\n" + htmlBody

		_ = smtp.SendMail(addr, nil, senderEmail, []string{userEmail}, []byte(message))
		log.Printf("[MAILPIT-REFUND-STATUS] Email Status Refund [%s] berhasil dikirim ke %s\n", orderCode, userEmail)
	}()
}

// SendAuthOTPEmail sends an HTML OTP email for Register, Login, or Forgot Password via Mailpit.
func SendAuthOTPEmail(userEmail, userName, otpCode, purpose string) {
	go func() {
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			smtpHost = "localhost"
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "1025"
		}
		senderEmail := os.Getenv("SMTP_SENDER_EMAIL")
		if senderEmail == "" {
			senderEmail = "noreply@symphoniatic.com"
		}
		senderName := os.Getenv("SMTP_SENDER_NAME")
		if senderName == "" {
			senderName = "SymphoniaTic Auth System"
		}

		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

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

		subject := fmt.Sprintf("🔐 [%s] KODE OTP: %s - SYMPHONIATIC", purpose, otpCode)

		nameDisplay := userName
		if nameDisplay == "" {
			nameDisplay = userEmail
		}

		htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #0b0f19; color: #e2e8f0; margin: 0; padding: 20px; }
        .container { max-width: 550px; margin: 0 auto; background: #131b2e; border-radius: 16px; border: 1px solid #1e293b; overflow: hidden; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
        .header { background: linear-gradient(135deg, #1e1b4b 0%%%%, #312e81 100%%%%); padding: 25px 20px; text-align: center; border-bottom: 2px solid #6366f1; }
        .header h1 { margin: 0; color: #fbbf24; font-size: 22px; letter-spacing: 2px; }
        .header p { margin: 5px 0 0 0; color: #94a3b8; font-size: 13px; }
        .content { padding: 30px 25px; text-align: center; }
        .greeting { text-align: left; color: #cbd5e1; font-size: 15px; margin-bottom: 15px; }
        .otp-box { background: #0f172a; border: 2px dashed #6366f1; border-radius: 12px; padding: 20px; margin: 25px 0; display: inline-block; width: 80%%%%; }
        .otp-code { font-size: 36px; font-weight: bold; color: #38bdf8; letter-spacing: 10px; font-family: monospace; }
        .info { font-size: 13px; color: #94a3b8; margin-top: 15px; }
        .alert-warning { font-size: 12px; color: #f59e0b; background: rgba(245, 158, 11, 0.1); border: 1px solid rgba(245, 158, 11, 0.3); border-radius: 8px; padding: 10px; margin-top: 20px; text-align: left; }
        .footer { background: #090d16; padding: 18px; text-align: center; font-size: 12px; color: #64748b; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>SYMPHONIATIC</h1>
            <p>%s</p>
        </div>
        <div class="content">
            <div class="greeting">Halo <strong>%s</strong>,</div>
            <p style="text-align: left; color: #cbd5e1; font-size: 14px; margin: 0;">%s</p>

            <div class="otp-box">
                <div class="otp-code">%s</div>
            </div>

            <p class="info">Kode OTP ini berlaku selama <strong>5 menit</strong>. Jangan berikan kode ini kepada siapapun demi menjaga keamanan akun Anda.</p>

            <div class="alert-warning">
                ⚠️ Jika Anda tidak merasa melakukan permintaan kode ini, abaikan email ini dan pastikan kata sandi Anda tetap aman.
            </div>
        </div>
        <div class="footer">
            &copy; 2026 SymphoniaTic Security System. Seluruh Hak Cipta Dilindungi.
        </div>
    </div>
</body>
</html>`, title, subtitle, nameDisplay, warning, otpCode)

		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = userEmail
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"

		message := ""
		for k, v := range headers {
			message += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		message += "\r\n" + htmlBody

		err := smtp.SendMail(addr, nil, senderEmail, []string{userEmail}, []byte(message))
		if err != nil {
			log.Printf("[MAILPIT-ERROR] Gagal mengirim OTP email [%s] ke %s: %v\n", purpose, userEmail, err)
		} else {
			log.Printf("[MAILPIT-SUCCESS] OTP email [%s] berhasil dikirim ke %s via Mailpit (%s)\n", purpose, userEmail, addr)
		}
	}()
}


