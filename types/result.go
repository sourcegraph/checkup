package types

import (
	"fmt"
	"sort"
	"time"

	"github.com/fatih/color"
)

// Result is the result of a health check.
type Result struct {
	// Title is the title (or name) of the thing that was checked.
	// It should be unique, as it acts like an identifier to users.
	Title string `json:"title,omitempty"`

	// Endpoint is the URL/address/path/identifier/locator/whatever
	// of what was checked.
	Endpoint string `json:"endpoint,omitempty"`

	// Timestamp is when the check occurred; UTC UnixNano format.
	Timestamp int64 `json:"timestamp,omitempty"`

	// Times is a list of each individual check attempt.
	Times Attempts `json:"times,omitempty"`

	// ThresholdRTT is the maximum RTT that was tolerated before
	// considering performance to be degraded. Leave 0 if irrelevant.
	ThresholdRTT time.Duration `json:"threshold,omitempty"`

	// Stats is a placeholder to hold ComputeStats() if ThresholdRTT > 0
	Stats Stats `json:"-"`

	// Healthy, Degraded, and Down contain the ultimate conclusion
	// about the endpoint. Exactly one of these should be true;
	// any more or less is a bug.
	Healthy  bool `json:"healthy,omitempty"`
	Degraded bool `json:"degraded,omitempty"`
	Down     bool `json:"down,omitempty"`

	// Notice contains a description of some condition of this
	// check that might have affected the result in some way.
	// For example, that the median RTT is above the threshold.
	Notice string `json:"notice,omitempty"`

	// Message is an optional message to show on the status page.
	// For example, what you're doing to fix a problem.
	Message string `json:"message,omitempty"`
}

func NewResult() Result {
	return Result{
		Timestamp: Timestamp(),
	}
}

// ComputeStats computes basic statistics about r.
func (r Result) ComputeStats() Stats {
	var s Stats

	for _, a := range r.Times {
		s.Total += a.RTT
		if a.RTT < s.Min || s.Min == 0 {
			s.Min = a.RTT
		}
		if a.RTT > s.Max || s.Max == 0 {
			s.Max = a.RTT
		}
	}
	sorted := make(Attempts, len(r.Times))
	copy(sorted, r.Times)
	sort.Sort(sorted)

	half := len(sorted) / 2
	if len(sorted)%2 == 0 {
		s.Median = (sorted[half-1].RTT + sorted[half].RTT) / 2
	} else {
		s.Median = sorted[half].RTT
	}

	s.Mean = time.Duration(int64(s.Total) / int64(len(r.Times)))

	return s
}

// DisableColor disables ANSI colors in the Result default string.
func DisableColor() {
	color.NoColor = true
}

// String returns a human-readable rendering of r.
func (r Result) String() string {
	stats := r.ComputeStats()
	s := fmt.Sprintf("== %s - %s\n", r.Title, r.Endpoint)
	s += fmt.Sprintf("  Threshold: %s\n", r.ThresholdRTT)
	s += fmt.Sprintf("        Max: %s\n", stats.Max)
	s += fmt.Sprintf("        Min: %s\n", stats.Min)
	s += fmt.Sprintf("     Median: %s\n", stats.Median)
	s += fmt.Sprintf("       Mean: %s\n", stats.Mean)
	s += fmt.Sprintf("        All: %v\n", r.Times)
	statusLine := fmt.Sprintf(" Assessment: %v\n", r.Status())
	switch r.Status() {
	case StatusHealthy:
		statusLine = color.GreenString(statusLine)
	case StatusDegraded:
		statusLine = color.YellowString(statusLine)
	case StatusDown:
		statusLine = color.RedString(statusLine)
	}
	s += statusLine
	return s
}

// Status returns a text representation of the overall status
// indicated in r.
func (r Result) Status() StatusText {
	switch {
	case r.Down:
		return StatusDown
	case r.Degraded:
		return StatusDegraded
	case r.Healthy:
		return StatusHealthy
	}
	return StatusUnknown
}
