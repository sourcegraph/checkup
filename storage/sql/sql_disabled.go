// +build !sql

package sql

import (
	"errors"
	"encoding/json"

	"github.com/sourcegraph/checkup/types"
)

type Storage struct{}

// New creates a new Storage instance based on json config
func New(_ json.RawMessage) (Storage, error) {
	return Storage{}, errors.New("sql data store is disabled")
}

func (_ Storage) Store(results []types.Result) error {
	return errors.New("sql data store is disabled")
}
