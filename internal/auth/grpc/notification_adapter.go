package grpc

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
)

type NotificationAdapter struct {
	client notification.NotificationServiceClient
	appURL string
}

func NewNotificationAdapter(client notification.NotificationServiceClient, appURL string) *NotificationAdapter {
	return &NotificationAdapter{client: client, appURL: appURL}
}

func (a *NotificationAdapter) SendVerification(email, token string) error {
	link := fmt.Sprintf("%s/api/v1/auth/confirm?token=%s", a.appURL, token)

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

	_, err := a.client.SendEmail(context.Background(), &notification.SendEmailRequest{
		To:        email,
		Subject:   subject,
		TextBody:  textBody,
		HtmlBody:  htmlBody,
		EmailType: notification.EmailType_EMAIL_TYPE_VERIFICATION,
	})
	return err
}
