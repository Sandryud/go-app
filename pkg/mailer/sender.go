package mailer

import "context"

// EmailSender описывает контракт для отправки кода подтверждения email.
type EmailSender interface {
	SendEmailVerificationCode(ctx context.Context, email, code string) error
}
