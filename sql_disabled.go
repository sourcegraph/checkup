// +build !sql

package checkup

import (
	"errors"
)

type SQL struct{}

func (sql SQL) Store(results []Result) error {
	return errors.New("sql data store is disabled")
}
