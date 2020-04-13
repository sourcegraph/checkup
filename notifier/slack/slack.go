package slack

import (
	"encoding/json"
	"fmt"
	"strings"

	slack "github.com/ashwanthkumar/slack-go-webhook"

	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "slack"

// Notifier consist of all the sub components required to use Slack API
type Notifier struct {
	Username string `json:"username"`
	Channel  string `json:"channel"`
	Webhook  string `json:"webhook"`
}

// New creates a new Notifier instance based on json config
func New(config json.RawMessage) (Notifier, error) {
	var notifier Notifier
	err := json.Unmarshal(config, &notifier)
	return notifier, err
}

// Type returns the notifier package name
func (Notifier) Type() string {
	return Type
}

// Notify implements notifier interface
func (s Notifier) Notify(results []types.Result) error {
	errs := make(types.Errors, 0)
	for _, result := range results {
		if !result.Healthy {
			if err := s.Send(result); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Send request via Slack API to create incident
func (s Notifier) Send(result types.Result) error {
	color := "danger"
	attach := slack.Attachment{}
	attach.AddField(slack.Field{Title: result.Title, Value: result.Endpoint})
	attach.AddField(slack.Field{Title: "Status", Value: strings.ToUpper(fmt.Sprint(result.Status()))})
	attach.Color = &color
	payload := slack.Payload{
		Text:        result.Title,
		Username:    s.Username,
		Channel:     s.Channel,
		Attachments: []slack.Attachment{attach},
	}

	return types.Errors(slack.Send(s.Webhook, "", payload))
}
