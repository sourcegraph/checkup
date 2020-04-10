package checkup

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sourcegraph/checkup/storage/fs"
	"github.com/sourcegraph/checkup/storage/github"
	"github.com/sourcegraph/checkup/storage/s3"
	"github.com/sourcegraph/checkup/storage/sql"
)

func storageDecode(typeName string, config json.RawMessage) (Storage, error) {
	switch typeName {
	case "s3":
		return s3.New(config)
	case "github":
		return github.New(config)
	case "fs":
		return fs.New(config)
	case "sql":
		return sql.New(config)
	default:
		return nil, errors.New(strings.Replace(errUnknownStorageType, "%T", typeName, -1))
	}
}

func storageType(ch interface{}) (string, error) {
	var typeName string
	switch ch.(type) {
	case s3.Storage, *s3.Storage:
		typeName = "s3"
	case github.Storage, *github.Storage:
		typeName = "github"
	case fs.Storage, *fs.Storage:
		typeName = "fs"
	case sql.Storage, *sql.Storage:
		typeName = "sql"
	default:
		return "", fmt.Errorf(errUnknownStorageType, ch)
	}
	return typeName, nil
}
