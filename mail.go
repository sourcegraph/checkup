package checkup

import (
	"fmt"
	"strings"
	"log"
	"gopkg.in/gomail.v2"
)

type MailNotifier struct {
	Emails     string `json:"emails"`
	SMTPServer string `json:"smtpServer"`
	SMTPPort   int    `json:"smtpPort"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// Notify implements notifier interface
func (m MailNotifier) Notify(results []Result) error {
	var servicesUnavailable []Result
	for _, result := range results {
		if !result.Healthy {
			servicesUnavailable = append(servicesUnavailable, result)
		}
	}
	if len(servicesUnavailable) == 0 {
		return nil
	}
	mail := gomail.NewMessage()
	mail.SetHeader("From", m.Username)
	mail.SetHeader("To", strings.Split(m.Emails, ",")...)
	mail.SetHeader("Subject", "Service Unavailable")
	var respBody = "<h1>Services with problems:</h1><ul>"
	for _, result := range servicesUnavailable {
		respBody += fmt.Sprintf("<li> %s </li>", result.Title)

	}
	respBody += "</ul>"

	mail.SetBody("text/html", respBody)

	d := gomail.NewDialer(m.SMTPServer, m.SMTPPort, m.Username, m.Password)
	err := d.DialAndSend(mail)
	if err != nil {
		log.Println("Error ocurred when sending mail")
		log.Println(err)
	}
	return nil
}
