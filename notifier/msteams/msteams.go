package msteams

import (
	"encoding/json"
	"fmt"
	"strings"

  msteams "github.com/ykorzikowski/msteams-go-webhook"

	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "msteams"

// Notifier consist of all the sub components required to use MSTeams API
type Notifier struct {
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

// Send request via MSTeams API to create incident
func (s Notifier) Send(result types.Result) error {

  title := ""
  markdown := true

  section := msteams.Section {
    ActivityTitle: &result.Title,
    ActivitySubTitle: &title,
    ActivityImage: &title,
    Markdown: &markdown,
  }
  section.AddFact(msteams.Fact { Name: result.Title, Value: result.Endpoint }).AddFact(msteams.Fact { Name: "Status", Value: strings.ToUpper(fmt.Sprint(result.Status())) })

  payload := msteams.Payload {
    Type: "MessageCard",
    Context: "http://schema.org/extensions",
    ThemeColor: "ff0000",
    Summary: result.Title,
    Sections: []msteams.Section{section},
  }

	return types.Errors(msteams.Send(s.Webhook, "", payload))
}
