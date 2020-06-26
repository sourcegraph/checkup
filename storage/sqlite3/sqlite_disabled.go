// +build !sqlite3

package sqlite3

import (
	"encoding/json"
	"errors"

	"github.com/sourcegraph/checkup/types"
)

var errStoreDisabled = errors.New("sqlite data store is disabled")

// New creates a new Storage instance based on json config
func New(_ json.RawMessage) (Storage, error) {
	return Storage{}, errStoreDisabled
}

// Type returns the storage driver package name
func (Storage) Type() string {
	return Type
}

func (Storage) Store(results []types.Result) error {
	return errStoreDisabled
}
