package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"time"
)

type Service struct {
	from     string
	appURL   string
	smtpHost string
	smtpPort string
}

type Sender interface {
	SendVerification(email, token string) error
}

func New(from, appURL, smtpHost, smtpPort string) *Service {
	return &Service{
		from:     from,
		appURL:   appURL,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
	}
}

func (s *Service) SendVerification(email, token string) error {
	link := fmt.Sprintf("%s/api/v1/auth/confirm?token=%s", s.appURL, token)

	subject := "Подтверждение email"
	textBody := fmt.Sprintf(
		"Здравствуйте!\n\nПодтвердите email, перейдя по ссылке:\n%s\n\nЕсли вы не регистрировались, просто проигнорируйте это письмо.\n",
		link,
	)

	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="ru">
  <head>
    <meta charset="UTF-8">
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Подтверждение email</title>
  </head>
  <body style="margin:0; padding:0; background-color:#f6f8fb; font-family:Arial,Helvetica,sans-serif; color:#111827;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background-color:#f6f8fb; padding:32px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:560px; background:#ffffff; border-radius:16px; overflow:hidden; box-shadow:0 6px 24px rgba(0,0,0,0.08);">
            <tr>
              <td style="padding:32px 32px 24px 32px;">
                <div style="font-size:14px; color:#6b7280; margin-bottom:12px;">KISS Colab</div>
                <h1 style="margin:0 0 16px 0; font-size:24px; line-height:1.3; color:#111827;">Подтвердите email</h1>
                <p style="margin:0 0 24px 0; font-size:16px; line-height:1.6; color:#374151;">
                  Спасибо за регистрацию. Чтобы активировать аккаунт, нажмите на кнопку ниже.
                </p>
                <p style="margin:0 0 28px 0;">
                  <a href="%s" style="display:inline-block; background:#111827; color:#ffffff; text-decoration:none; padding:12px 20px; border-radius:10px; font-size:16px;">
                    Подтвердить email
                  </a>
                </p>
                <p style="margin:0; font-size:14px; line-height:1.6; color:#6b7280;">
                  Если кнопка не работает, откройте ссылку вручную:<br>
                  <a href="%s" style="color:#2563eb; word-break:break-all;">%s</a>
                </p>
              </td>
            </tr>
            <tr>
              <td style="padding:18px 32px 32px 32px; font-size:12px; line-height:1.6; color:#9ca3af;">
                Если вы не регистрировались, просто проигнорируйте это письмо.
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`, link, link, link)

	return s.sendMultipart(email, subject, textBody, htmlBody)
}

func (s *Service) sendMultipart(to, subject, textBody, htmlBody string) error {
	boundary := fmt.Sprintf("mixed-%d", time.Now().UnixNano())

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: KISS Colab <%s>\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	msg.WriteString("Reply-To: support@kisscolab.ru\r\n")
	msg.WriteString("\r\n")

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(textBody + "\r\n")

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	msg.WriteString(htmlBody + "\r\n")

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	return smtp.SendMail(addr, nil, s.from, []string{to}, msg.Bytes())
}
