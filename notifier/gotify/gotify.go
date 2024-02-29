package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sourcegraph/checkup/types"
)

const Type = "gotify"

type Notifier struct {
	Token   string `json:"token"`
	Webhook string `json:"webhook"`
}

func New(config json.RawMessage) (Notifier, error) {
	var notifier Notifier
	err := json.Unmarshal(config, &notifier)
	return notifier, err
}

func (Notifier) Type() string {
	return Type
}

func (g Notifier) Notify(results []types.Result) error {
	var issues []types.Result
	for _, result := range results {
		if !result.Healthy {
			issues = append(issues, result)
		}
	}
	if len(issues) == 0 {
		return nil
	}

	return g.Send(issues)
}

func (g Notifier) Send(results []types.Result) error {
	message := renderMessage(results)
	requestBody, err := json.Marshal(struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		Priority int    `json:"priority"`
	}{"Checkup", message, 5})

	if err != nil {
		return fmt.Errorf("gotify: error marshalling body: %w", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client := &http.Client{}
	webhookUrl, err := url.Parse(g.Webhook)

	if err != nil {
		return fmt.Errorf("gotify: error in parse webhook: %w", err)
	}

	query := webhookUrl.Query()
	query.Add("token", g.Token)
	webhookUrl.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST",
		webhookUrl.String(),
		bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// response
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gotify: error issuing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > http.StatusBadRequest {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("gotify: error reading response body", err)
		}
		bodyString := string(bodyBytes)
		log.Println("gotify: response body:", bodyString)
		return fmt.Errorf("gotify: expected status 200, got %d", resp.StatusCode)
	}
	return err
}

func renderMessage(issues []types.Result) string {
	body := []string{"Checkup has detected the following issues:\n"}
	for _, issue := range issues {
		status := strings.ToUpper(fmt.Sprint(issue.Status()))
		text := fmt.Sprintf("%s (%s) - Status: %s - Attempts: %d",
			issue.Title, issue.Endpoint, status, issue.Times.Len())
		body = append(body, text)
	}
	return strings.Join(body, "\n")

}
