package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"auto-hub/web/internal/config"
)

// EmailService sends emails via the internal mail service.
type EmailService struct {
	cfg *config.Config
}

// NewEmailService creates a new EmailService.
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

// SendVerificationEmail sends a verification code email.
func (s *EmailService) SendVerificationEmail(toEmail, code string) error {
	if s.cfg.InternalAPIToken == "" {
		return fmt.Errorf("internal api token not configured")
	}

	payload := map[string]interface{}{
		"from":    "noreply@auto-hub.ca",
		"to":      toEmail,
		"subject": "Auto-Hub Email Verification",
		"text":    fmt.Sprintf("Your verification code is: %s", code),
		"html":    fmt.Sprintf("<p>Your verification code is: <strong>%s</strong></p>", code),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := s.cfg.MailServiceURL + "/internal/send"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.InternalAPIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mail service returned %d", resp.StatusCode)
	}
	return nil
}
