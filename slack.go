package checkup

import (
	"fmt"
	"log"
	"strings"

	slack "github.com/ashwanthkumar/slack-go-webhook"
)
var notifications = make(map[string]int)
// Slack consist of all the sub components required to use Slack API
type Slack struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Channel  string `json:"channel"`
	Webhook  string `json:"webhook"`
}

// Notify implements notifier interface
func (s Slack) Notify(results []Result) error {
	for _, result := range results {
		if !result.Healthy {
			notifications[result.Title] = 1
			s.Send(result, "danger")
		} else if notifications[result.Title] == 1 {
			notifications[result.Title] = 0
			s.Send(result, "good")
		}
	}
	return nil
}

func (s Slack) Send(result Result, color string) error {
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
