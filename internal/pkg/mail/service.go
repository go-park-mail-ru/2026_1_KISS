package mail

import (
	"fmt"
	"log"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) SendVerification(email, token string) {
	link := fmt.Sprintf("http://localhost:8080/api/v1/auth/confirm?token=%s", token)

	log.Println("=== EMAIL VERIFICATION ===")
	log.Println("to:", email)
	log.Println("link:", link)
}
