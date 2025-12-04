package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"workout-app/internal/config"
	"workout-app/pkg/logger"
)

// SMTPSender реализует отправку писем через стандартную библиотеку net/smtp.
// Используется для отправки кода подтверждения email.
type SMTPSender struct {
	cfg    *config.EmailConfig
	logger logger.Logger
}

// NewSMTPSender создаёт новый SMTP-отправитель на основе EmailConfig.
func NewSMTPSender(cfg *config.EmailConfig, logger logger.Logger) *SMTPSender {
	return &SMTPSender{
		cfg:    cfg,
		logger: logger,
	}
}

// SendEmailVerificationCode отправляет письмо с кодом подтверждения email.
// Используется как для подтверждения email при регистрации, так и для подтверждения изменения email.
func (s *SMTPSender) SendEmailVerificationCode(ctx context.Context, email, code string) error {
	subject := "Your verification code"
	body := fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in a few minutes.", code)

	msg := buildMessage(s.cfg.FromEmail, email, subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	// В данной реализации контекст используется только для логгирования;
	// net/smtp не поддерживает контекст из коробки.
	if err := smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{email}, []byte(msg)); err != nil {
		s.logger.Error("failed to send verification email", map[string]any{
			"email": email,
			"err":   err.Error(),
		})
		return err
	}

	s.logger.Info("verification email sent", map[string]any{
		"email": email,
	})
	return nil
}

func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}
