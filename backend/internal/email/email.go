// backend/internal/email/email.go

// Package email fornece funcionalidades para envio de emails usando a biblioteca padrão do Go.
//
// Este pacote implementa um serviço de email para enviar mensagens transacionais como
// recuperação de senha, confirmação de cadastro, etc.
//
// O serviço usa a biblioteca net/smtp padrão do Go e suporta autenticação SMTP.

package email

import (
	"bytes"
	"fmt"
	"gosveltekit/internal/config"
	"html/template"
	"net/smtp"
)

// EmailServiceInterface defines the interface for email services
type EmailServiceInterface interface {
	SendPasswordResetEmail(to, token, username, displayName string) error
}

// EmailService é o serviço responsável pelo envio de emails
type EmailService struct {
	config *config.EmailConfig
}

// NewEmailService cria uma nova instância do serviço de email
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		config: &cfg.Email,
	}
}

// EmailData contém dados dinâmicos para templates de email
type EmailData struct {
	Username     string
	ResetLink    string
	DisplayName  string
	AppName      string
	SupportEmail string
}

// SendPasswordResetEmail envia um email de recuperação de senha com um link contendo o token
func (s *EmailService) SendPasswordResetEmail(to, token, username, displayName string) error {
	subject := "Recuperação de Senha"
	resetLink := s.config.ResetURL + token

	// Dados para o template de email
	data := EmailData{
		Username:     username,
		ResetLink:    resetLink,
		DisplayName:  displayName,
		AppName:      "GoSvelteKit",
		SupportEmail: s.config.FromEmail,
	}

	// HTML para o corpo do email
	htmlBody := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Recuperação de Senha</title>
		<style>
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 0; background-color: #f9f9f9; color: #333; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; }
			.header { background-color: #1e293b; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
			.content { background-color: white; padding: 20px; border-radius: 0 0 5px 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
			.button { display: inline-block; background-color: #1e293b; color: white; text-decoration: none; padding: 10px 20px; border-radius: 5px; margin: 20px 0; }
			.footer { margin-top: 20px; text-align: center; font-size: 12px; color: #666; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>Recuperação de Senha</h1>
			</div>
			<div class="content">
				<p>Olá {{.DisplayName}},</p>
				<p>Recebemos uma solicitação para redefinir a senha da sua conta.</p>
				<p>Se você não solicitou uma nova senha, ignore este email.</p>
				<p>Para redefinir sua senha, clique no botão abaixo:</p>
				<p style="text-align: center;">
					<a href="{{.ResetLink}}" class="button">Redefinir Senha</a>
				</p>
				<p>Ou copie e cole o seguinte link no seu navegador:</p>
				<p>{{.ResetLink}}</p>
				<p>Este link expirará em 1 hora por motivos de segurança.</p>
				<p>Atenciosamente,<br>Equipe {{.AppName}}</p>
			</div>
			<div class="footer">
				<p>Este é um email automático, por favor não responda.<br>
				Em caso de dúvidas, entre em contato com {{.SupportEmail}}</p>
			</div>
		</div>
	</body>
	</html>
	`

	// Criamos um template a partir do HTML
	t, err := template.New("reset_email").Parse(htmlBody)
	if err != nil {
		return fmt.Errorf("erro ao analisar template: %w", err)
	}

	// Aplicamos os dados ao template
	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("erro ao executar template: %w", err)
	}

	// Enviamos o email usando a função auxiliar
	return s.sendEmail(to, subject, body.String())
}

// sendEmail é uma função auxiliar que envia um email usando SMTP
func (s *EmailService) sendEmail(to, subject, htmlBody string) error {
	// Configurações de SMTP
	host := s.config.SMTPHost
	port := s.config.SMTPPort
	username := s.config.SMTPUsername
	password := s.config.SMTPPassword
	fromEmail := s.config.FromEmail
	fromName := s.config.FromName

	// Construir o cabeçalho do email
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Construir a mensagem com cabeçalhos e corpo
	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	// Autenticação SMTP
	auth := smtp.PlainAuth("", username, password, host)

	// Endereço do servidor SMTP
	addr := fmt.Sprintf("%s:%d", host, port)

	// Enviamos o email
	return smtp.SendMail(
		addr,
		auth,
		fromEmail,
		[]string{to},
		message.Bytes(),
	)
}
