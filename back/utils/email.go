// ============================================================
// 文件：back/utils/email.go
// 作用：通过 SMTP 协议发送邮件，用于验证码登录功能。
//
// 邮件发送流程：
//   1. 从配置读取 SMTP 服务器信息（地址、端口、发件人账号密码）
//   2. 用 TLS 加密连接到 SMTP 服务器（比普通 TCP 更安全，防止密码被截获）
//   3. 用 PLAIN 认证（账号+密码）登录邮件服务器
//   4. 设置发件人、收件人
//   5. 写入邮件正文（包括 From/To/Subject/Content-Type 头部）
//   6. 发送并断开连接
//
// TLS vs STARTTLS：
//   这里用 tls.Dial（直接 TLS 加密连接，端口通常 465）
//   区别于 STARTTLS（先建普通连接再升级为加密，端口通常 587）
//   Gmail 的 465 端口用的是直接 TLS，所以配置里 smtp_port = 465。
//
// InsecureSkipVerify = false：
//   不跳过证书验证，正式生产环境里应该保持 false，确保服务器真实可信。
// ============================================================
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

	tlsConfig := &tls.Config{InsecureSkipVerify: false, ServerName: c.SmtpHost}
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
