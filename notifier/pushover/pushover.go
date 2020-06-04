package pushover

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gregdel/pushover"
	"github.com/sourcegraph/checkup/types"
)

const Type = "pushover"

type Notifier struct {
	Token     string `json:"token"`
	Recipient string `json:"recipient"`
	Subject   string `json:"subject,omitempty"`
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

func (p Notifier) Notify(results []types.Result) error {
	issues := []types.Result{}
	for _, result := range results {
		if !result.Healthy {
			issues = append(issues, result)
		}
	}

	if len(issues) == 0 {
		return nil
	}

	app := pushover.New(p.Token)
	recipient := pushover.NewRecipient(p.Recipient)
	msg := pushover.NewMessageWithTitle(renderMessage(issues), p.Subject)

	_, err := app.SendMessage(msg, recipient)
	return err
}

func renderMessage(issues []types.Result) string {
	body := []string{"Checkup has detected the following issues:", "\n\n"}
	for _, issue := range issues {
		format := "%s - Status: %s"
		body = append(body, fmt.Sprintf(format, issue.Title, issue.Status()))
	}
	return strings.Join(body, "\n")
}
