package services

import (
	"fmt"
	"log"

	"github.com/fitreg/api/config"
	resend "github.com/resendlabs/resend-go"
)

type EmailService interface {
	SendCoachInviteExisting(coachName, toEmail, appURL string) error
	SendCoachInviteNew(coachName, toEmail, token, appURL string) error
}

type resendEmailService struct {
	client    *resend.Client
	emailFrom string
}

type noopEmailService struct{}

func NewEmailService(cfg *config.Config) EmailService {
	if cfg.ResendAPIKey == "" {
		log.Println("WARN: RESEND_API_KEY not set, email sending disabled")
		return &noopEmailService{}
	}
	return &resendEmailService{
		client:    resend.NewClient(cfg.ResendAPIKey),
		emailFrom: cfg.EmailFrom,
	}
}

func (s *resendEmailService) SendCoachInviteExisting(coachName, toEmail, appURL string) error {
	params := &resend.SendEmailRequest{
		From:    s.emailFrom,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s te invitó a entrenar en FitReg", coachName),
		Html: fmt.Sprintf(`<p>Hola,</p>
<p><strong>%s</strong> te envió una invitación para ser tu coach en FitReg.</p>
<p><a href="%s/invitations">Ver invitación →</a></p>
<p style="color:#999;font-size:12px">Si no esperabas este mensaje, podés ignorarlo.</p>`,
			coachName, appURL),
	}
	_, err := s.client.Emails.Send(params)
	return err
}

func (s *resendEmailService) SendCoachInviteNew(coachName, toEmail, token, appURL string) error {
	params := &resend.SendEmailRequest{
		From:    s.emailFrom,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s te invitó a unirte a FitReg", coachName),
		Html: fmt.Sprintf(`<p>Hola,</p>
<p><strong>%s</strong> te invitó a FitReg para ser tu coach de entrenamiento.</p>
<p>FitReg es la plataforma donde tu coach planifica tus semanas, vos cargás tus resultados y los dos ven el progreso.</p>
<p><a href="%s/join?token=%s">Unirme →</a></p>
<p style="color:#999;font-size:12px">Si no conocés a este coach, podés ignorar este mensaje.</p>`,
			coachName, appURL, token),
	}
	_, err := s.client.Emails.Send(params)
	return err
}

func (s *noopEmailService) SendCoachInviteExisting(coachName, toEmail, appURL string) error {
	log.Printf("INFO [email disabled] would send coach-invite-existing to %s", toEmail)
	return nil
}

func (s *noopEmailService) SendCoachInviteNew(coachName, toEmail, token, appURL string) error {
	log.Printf("INFO [email disabled] would send coach-invite-new to %s token=%s", toEmail, token)
	return nil
}
