package appinsights

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "appinsights"

// Exporter implements a Exporter by sending Checker output to an external telemetry tool
type Exporter struct {
	// InstrumentationKey is a GUID used to send trackAvailability()
	// telemetry to Application Insights
	InstrumentationKey string `json:"instrumentation_key"`

	// TestName identifies the test name sent
	// in Application Insights trackAvailability() events
	TestName string `json:"test_name,omitempty"`

	// TestLocation identifies the test location sent
	// in Application Insights trackAvailability() events
	TestLocation string `json:"test_location,omitempty"`

	// Tags will be applied to all telemetry items
	Tags map[string]string `json:"tags,omitempty"`

	// TelemetryClient is the appinsights.Client with which to
	// send Application Insights trackAvailability() events
	// Automatically created if InstrumentationKey is set.
	TelemetryClient appinsights.TelemetryClient
}

// New creates a new Exporter instance based on json config
func New(config json.RawMessage) (Exporter, error) {
	var exporter Exporter
	err := json.Unmarshal(config, &exporter)
	if exporter.TestLocation == "" {
		exporter.TestLocation = "Checkup Exporter"
	}
	exporter.TelemetryClient = appinsights.NewTelemetryClient(exporter.InstrumentationKey)
	return exporter, err
}

// Type returns the logger package name
func (Exporter) Type() string {
	return Type
}

// Export takes a list of Checker results and sends them to the configured
// Application Insights instance.
func (c Exporter) Export(results []types.Result) error {
	for _, result := range results {
		c.Send(result)
	}
	return c.Close()
}

// Close will submit all queued telemetry to the configured Application Insights
// service.
// Ref: https://github.com/microsoft/ApplicationInsights-Go#shutdown
func (c Exporter) Close() error {
	select {
	case <-c.TelemetryClient.Channel().Close(10 * time.Second):
		// Ten second timeout for retries.
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("Failed to submit telemetry after retries")
	}
}

// Send sends a result to the exporter
func (c Exporter) Send(conclude types.Result) {
	attempts := len(conclude.Times)
	rtts := make([]string, attempts)
	message := "Passed"
	if conclude.Degraded || conclude.Down {
		for i := 0; i < attempts; i++ {
			rtts[i] = conclude.Times[i].RTT.String()
		}
		message = fmt.Sprintf("Number of attempts = %d (%s)", len(conclude.Times), strings.Join(rtts, " "))
		if conclude.Notice != "" {
			message = fmt.Sprintf("%s - ", message)
		}
	}

	availability := appinsights.NewAvailabilityTelemetry(conclude.Title, conclude.Stats.Mean, conclude.Healthy)
	availability.RunLocation = c.TestLocation
	availability.Message = message
	availability.Id = uuid.New().String()
	for k, v := range c.Tags {
		c.TelemetryClient.Context().CommonProperties[k] = v
	}

	// Submit the telemetry
	c.TelemetryClient.Track(availability)
	return
}
