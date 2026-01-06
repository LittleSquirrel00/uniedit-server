package user

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"

	"go.uber.org/zap"
)

// SMTPConfig holds SMTP configuration.
type SMTPConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	FromAddress string
	FromName    string
	BaseURL     string // For verification links
}

// SMTPEmailSender sends emails via SMTP.
type SMTPEmailSender struct {
	config *SMTPConfig
	logger *zap.Logger
}

// NewSMTPEmailSender creates a new SMTP email sender.
func NewSMTPEmailSender(config *SMTPConfig, logger *zap.Logger) *SMTPEmailSender {
	return &SMTPEmailSender{
		config: config,
		logger: logger,
	}
}

// SendVerificationEmail sends a verification email.
func (s *SMTPEmailSender) SendVerificationEmail(ctx context.Context, email, name, token string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.config.BaseURL, token)

	subject := "Verify your email address"
	body, err := s.renderTemplate(verificationEmailTemplate, map[string]string{
		"Name":      name,
		"VerifyURL": verifyURL,
	})
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	return s.sendEmail(email, subject, body)
}

// SendPasswordResetEmail sends a password reset email.
func (s *SMTPEmailSender) SendPasswordResetEmail(ctx context.Context, email, name, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, token)

	subject := "Reset your password"
	body, err := s.renderTemplate(passwordResetEmailTemplate, map[string]string{
		"Name":     name,
		"ResetURL": resetURL,
	})
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	return s.sendEmail(email, subject, body)
}

func (s *SMTPEmailSender) sendEmail(to, subject, body string) error {
	from := s.config.FromAddress
	if s.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.User != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)
	}

	if err := smtp.SendMail(addr, auth, s.config.FromAddress, []string{to}, []byte(msg)); err != nil {
		s.logger.Error("failed to send email",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("send email: %w", err)
	}

	s.logger.Info("email sent", zap.String("to", to), zap.String("subject", subject))
	return nil
}

func (s *SMTPEmailSender) renderTemplate(tmpl string, data map[string]string) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

const verificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 6px; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to UniEdit!</h1>
        <p>Hi {{.Name}},</p>
        <p>Thanks for signing up. Please verify your email address by clicking the button below:</p>
        <p><a href="{{.VerifyURL}}" class="button">Verify Email</a></p>
        <p>Or copy and paste this link into your browser:</p>
        <p>{{.VerifyURL}}</p>
        <p>This link will expire in 24 hours.</p>
        <div class="footer">
            <p>If you didn't create an account, you can safely ignore this email.</p>
        </div>
    </div>
</body>
</html>
`

const passwordResetEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 6px; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Reset Your Password</h1>
        <p>Hi {{.Name}},</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>
        <p><a href="{{.ResetURL}}" class="button">Reset Password</a></p>
        <p>Or copy and paste this link into your browser:</p>
        <p>{{.ResetURL}}</p>
        <p>This link will expire in 1 hour.</p>
        <div class="footer">
            <p>If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
        </div>
    </div>
</body>
</html>
`

// NoOpEmailSender is a no-op email sender for testing/development.
type NoOpEmailSender struct {
	logger *zap.Logger
}

// NewNoOpEmailSender creates a no-op email sender.
func NewNoOpEmailSender(logger *zap.Logger) *NoOpEmailSender {
	return &NoOpEmailSender{logger: logger}
}

// SendVerificationEmail logs but doesn't send.
func (s *NoOpEmailSender) SendVerificationEmail(ctx context.Context, email, name, token string) error {
	s.logger.Info("verification email (no-op)",
		zap.String("email", email),
		zap.String("name", name),
		zap.String("token", token),
	)
	return nil
}

// SendPasswordResetEmail logs but doesn't send.
func (s *NoOpEmailSender) SendPasswordResetEmail(ctx context.Context, email, name, token string) error {
	s.logger.Info("password reset email (no-op)",
		zap.String("email", email),
		zap.String("name", name),
		zap.String("token", token),
	)
	return nil
}
