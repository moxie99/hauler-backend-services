package api

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/resend/resend-go/v2"
)

// EmailService handles sending emails using Resend
type EmailService struct {
	client    *resend.Client
	fromEmail string
}

// SendVerificationCode sends a verification code to the specified email address using Resend
func (es *EmailService) SendVerificationCode(email, code string) error {
	// Email template
	subject := "Verify your email address"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #4F46E5; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
				.code { font-size: 32px; font-weight: bold; color: #4F46E5; text-align: center; padding: 20px; background-color: white; border-radius: 5px; margin: 20px 0; letter-spacing: 5px; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Email Verification</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>Thank you for registering as a driver. Please use the verification code below to verify your email address:</p>
					<div class="code">%s</div>
					<p>This code will expire in 15 minutes.</p>
					<p>If you didn't request this code, please ignore this email.</p>
				</div>
				<div class="footer">
					<p>© Hauler Services. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, code)

	textBody := fmt.Sprintf(`
Email Verification

Hello,

Thank you for registering as a driver. Please use the verification code below to verify your email address:

%s

This code will expire in 15 minutes.

If you didn't request this code, please ignore this email.

© Hauler Services. All rights reserved.
	`, code)

	// Send email via Resend
	params := &resend.SendEmailRequest{
		From:    es.fromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := es.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send verification email via Resend: %v", err)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	log.Printf("Verification email sent successfully via Resend. Email ID: %s", sent.Id)
	return nil
}

// SendPasswordResetCode sends a password reset verification code to the specified email address
func (es *EmailService) SendPasswordResetCode(email, code string) error {
	subject := "Reset your password"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #DC2626; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
				.code { font-size: 32px; font-weight: bold; color: #DC2626; text-align: center; padding: 20px; background-color: white; border-radius: 5px; margin: 20px 0; letter-spacing: 5px; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
				.warning { background-color: #FEF2F2; border-left: 4px solid #DC2626; padding: 15px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Password Reset</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>We received a request to reset your password. Please use the verification code below to proceed:</p>
					<div class="code">%s</div>
					<p>This code will expire in 15 minutes.</p>
					<div class="warning">
						<p><strong>Security Notice:</strong> If you didn't request this password reset, please ignore this email. Your account remains secure.</p>
					</div>
				</div>
				<div class="footer">
					<p>© Hauler Services. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, code)

	textBody := fmt.Sprintf(`
Password Reset

Hello,

We received a request to reset your password. Please use the verification code below to proceed:

%s

This code will expire in 15 minutes.

Security Notice: If you didn't request this password reset, please ignore this email. Your account remains secure.

© Hauler Services. All rights reserved.
	`, code)

	params := &resend.SendEmailRequest{
		From:    es.fromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := es.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send password reset email via Resend: %v", err)
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	log.Printf("Password reset email sent successfully via Resend. Email ID: %s", sent.Id)
	return nil
}

// SendLoginOTP sends a login verification OTP to the specified email address
func (es *EmailService) SendLoginOTP(email, code string) error {
	subject := "Login Verification Code"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #1E40AF; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
				.code { font-size: 32px; font-weight: bold; color: #1E40AF; text-align: center; padding: 20px; background-color: white; border-radius: 5px; margin: 20px 0; letter-spacing: 5px; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
				.warning { background-color: #FEF2F2; border-left: 4px solid #DC2626; padding: 15px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Login Verification</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>A login attempt was made to your account. Please use the verification code below to complete your login:</p>
					<div class="code">%s</div>
					<p>This code will expire in 15 minutes.</p>
					<div class="warning">
						<p><strong>Security Notice:</strong> If you didn't attempt to log in, please change your password immediately and contact support.</p>
					</div>
				</div>
				<div class="footer">
					<p>&copy; Hauler Services. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, code)

	textBody := fmt.Sprintf(`
Login Verification

Hello,

A login attempt was made to your account. Please use the verification code below to complete your login:

%s

This code will expire in 15 minutes.

Security Notice: If you didn't attempt to log in, please change your password immediately and contact support.

Hauler Services. All rights reserved.
	`, code)

	params := &resend.SendEmailRequest{
		From:    es.fromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := es.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send login OTP email via Resend: %v", err)
		return fmt.Errorf("failed to send login OTP email: %w", err)
	}

	log.Printf("Login OTP email sent successfully via Resend. Email ID: %s", sent.Id)
	return nil
}

// SendChangePasswordOTP sends a change password verification OTP to the specified email address
func (es *EmailService) SendChangePasswordOTP(email, code string) error {
	subject := "Change Password Verification Code"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #F59E0B; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
				.code { font-size: 32px; font-weight: bold; color: #F59E0B; text-align: center; padding: 20px; background-color: white; border-radius: 5px; margin: 20px 0; letter-spacing: 5px; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
				.warning { background-color: #FFFBEB; border-left: 4px solid #F59E0B; padding: 15px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Change Password</h1>
				</div>
				<div class="content">
					<p>Hello,</p>
					<p>A request was made to change your password. Please use the verification code below to proceed:</p>
					<div class="code">%s</div>
					<p>This code will expire in 15 minutes.</p>
					<div class="warning">
						<p><strong>Security Notice:</strong> If you didn't request this change, please secure your account immediately and contact support.</p>
					</div>
				</div>
				<div class="footer">
					<p>&copy; Hauler Services. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, code)

	textBody := fmt.Sprintf(`
Change Password Verification

Hello,

A request was made to change your password. Please use the verification code below to proceed:

%s

This code will expire in 15 minutes.

Security Notice: If you didn't request this change, please secure your account immediately and contact support.

Hauler Services. All rights reserved.
	`, code)

	params := &resend.SendEmailRequest{
		From:    es.fromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := es.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send change password OTP email via Resend: %v", err)
		return fmt.Errorf("failed to send change password OTP email: %w", err)
	}

	log.Printf("Change password OTP email sent successfully via Resend. Email ID: %s", sent.Id)
	return nil
}

// SendAdminWelcomeEmail sends a welcome email to newly created admin with login credentials
func (es *EmailService) SendAdminWelcomeEmail(email, password, firstName string) error {
	subject := "Welcome to Hauler Admin Panel"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #10B981; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
				.credentials { background-color: white; padding: 20px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #10B981; }
				.credential-item { margin: 10px 0; }
				.credential-label { font-weight: bold; color: #10B981; }
				.credential-value { font-family: monospace; background-color: #f3f4f6; padding: 5px 10px; border-radius: 3px; display: inline-block; }
				.footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
				.warning { background-color: #FEF2F2; border-left: 4px solid #DC2626; padding: 15px; margin: 20px 0; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Welcome to Hauler Admin Panel</h1>
				</div>
				<div class="content">
					<p>Hello %s,</p>
					<p>Your admin account has been successfully created. Below are your login credentials:</p>
					<div class="credentials">
						<div class="credential-item">
							<span class="credential-label">Email:</span>
							<span class="credential-value">%s</span>
						</div>
						<div class="credential-item">
							<span class="credential-label">Temporary Password:</span>
							<span class="credential-value">%s</span>
						</div>
					</div>
					<div class="warning">
						<p><strong>Important Security Notice:</strong></p>
						<ul>
							<li>You will be required to change your password on first login</li>
							<li>Do not share your credentials with anyone</li>
							<li>Keep your password secure and confidential</li>
						</ul>
					</div>
					<p>Please login to the admin panel and change your password immediately.</p>
				</div>
				<div class="footer">
					<p>&copy; Hauler Services. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, firstName, email, password)

	textBody := fmt.Sprintf(`
Welcome to Hauler Admin Panel

Hello %s,

Your admin account has been successfully created. Below are your login credentials:

Email: %s
Temporary Password: %s

Important Security Notice:
- You will be required to change your password on first login
- Do not share your credentials with anyone
- Keep your password secure and confidential

Please login to the admin panel and change your password immediately.

Hauler Services. All rights reserved.
	`, firstName, email, password)

	params := &resend.SendEmailRequest{
		From:    es.fromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	sent, err := es.client.Emails.Send(params)
	if err != nil {
		log.Printf("Failed to send admin welcome email via Resend: %v", err)
		return fmt.Errorf("failed to send admin welcome email: %w", err)
	}

	log.Printf("Admin welcome email sent successfully via Resend. Email ID: %s", sent.Id)
	return nil
}

// GenerateVerificationCode generates a cryptographically secure random 6-digit verification code
func GenerateVerificationCode() string {
	// Generate a random number between 0 and 999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to a simple method if crypto/rand fails (shouldn't happen)
		log.Printf("Warning: crypto/rand failed, using fallback: %v", err)
		return fmt.Sprintf("%06d", 123456) // This should never happen in practice
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// NewEmailService creates a new email service instance with Resend client
func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Fatal("RESEND_API_KEY environment variable is required")
	}

	fromEmail := os.Getenv("RESEND_FROM_EMAIL")
	if fromEmail == "" {
		// Default to a generic email - should be configured in production
		fromEmail = "onboarding@resend.dev"
		log.Printf("Warning: RESEND_FROM_EMAIL not set, using default: %s", fromEmail)
	}

	client := resend.NewClient(apiKey)

	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
	}
}
