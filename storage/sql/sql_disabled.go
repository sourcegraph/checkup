// +build !sql

package sql

import (
	"encoding/json"
	"errors"

	"github.com/sourcegraph/checkup/types"
)

type Storage struct{}

var errStoreDisabled = errors.New("sql data store is disabled")

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
