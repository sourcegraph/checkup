// Package checkup provides means for checking and reporting the
// status and performance of various endpoints in a distributed,
// lock-free, self-hosted fashion.
package checkup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sourcegraph/checkup/types"
)

// Checkup performs a routine checkup on endpoints or
// services.
type Checkup struct {
	// Checkers is the list of Checkers to use with
	// which to perform checks.
	Checkers []Checker `json:"checkers,omitempty"`

	// ConcurrentChecks is how many checks, at most, to
	// perform concurrently. Default is
	// DefaultConcurrentChecks.
	ConcurrentChecks int `json:"concurrent_checks,omitempty"`

	// Timestamp is the timestamp to force for all checks.
	// Useful if wanting to perform distributed check
	// "at the same time" even if they might actually
	// be a few milliseconds or seconds apart.
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Storage is the storage mechanism for saving the
	// results of checks. Required if calling Store().
	// If Storage is also a Maintainer, its Maintain()
	// method will be called by c.CheckAndStore().
	Storage Storage `json:"storage,omitempty"`

	// Notifiers are list of notifiers to invoke with
	// the results after checks from all checkers have
	// completed. Notifier may evaluate and choose to
	// send a notification of potential problems.
	Notifiers []Notifier `json:"notifiers,omitempty"`

	// Exporters are a list of services optionally
	// available to Checkers.  If configured, a check
	// will run the Export() function after finalizing
	// each Result and push the result information to
	// the external service.
	Exporters []Exporter `json:"exporters,omitempty"`
}

// Check performs the health checks. An error is only
// returned in the case of a misconfiguration or if
// any one of the Checkers returns an error.
func (c Checkup) Check() ([]types.Result, error) {
	if c.ConcurrentChecks == 0 {
		c.ConcurrentChecks = DefaultConcurrentChecks
	}
	if c.ConcurrentChecks < 0 {
		return nil, fmt.Errorf("invalid value for ConcurrentChecks: %d (must be set > 0)",
			c.ConcurrentChecks)
	}

	results := make([]types.Result, len(c.Checkers))
	errs := make(types.Errors, len(c.Checkers))
	throttle := make(chan struct{}, c.ConcurrentChecks)
	wg := sync.WaitGroup{}

	for i, checker := range c.Checkers {
		throttle <- struct{}{}
		wg.Add(1)
		go func(i int, checker Checker) {
			results[i], errs[i] = checker.Check()
			<-throttle
			wg.Done()
		}(i, checker)
	}
	wg.Wait()

	if !c.Timestamp.IsZero() {
		for i := range results {
			results[i].Timestamp = c.Timestamp.UTC().UnixNano()
		}
	}

	if !errs.Empty() {
		return results, errs
	}

	for _, service := range c.Notifiers {
		err := service.Notify(results)
		if err != nil {
			log.Printf("ERROR sending notifications for %s: %s", service.Type(), err)
		}
	}

	for _, exporter := range c.Exporters {
		err := exporter.Export(results)
		if err != nil {
			log.Printf("ERROR sending exports for %s: %s", exporter.Type(), err)
		}
	}

	return results, nil
}

// CheckAndStore performs health checks and immediately
// stores the results to the configured storage if there
// were no errors. Checks are not performed if c.Storage
// is nil. If c.Storage is also a Maintainer, Maintain()
// will be called if Store() is successful.
func (c Checkup) CheckAndStore() error {
	if c.Storage == nil {
		return fmt.Errorf("no storage mechanism defined")
	}
	results, err := c.Check()
	if err != nil {
		return err
	}

	err = c.Storage.Store(results)
	if err != nil {
		return err
	}

	if m, ok := c.Storage.(Maintainer); ok {
		return m.Maintain()
	}

	return nil
}

// CheckAndStoreEvery calls CheckAndStore every interval. It returns
// the ticker that it's using so you can stop it when you don't want
// it to run anymore. This function does NOT block (it runs the ticker
// in a goroutine). Any errors are written to the standard logger. It
// would not be wise to set an interval lower than the time it takes
// to perform the checks.
func (c Checkup) CheckAndStoreEvery(interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := c.CheckAndStore(); err != nil {
				log.Println(err)
			}
		}
	}()
	return ticker
}

