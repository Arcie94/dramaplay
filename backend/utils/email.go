package utils

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendEmail(to []string, subject string, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpSender := os.Getenv("SMTP_SENDER")

	if smtpHost == "" || smtpUser == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	port, _ := strconv.Atoi(smtpPort)

	m := gomail.NewMessage()
	m.SetHeader("From", smtpSender)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(smtpHost, port, smtpUser, smtpPass)

	// For local dev with simple SMTP or if cert issues arise
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
