package appinsights

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

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
	MaxRetries int `json:"max_retries,omitempty"`

	// RetryInterval specifies the time between retries in seconds.  To enable retries,
	// both RetryInterval and MaxRetries must
	// be greater than 0.  Default is 0 (disabled).
	RetryInterval int `json:"retry_interval,omitempty"`

	// Timeout specifies the number of seconds to wait for telemetry submission
	// before returning error from close() if retries are disabled.  If omitted or
	// set to 0, timeout will be 2 seconds.  If retries are enabled, this setting
	// is ignored.
	Timeout int `json:"timeout,omitempty"`

	// telemetryConfig defines the settings for client
	telemetryConfig *appinsights.TelemetryConfiguration

	// client is the appinsights.Client used to
	// send Application Insights trackAvailability() events
	client appinsights.TelemetryClient
}

// New creates a new Storage instance based on json config
func New(config json.RawMessage) (Storage, error) {
	var storage Storage
	err := json.Unmarshal(config, &storage)

	if storage.InstrumentationKey == "" {
		err = fmt.Errorf("Must supply value for InstrumentationKey")
	}
	if storage.MaxRetries < 0 {
		err = fmt.Errorf("Invalid storage max_retries: %d", storage.MaxRetries)
	}
	if storage.RetryInterval < 0 {
		err = fmt.Errorf("Invalid storage retry_interval: %d", storage.RetryInterval)
	}
	if storage.Timeout < 0 {
		err = fmt.Errorf("Invalid storage timeout: %d", storage.Timeout)
	}

	storage.telemetryConfig = appinsights.NewTelemetryConfiguration(storage.InstrumentationKey)
	if storage.TestLocation == "" {
		storage.TestLocation = "Checkup Monitor"
	}
	if storage.TestLocation == "" {
		storage.TestLocation = "Checkup Monitor"
	}
	if storage.Timeout == 0 {
		storage.Timeout = 2
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
	c.telemetryConfig.InstrumentationKey = c.InstrumentationKey
	c.client = appinsights.NewTelemetryClientFromConfig(c.telemetryConfig)
	for k, v := range c.Tags {
		c.client.Context().CommonProperties[k] = v
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
		case <-c.client.Channel().Close():
			return nil
		case <-time.After(time.Duration(c.Timeout) * time.Second):
			return fmt.Errorf("Failed to submit telemetry before timeout expired")
		}
	}
	select {
	case <-c.client.Channel().Close(time.Duration(c.RetryInterval) * time.Second):
		return nil
	case <-time.After((time.Duration(c.MaxRetries) + 1) * time.Duration(c.RetryInterval) * time.Second):
		return fmt.Errorf("Failed to submit telemetry after retries")
	}
}

// Send a result
// Multiple test result measurements will be added to the telemetry item's
// customMeasurements field
func (c Storage) send(conclude types.Result) {
	message := string(conclude.Status())

	if conclude.Notice != "" {
		message = fmt.Sprintf("%s - %s", message, conclude.Notice)
	}

	stats := conclude.ComputeStats()
	availability := appinsights.NewAvailabilityTelemetry(conclude.Title, stats.Mean, conclude.Healthy)
	availability.RunLocation = c.TestLocation
	availability.Message = message
	availability.Id = strconv.Itoa(conclude.Timestamp)
	for i := 0; i < len(conclude.Times); i++ {
		k := strconv.Itoa(i)
		availability.GetMeasurements()[k] = float64(conclude.Times[i].RTT)
	}
	availability.GetProperties()["ThresholdRTT"] = conclude.ThresholdRTT.String()

	// Submit the telemetry
	c.client.Track(availability)
	return
}
