package checkup

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/checkup/notifier/mail"
	"github.com/sourcegraph/checkup/notifier/slack"
    "github.com/ykorzikowski/checkup/notifier/msteams"
)

func notifierDecode(typeName string, config json.RawMessage) (Notifier, error) {
	switch typeName {
	case mail.Type:
		return mail.New(config)
	case slack.Type:
		return slack.New(config)
    case msteams.Type:
        return msteams.New(config)
	default:
		return nil, fmt.Errorf(errUnknownNotifierType, typeName)
	}
}
