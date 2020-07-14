package checkup

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/checkup/storage/appinsights"
	"github.com/sourcegraph/checkup/storage/fs"
	"github.com/sourcegraph/checkup/storage/github"
	"github.com/sourcegraph/checkup/storage/mysql"
	"github.com/sourcegraph/checkup/storage/postgres"
	"github.com/sourcegraph/checkup/storage/s3"
	"github.com/sourcegraph/checkup/storage/sql"
	"github.com/sourcegraph/checkup/storage/sqlite3"
)

func storageDecode(typeName string, config json.RawMessage) (Storage, error) {
	switch typeName {
	case sqlite3.Type:
		return sqlite3.New(config)
	case mysql.Type:
		return mysql.New(config)
	case postgres.Type:
		return postgres.New(config)
	case s3.Type:
		return s3.New(config)
	case github.Type:
		return github.New(config)
	case fs.Type:
		return fs.New(config)
	case sql.Type:
		return sql.New(config)
	case appinsights.Type:
		return appinsights.New(config)
	default:
		return nil, fmt.Errorf(errUnknownStorageType, typeName)
	}
}