// MarshalJSON marshals c into JSON with type information
// included on the interface values.
func (c Checkup) MarshalJSON() ([]byte, error) {
	// Start with the fields of c that don't require special
	// handling; unfortunately this has to mimic c's definition.
	easy := struct {
		ConcurrentChecks int       `json:"concurrent_checks,omitempty"`
		Timestamp        time.Time `json:"timestamp,omitempty"`
	}{
		ConcurrentChecks: c.ConcurrentChecks,
		Timestamp:        c.Timestamp,
	}
	result, err := json.Marshal(easy)
	if err != nil {
		return result, err
	}

	wrap := func(key string, value []byte) {
		b := append([]byte{result[0]}, []byte(`"`+key+`":`)...)
		b = append(b, value...)
		if len(result) > 2 {
			b = append(b, ',')
		}
		result = append(b, result[1:]...)
	}

	// Checkers
	if len(c.Checkers) > 0 {
		var checkers [][]byte
		for _, ch := range c.Checkers {
			chb, err := json.Marshal(ch)
			if err != nil {
				return result, err
			}
			chb = []byte(fmt.Sprintf(`{"type":"%s",%s`, ch.Type(), string(chb[1:])))
			checkers = append(checkers, chb)
		}

		allCheckers := []byte{}
		allCheckers = append(allCheckers, '[')
		allCheckers = append(allCheckers, bytes.Join(checkers, []byte(","))...)
		allCheckers = append(allCheckers, ']')
		wrap("checkers", allCheckers)
	}

	// Storage
	if c.Storage != nil {
		sb, err := json.Marshal(c.Storage)
		if err != nil {
			return result, err
		}
		sb = []byte(fmt.Sprintf(`{"type":"%s",%s`, c.Storage.Type(), string(sb[1:])))
		wrap("storage", sb)
	}

	// Notifiers
	if len(c.Notifiers) > 0 {
		var checkers [][]byte
		for _, ch := range c.Notifiers {
			chb, err := json.Marshal(ch)
			if err != nil {
				return result, err
			}

			chb = []byte(fmt.Sprintf(`{"type":"%s",%s`, ch.Type(), string(chb[1:])))
			checkers = append(checkers, chb)
		}

		allNotifiers := []byte{}
		allNotifiers = append(allNotifiers, '[')
		allNotifiers = append(allNotifiers, bytes.Join(checkers, []byte(","))...)
		allNotifiers = append(allNotifiers, ']')
		wrap("notifiers", allNotifiers)
	}

	// Exporters
	if len(c.Exporters) > 0 {
		var checkers [][]byte
		for _, ch := range c.Exporters {
			chb, err := json.Marshal(ch)
			if err != nil {
				return result, err
			}

			chb = []byte(fmt.Sprintf(`{"type":"%s",%s`, ch.Type(), string(chb[1:])))
			checkers = append(checkers, chb)
		}

		allExporters := []byte{'['}
		allExporters = append([]byte{'['}, bytes.Join(checkers, []byte(","))...)
		allExporters = append(allExporters, ']')
		wrap("exporters", allExporters)
	}

	return result, nil
}

// UnmarshalJSON unmarshales b into c. To succeed, it
// requires type information for the interface values.
func (c *Checkup) UnmarshalJSON(b []byte) error {
	// Unmarshal as much of b as we can; this requires
	// a type that doesn't implement json.Unmarshaler,
	// hence the conversion. We also know that the
	// interface types will ultimately cause an error,
	// but we can ignore it because we handle it below.
	type checkup2 *Checkup
	_ = json.Unmarshal(b, checkup2(c))

	// clean the slate
	c.Checkers = []Checker{}
	c.Notifiers = []Notifier{}
	c.Exporters = []Exporter{}

	// Begin unmarshaling interface values by
	// collecting the raw JSON
	raw := struct {
		Checkers  []json.RawMessage `json:"checkers"`
		Storage   json.RawMessage   `json:"storage"`
		Notifier  json.RawMessage   `json:"notifier"`
		Notifiers []json.RawMessage `json:"notifiers"`
		Exporter  json.RawMessage   `json:"exporter"`
		Exporters []json.RawMessage `json:"exporters"`
	}{}
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	// Then collect the concrete type information
	configTypes := struct {
		Checkers []struct {
			Type string `json:"type"`
		}
		Storage struct {
			Type string `json:"type"`
		}
		Notifier struct {
			Type string `json:"type"`
		}
		Notifiers []struct {
			Type string `json:"type"`
		}
		Exporter struct {
			Type string `json:"type"`
		}
		Exporters []struct {
			Type string `json:"type"`
		}
	}{}
	err = json.Unmarshal(b, &configTypes)
	if err != nil {
		return err
	}

	// Finally, we unmarshal the remaining values using type
	// assertions with the help of the type information
	for i, t := range configTypes.Checkers {
		checker, err := checkerDecode(t.Type, raw.Checkers[i])
		if err != nil {
			return err
		}
		c.Checkers = append(c.Checkers, checker)
	}
	if raw.Storage != nil {
		storage, err := storageDecode(configTypes.Storage.Type, raw.Storage)
		if err != nil {
			return err
		}
		c.Storage = storage
	}
	if raw.Notifier != nil {
		notifier, err := notifierDecode(configTypes.Notifier.Type, raw.Notifier)
		if err != nil {
			return err
		}
		// Move `notifier` into `notifiers[]`
		c.Notifiers = append(c.Notifiers, notifier)
	}
	for i, n := range configTypes.Notifiers {
		notifier, err := notifierDecode(n.Type, raw.Notifiers[i])
		if err != nil {
			return err
		}
		c.Notifiers = append(c.Notifiers, notifier)
	}
	if raw.Exporter != nil {
		exporter, err := exporterDecode(configTypes.Exporter.Type, raw.Exporter)
		if err != nil {
			return err
		}
		// Move `exporter` into `exporters[]`
		c.Exporters = append(c.Exporters, exporter)
	}
	for i, n := range configTypes.Exporters {
		exporter, err := exporterDecode(n.Type, raw.Exporters[i])
		if err != nil {
			return err
		}
		c.Exporters = append(c.Exporters, exporter)
	}
	return nil
}

// DefaultConcurrentChecks is how many checks,
// at most, to perform concurrently.
var DefaultConcurrentChecks = 5
