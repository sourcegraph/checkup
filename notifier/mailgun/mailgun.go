package mailgun

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mailgun "github.com/mailgun/mailgun-go/v4"
	"github.com/sourcegraph/checkup/types"
)

const Type = "mailgun"

type Notifier struct {
	APIKey  string `json:"apikey"`
	Domain  string `json:"domain"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject,omitempty"`
}

func New(config json.RawMessage) (Notifier, error) {
	var notifier Notifier
	err := json.Unmarshal(config, &notifier)
	if strings.TrimSpace(notifier.Subject) == "" {
		notifier.Subject = "Checkup: Service Unavailable"
	}
	return notifier, err
}

func (Notifier) Type() string {
	return Type
}

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

	mg := mailgun.NewMailgun(m.Domain, m.APIKey)
	msg := mg.NewMessage(m.From, m.Subject, renderMessage(issues), m.To)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := mg.Send(ctx, msg)
	return err
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
