package types

// StatusText is the textual representation of the
// result of a status check.
type StatusText string

// PriorityOver returns whether s has priority over other.
// For example, a Down status has priority over Degraded.
func (s StatusText) PriorityOver(other StatusText) bool {
	if s == other {
		return false
	}
	switch s {
	case StatusDown:
		return true
	case StatusDegraded:
		if other == StatusDown {
			return false
		}
		return true
	case StatusHealthy:
		if other == StatusUnknown {
			return true
		}
		return false
	}
	return false
}

// Text representations for the status of a check.
const (
	StatusHealthy  StatusText = "healthy"
	StatusDegraded StatusText = "degraded"
	StatusDown     StatusText = "down"
	StatusUnknown  StatusText = "unknown"
)
