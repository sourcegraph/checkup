package mail

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/gomail.v2"

	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "mail"

// Notifier consist of all the sub components required to send E-mail notifications
type Notifier struct {
	// From contains the e-mail address notifications are sent from
	From string `json:"from"`

	// To contains a list of e-mail address destinations
	To []string `json:"to"`

	// Subject contains customizable subject line
	Subject string `json:"subject,omitempty"`

	// SMTP contains all relevant mail server settings
	SMTP struct {
		Server   string `json:"server"`
		Port     int    `json:"port,omitempty"`
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"smtp"`
}

// New creates a new Notifier instance based on json config
func New(config json.RawMessage) (Notifier, error) {
	var notifier Notifier
	err := json.Unmarshal(config, &notifier)
	// Fall back to port 25 if not defined
	if notifier.SMTP.Port == 0 {
		notifier.SMTP.Port = 25
	}
	if strings.TrimSpace(notifier.Subject) == "" {
		notifier.Subject = "Checkup: Service Unavailable"
	}
	return notifier, err
}

// Type returns the notifier package name
func (Notifier) Type() string {
	return Type
}

// Notify implements notifier interface
func (m Notifier) Notify(results []types.Result) error {
	issues := []types.Result{}
	for _, result := range results {
		if !result.Healthy {
			issues = append(issues, result)
		}
	}

	if len(issues) == 0 {
		return nil
	}

	message := gomail.NewMessage()
	message.SetHeader("From", m.From)
	message.SetHeader("To", m.To...)
	message.SetHeader("Subject", m.Subject)
	message.SetBody("text/html", renderMessage(issues))

	dialer := gomail.NewDialer(m.SMTP.Server, m.SMTP.Port, m.SMTP.Username, m.SMTP.Password)
	return dialer.DialAndSend(message)
}

func renderMessage(issues []types.Result) string {
	body := []string{"<b>Checkup has detected the following issues:</b>", "<br/><br/>", "<ul>"}
	for _, issue := range issues {
		format := "<li>%s - Status <b>%s</b></li>"
		body = append(body, fmt.Sprintf(format, issue.Title, issue.Status()))
	}
	body = append(body, "</ul>")
	return strings.Join(body, "\n")
}
