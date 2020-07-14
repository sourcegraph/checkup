package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sourcegraph/checkup/types"
)

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
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// Send request via Slack API to create incident
func (s Notifier) Send(result types.Result) error {
	status := strings.ToUpper(fmt.Sprint(result.Status()))

	attach := &Payload{}
	attach.Title = "Checkup"
	embed := &Embed{
		Color: 0xc21408,
	}
	embed.AddField(&Field{
		Name:   "Name",
		Value:  result.Title,
		Inline: true,
	})
	embed.AddField(&Field{
		Name:   "Status",
		Value:  fmt.Sprintf("**%s**", status),
		Inline: true,
	})
	embed.AddField(&Field{
		Name:   "Endpoint",
		Value:  result.Endpoint,
		Inline: true,
	})
	attach.AddEmbed(embed)
	attach.Avatar = "https://placekitten.com/400/400"

	requestBody, err := json.Marshal(attach)

	fmt.Println(string(requestBody))

	if err != nil {
		return fmt.Errorf("discord: error marshalling body: %w", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	// client
	client := &http.Client{}

	// request with timeout
	req, err := http.NewRequestWithContext(ctx, "POST", s.Webhook, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("discord: error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// response
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: error issuing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("discord: error reading response body", err)
		}
		bodyString := string(bodyBytes)
		log.Println("discord: response body:", bodyString)
		return fmt.Errorf("discord: expected status 200, got %d", resp.StatusCode)
	}
	return nil
}
