package grpc

import (
	"context"
	"errors"
	"net/smtp"
	"testing"

	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockSMTPSender struct {
	err error
}

func (m *mockSMTPSender) SendMail(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
	return m.err
}

func TestSendEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *pb.SendEmailRequest
		smtpErr  error
		wantCode codes.Code
	}{
		{
			name: "success",
			req: &pb.SendEmailRequest{
				To:       "user@example.com",
				Subject:  "Test",
				TextBody: "Hello",
				HtmlBody: "<p>Hello</p>",
			},
			smtpErr:  nil,
			wantCode: codes.OK,
		},
		{
			name: "smtp failure",
			req: &pb.SendEmailRequest{
				To:       "user@example.com",
				Subject:  "Test",
				TextBody: "Hello",
			},
			smtpErr:  errors.New("connection refused"),
			wantCode: codes.Internal,
		},
		{
			name: "empty to",
			req: &pb.SendEmailRequest{
				To:       "",
				Subject:  "Test",
				TextBody: "Hello",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty subject",
			req: &pb.SendEmailRequest{
				To:       "user@example.com",
				Subject:  "",
				TextBody: "Hello",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty body",
			req: &pb.SendEmailRequest{
				To:      "user@example.com",
				Subject: "Test",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender := &mockSMTPSender{err: tt.smtpErr}
			srv := NewServerWithSender("noreply@test.com", "localhost", "25", sender)

			resp, err := srv.SendEmail(context.Background(), tt.req)

			if tt.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resp == nil {
					t.Fatal("expected non-nil response")
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("expected code %v, got %v", tt.wantCode, st.Code())
			}
		})
	}
}
