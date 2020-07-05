package checkup

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/checkup/exporter/appinsights"
)

func exporterDecode(typeName string, config json.RawMessage) (Exporter, error) {
	switch typeName {
	case appinsights.Type:
		return appinsights.New(config)
	default:
		return nil, fmt.Errorf(errUnknownExporterType, typeName)
	}
}
