package checkup

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/checkup/notifier/mail"
	"github.com/sourcegraph/checkup/notifier/mailgun"
	"github.com/sourcegraph/checkup/notifier/pushover"
	"github.com/sourcegraph/checkup/notifier/slack"
)

func notifierDecode(typeName string, config json.RawMessage) (Notifier, error) {
	switch typeName {
	case mail.Type:
		return mail.New(config)
	case slack.Type:
		return slack.New(config)
	case mailgun.Type:
		return mailgun.New(config)
	case pushover.Type:
		return pushover.New(config)
	default:
		return nil, fmt.Errorf(errUnknownNotifierType, typeName)
	}
}
