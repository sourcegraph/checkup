package slack

import (
	"fmt"
	"log"
	"strings"
	"encoding/json"

	slack "github.com/ashwanthkumar/slack-go-webhook"

	"github.com/sourcegraph/checkup/types"
)

// Notifier consist of all the sub components required to use Slack API
type Notifier struct {
	Name     string `json:"name"`
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

// Notify implements notifier interface
func (s Notifier) Notify(results []types.Result) error {
	for _, result := range results {
		if !result.Healthy {
			s.Send(result)
		}
	}
	return nil
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

	err := slack.Send(s.Webhook, "", payload)
	if len(err) > 0 {
		log.Printf("ERROR: %s", err)
	}
	log.Printf("Create request for %s", result.Endpoint)
	return nil
}
