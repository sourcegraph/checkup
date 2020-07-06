package appinsights

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "appinsights"

// Storage will send results to Azure Application Insights
type Storage struct {
	// InstrumentationKey is a GUID that identifies an app insights instance
	InstrumentationKey string `json:"instrumentation_key"`

	// TestName identifies the test name sent
	// in Application Insights trackAvailability() events
	TestName string `json:"test_name,omitempty"`

	// TestLocation identifies the test location sent
	// in Application Insights trackAvailability() events
	TestLocation string `json:"test_location,omitempty"`

	// Tags will be applied to all telemetry items and
	// visible in the customProperties field when viewing the
	// submitted data
	Tags map[string]string `json:"tags,omitempty"`

	// TelemetryClient is the appinsights.Client with which to
	// send Application Insights trackAvailability() events
	// Automatically created if InstrumentationKey is set.
	TelemetryClient appinsights.TelemetryClient
}

// New creates a new Storage instance based on json config
func New(config json.RawMessage) (Storage, error) {
	var storage Storage
	err := json.Unmarshal(config, &storage)
	if storage.TestLocation == "" {
		storage.TestLocation = "Checkup Monitor"
	}
	storage.TelemetryClient = appinsights.NewTelemetryClient(storage.InstrumentationKey)
	for k, v := range storage.Tags {
		storage.TelemetryClient.Context().CommonProperties[k] = v
	}
	return storage, err
}

// Type returns the logger package name
func (Storage) Type() string {
	return Type
}

// Store takes a list of Checker results and sends them to the configured
// Application Insights instance.
func (c Storage) Store(results []types.Result) error {
	for _, result := range results {
		c.send(result)
	}
	return c.close()
}

// Close will submit all queued telemetry to the configured Application Insights
// service.
// Ref: https://github.com/microsoft/ApplicationInsights-Go#shutdown
func (c Storage) close() error {
	select {
	case <-c.TelemetryClient.Channel().Close(10 * time.Second):
		// Ten second timeout for retries.
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("Failed to submit telemetry after retries")
	}
}

// Send a result
// Multiple test result measurements will be added to the telemetry item's
// common properties
func (c Storage) send(conclude types.Result) {
	total := int64(0)
	message := "Passed"
	if conclude.Degraded {
		message = "Degraded"
	} else if conclude.Down {
		message = "Down"
	}
	message = fmt.Sprintf("%s ", conclude.Notice)
	mean := time.Duration(total / int64(len(conclude.Times)))
	availability := appinsights.NewAvailabilityTelemetry(conclude.Title, mean, conclude.Healthy)
	availability.RunLocation = c.TestLocation
	availability.Message = message
	availability.Id = uuid.New().String()
	availability.GetProperties()["ThresholdRTT"] = conclude.ThresholdRTT.String()
	for i := 0; i < len(conclude.Times); i++ {
		k := strconv.Itoa(i)
		total += int64(conclude.Times[i].RTT)
		availability.GetMeasurements()[k] = float64(conclude.Times[i].RTT)
	}

	// Submit the telemetry
	c.TelemetryClient.Track(availability)
	return
}
