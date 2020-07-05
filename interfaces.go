package checkup

import (
	"github.com/sourcegraph/checkup/types"
)

// Checker can create a types.Result.
type Checker interface {
	Type() string
	Check() (types.Result, error)
}

// Storage can store results.
type Storage interface {
	Type() string
	Store([]types.Result) error
}

// StorageReader can read results from the Storage.
type StorageReader interface {
	// Fetch returns the contents of a check file.
	Fetch(checkFile string) ([]types.Result, error)
	// GetIndex returns the storage index, as a map where keys are check
	// result filenames and values are the associated check timestamps.
	GetIndex() (map[string]int64, error)
}

// Maintainer can maintain a store of results by
// deleting old check files that are no longer
// needed or performing other required tasks.
type Maintainer interface {
	Maintain() error
}

// Notifier can notify ops or sysadmins of
// potential problems. A Notifier should keep
// state to avoid sending repeated notices
// more often than the admin would like.
type Notifier interface {
	Type() string
	Notify([]types.Result) error
}

// Exporter is a service to send
// Result data for additional processing.
type Exporter interface {
	Type() string
	Export([]types.Result) error
}

// Provisioner is a type of storage mechanism that can
// provision itself for use with checkup. Provisioning
// need only happen once and is merely a convenience
// so that the user can get up and running with their
// status page more quickly. Presumably, the info
// returned from Provision should be used on the status
// page side of things ot access the check files (like
// a key pair that is used for read-only access).
type Provisioner interface {
	Provision() (types.ProvisionInfo, error)
}
