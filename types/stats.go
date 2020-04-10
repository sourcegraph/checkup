package types

import (
	"time"
)

// Stats is a type that holds information about a Result,
// especially its various Attempts.
type Stats struct {
	Total  time.Duration `json:"total,omitempty"`
	Mean   time.Duration `json:"mean,omitempty"`
	Median time.Duration `json:"median,omitempty"`
	Min    time.Duration `json:"min,omitempty"`
	Max    time.Duration `json:"max,omitempty"`
}
