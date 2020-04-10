package checkup

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sourcegraph/checkup/notifier/slack"
)

func notifierDecode(typeName string, config json.RawMessage) (Notifier, error) {
	switch typeName {
	case "slack":
		return slack.New(config)
	default:
		return nil, errors.New(strings.Replace(errUnknownNotifierType, "%T", typeName, -1))
	}
}

func notifierType(ch interface{}) (string, error) {
	var typeName string
	switch ch.(type) {
	case slack.Notifier, *slack.Notifier:
		typeName = "slack"
	default:
		return "", fmt.Errorf(errUnknownNotifierType, ch)
	}
	return typeName, nil
}
