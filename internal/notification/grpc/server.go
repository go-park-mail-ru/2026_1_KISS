package grpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
)

type SMTPSender interface {
	SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type defaultSMTPSender struct{}

func (defaultSMTPSender) SendMail(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil { //nolint:gosec
			return err
		}
	}

	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

type Server struct {
	pb.UnimplementedNotificationServiceServer
	from     string
	smtpHost string
	smtpPort string
	sender   SMTPSender
}

func NewServer(from, smtpHost, smtpPort string) *Server {
	return &Server{
		from:     from,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		sender:   defaultSMTPSender{},
	}
}

func NewServerWithSender(from, smtpHost, smtpPort string, sender SMTPSender) *Server {
	return &Server{
		from:     from,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		sender:   sender,
	}
}

func (s *Server) SendEmail(_ context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	if req.GetTo() == "" {
		return nil, status.Error(codes.InvalidArgument, "to is required")
	}
	if req.GetSubject() == "" {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}
	if req.GetTextBody() == "" && req.GetHtmlBody() == "" {
		return nil, status.Error(codes.InvalidArgument, "at least one of text_body or html_body is required")
	}

	if s.smtpHost == "" {
		slog.Info("notification.SendEmail stub (no SMTP_HOST configured)",
			"to", req.GetTo(), "subject", req.GetSubject())
		return &pb.SendEmailResponse{}, nil
	}

	msg := s.buildMessage(req)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	if err := s.sender.SendMail(addr, nil, s.from, []string{req.GetTo()}, msg); err != nil {
		return nil, status.Errorf(codes.Internal, "send email: %v", err)
	}

	return &pb.SendEmailResponse{}, nil
}

func (s *Server) buildMessage(req *pb.SendEmailRequest) []byte {
	boundary := fmt.Sprintf("mixed-%d", time.Now().UnixNano())

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: KISS Colab <%s>\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", req.GetTo()))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", req.GetSubject()))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
	msg.WriteString("\r\n")

	if req.GetTextBody() != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		msg.WriteString(req.GetTextBody() + "\r\n")
	}

	if req.GetHtmlBody() != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		msg.WriteString(req.GetHtmlBody() + "\r\n")
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return msg.Bytes()
}
