// +build !sql

package sql

import (
	"encoding/json"
	"errors"

	"github.com/sourcegraph/checkup/types"
)

type Storage struct{}

// New creates a new Storage instance based on json config
func New(_ json.RawMessage) (Storage, error) {
	return Storage{}, errors.New("sql data store is disabled")
}

// Type returns the storage driver package name
func (Storage) Type() string {
	return Type
}

func (Storage) Store(results []types.Result) error {
	return errors.New("sql data store is disabled")
}
