package utils

import (
	"chatapp/back/internal/config"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"time"
)

func GenerateEmailCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func SendEmail(to, subject, body string) error {
	c := config.GetConfig().Email

	auth := smtp.PlainAuth("", c.Username, c.Password, c.SmtpHost)

	msg := "From: " + c.Username + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body + "\r\n"

	tlsConfig := &tls.Config{InsecureSkipVerify: true, ServerName: c.SmtpHost}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.SmtpHost, c.SmtpPort), tlsConfig)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, c.SmtpHost)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(c.Username); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}

	return client.Quit()
}
