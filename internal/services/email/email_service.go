package email

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"text/template"

	"go-fiber-core/internal/dtos/config"

	blackfriday "github.com/russross/blackfriday/v2"
	gomail "gopkg.in/gomail.v2"
)

// --- INTERFACES ---

// EmailSender define el contrato para el servicio de envío de correos base.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlContent string) error
}

// TemplateSender define el contrato para un servicio que envía emails usando plantillas.
type TemplateSender interface {
	SendFromTemplate(ctx context.Context, to, subject, templateName string, data any) error
}

// --- IMPLEMENTACIÓN DEL SERVICIO DE ENVÍO BASE (GOMAIL) ---

type gomailService struct {
	from   string
	dialer *gomail.Dialer
}

// NewGomailService crea la instancia que se conectará al servidor SMTP real.
func NewGomailService(cfg config.EmailConfig) EmailSender {
	dialer := gomail.NewDialer(
		cfg.SmtpHost,
		cfg.SmtpPort,
		cfg.SmtpUsername,
		cfg.SmtpPassword,
	)
	return &gomailService{
		from:   cfg.SmtpFrom,
		dialer: dialer,
	}
}

func (s *gomailService) Send(ctx context.Context, to, subject, htmlContent string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", s.from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlContent)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.dialer.DialAndSend(msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("error enviando correo con gomail: %w", err)
		}
		return nil
	}
}

// --- IMPLEMENTACIÓN DEL SERVICIO DE LOGGING (PARA DESARROLLO) ---

type logSender struct {
	from string
}

// NewLogSender crea la instancia que imprimirá los correos en la consola.
func NewLogSender(cfg config.EmailConfig) EmailSender {
	log.Println("📬 Usando el servicio de email de LOGGING. Los correos se imprimirán en la consola y no se enviarán.")
	return &logSender{from: cfg.SmtpFrom}
}

func (s *logSender) Send(ctx context.Context, to, subject, htmlContent string) error {
	log.Println("--- 📧 Nuevo Correo (Simulación) 📧 ---")
	log.Printf("De: %s", s.from)
	log.Printf("Para: %s", to)
	log.Printf("Asunto: %s", subject)
	log.Println("--- Contenido ---")
	log.Println(htmlContent)
	log.Println("------------------------------------")
	return nil
}

// --- IMPLEMENTACIÓN DEL SERVICIO DE PLANTILLAS ---

type templateSender struct {
	emailService EmailSender // Ahora SÍ puede ver la interfaz EmailSender
	templates    *template.Template
}

// NewTemplateSender crea el servicio que maneja las plantillas.
func NewTemplateSender(emailService EmailSender, templatesDir string) (TemplateSender, error) {
	templates, err := template.ParseGlob(filepath.Join(templatesDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("error al parsear el directorio de plantillas: %w", err)
	}

	return &templateSender{
		emailService: emailService,
		templates:    templates,
	}, nil
}

func (s *templateSender) SendFromTemplate(ctx context.Context, to, subject, templateName string, data any) error {
	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return fmt.Errorf("error ejecutando la plantilla '%s': %w", templateName, err)
	}

	htmlContent := blackfriday.Run(body.Bytes())
	return s.emailService.Send(ctx, to, subject, string(htmlContent))
}
