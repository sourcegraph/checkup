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

	// TestLocation identifies the test location sent
	// in Application Insights trackAvailability() events
	TestLocation string `json:"test_location,omitempty"`

	// Tags will be applied to all telemetry items and
	// visible in the customDimensions field when viewing the
	// submitted data
	Tags map[string]string `json:"tags,omitempty"`

	// MaxRetries specifies the number of retries before returning error
	// from close().  To enable retries, both RetryInterval and MaxRetries
	// must be greater than 0.  Default is 0 (disabled).
	MaxRetries time.Duration `json:"max_retries,omitempty"`

	// RetryInterval specifies the time between retries in seconds.  To enable retries,
	// both RetryInterval and MaxRetries must
	// be greater than 0.  Default is 0 (disabled).
	RetryInterval time.Duration `json:"retry_interval,omitempty"`

	// TelemetryClient is the appinsights.Client with which to
	// send Application Insights trackAvailability() events
	// Automatically created if InstrumentationKey is set.
	TelemetryClient appinsights.TelemetryClient

	// TelemetryConfig is used to modify the behavior of TelemetryClient
	TelemetryConfig *appinsights.TelemetryConfiguration
}

// New creates a new Storage instance based on json config
func New(config json.RawMessage) (Storage, error) {
	var storage Storage
	err := json.Unmarshal(config, &storage)
	storage.TelemetryConfig.InstrumentationKey = storage.InstrumentationKey

	if storage.TestLocation == "" {
		storage.TestLocation = "Checkup Monitor"
	}
	if storage.MaxRetries < 0 {
		err = fmt.Errorf("Invalid storage max_retries: %d", storage.MaxRetries)
	}
	if storage.RetryInterval < 0 {
		err = fmt.Errorf("Invalid storage retry_interval: %d", storage.RetryInterval)
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
	c.TelemetryClient = appinsights.NewTelemetryClientFromConfig(c.TelemetryConfig)
	for k, v := range c.Tags {
		c.TelemetryClient.Context().CommonProperties[k] = v
	}
	for _, result := range results {
		c.send(result)
	}
	return c.close()
}

// Close will submit all queued telemetry to the configured Application Insights
// service.  If either RetryInterval or MaxRetries are <= 0, Close() should proceed
// without a retry and a maximum timeout of 2 seconds.
// Ref: https://github.com/microsoft/ApplicationInsights-Go#shutdown
func (c Storage) close() error {
	if c.RetryInterval <= 0 || c.MaxRetries <= 0 {
		select {
		case <-c.TelemetryClient.Channel().Close():
			return nil
		case <-time.After(2 * time.Second):
			return fmt.Errorf("Failed to submit telemetry during close")
		}
	}
	select {
	case <-c.TelemetryClient.Channel().Close(c.RetryInterval * time.Second):
		return nil
	case <-time.After((c.MaxRetries + 1) * c.RetryInterval * time.Second):
		return fmt.Errorf("Failed to submit telemetry after retries")
	}
}

// Send a result
// Multiple test result measurements will be added to the telemetry item's
// customMeasurements field
func (c Storage) send(conclude types.Result) {
	total := int64(0)

	message := "Up"
	if conclude.Degraded {
		message = "Degraded"
	} else if conclude.Down {
		message = "Down"
	}
	if conclude.Notice != "" {
		message = fmt.Sprintf("%s - %s ", message, conclude.Notice)
	}

	availability := appinsights.NewAvailabilityTelemetry(conclude.Title, 0, conclude.Healthy)
	availability.RunLocation = c.TestLocation
	availability.Message = message
	availability.Id = uuid.New().String()
	for i := 0; i < len(conclude.Times); i++ {
		k := strconv.Itoa(i)
		total += int64(conclude.Times[i].RTT)
		availability.GetMeasurements()[k] = float64(conclude.Times[i].RTT)
	}
	if len(conclude.Times) > 0 {
		availability.Duration = time.Duration(total / int64(len(conclude.Times)))
	}
	availability.GetProperties()["ThresholdRTT"] = conclude.ThresholdRTT.String()

	// Submit the telemetry
	c.TelemetryClient.Track(availability)
	return
}
