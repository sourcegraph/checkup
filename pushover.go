package checkup

import (
	"fmt"

	"github.com/gregdel/pushover"
)

type Pushover struct {
	Name     string `json:"name"`
	UserKey  string `json:"userKey"`
	AppToken string `json:"appToken"`
}

func (s Pushover) Notify(results []Result) error {
	for _, result := range results {
		if !result.Healthy {
			s.Send(result)
		}
	}
	return nil
}

func (s Pushover) Send(result Result) error {
	// Create a new pushover app with a token
	app := pushover.New(s.AppToken)
	// Create a new recipient
	recipient := pushover.NewRecipient(s.UserKey)
	// Create the message to send
	message := pushover.NewMessageWithTitle(result.Endpoint+" Status: "+string(result.Status()), result.Title)

	// Send the message to the recipient
	_, err := app.SendMessage(message, recipient)
	return err
}

func (t *Pushover) String() string {
	return fmt.Sprintf("Pushover: %s with appToken %s", t.UserKey, t.AppToken)
}
