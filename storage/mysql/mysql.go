package mysql

import (
	"encoding/json"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/sourcegraph/checkup/storage/fs"
	"github.com/sourcegraph/checkup/types"
)

// New creates a new Storage instance based on json config
func New(config json.RawMessage) (Storage, error) {
	var storage Storage
	err := json.Unmarshal(config, &storage)
	return storage, err
}

// Type returns the storage driver package name
func (Storage) Type() string {
	return Type
}

func (opts Storage) connectionString() (string, error) {
	if opts.DSN == "" {
		return "", errors.New("missing MySQL DSN")
	}
	return opts.DSN, nil
}

func (opts Storage) dbConnect() (*sqlx.DB, error) {
	dsn, err := opts.connectionString()
	if err != nil {
		return nil, err
	}
	handle, err := sqlx.Connect(opts.Type(), dsn)
	if err != nil {
		return nil, err
	}
	if opts.Create {
		_, err = handle.Exec(schema)
		if err != nil {
			handle.Close()
			return nil, err
		}
	}
	return handle, err
}

// GetIndex returns the list of check results for the database.
func (opts Storage) GetIndex() (map[string]int64, error) {
	db, err := opts.dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	idx := make(map[string]int64)
	var check struct {
		Name      string `db:"name"`
		Timestamp int64  `db:"timestamp"`
	}

	rows, err := db.Queryx(`SELECT name,timestamp FROM checks`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		err := rows.StructScan(&check)
		if err != nil {
			rows.Close()
			return nil, err
		}
		idx[check.Name] = check.Timestamp
	}

	return idx, nil
}

// Fetch fetches results of the check with given name.
func (opts Storage) Fetch(name string) ([]types.Result, error) {
	db, err := opts.dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var checkResult []byte
	var results []types.Result

	err = db.Get(&checkResult, db.Rebind(`SELECT results FROM checks WHERE name=? LIMIT 1`), name)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(checkResult, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// Store stores results in the database.
func (opts Storage) Store(results []types.Result) error {
	db, err := opts.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	name := *fs.GenerateFilename()
	contents, err := json.Marshal(results)
	if err != nil {
		return err
	}

	// Insert data
	const insertResults = `INSERT INTO checks (name, timestamp, results) VALUES (?, ?, ?)`
	_, err = db.Exec(db.Rebind(insertResults), name, time.Now().UnixNano(), contents)
	return err
}

// Maintain deletes check files that are older than opts.CheckExpiry.
func (opts Storage) Maintain() error {
	if opts.CheckExpiry == 0 {
		return nil
	}

	db, err := opts.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	const query = `DELETE FROM checks WHERE timestamp < ?`
	ts := time.Now().Add(-1 * opts.CheckExpiry).UnixNano()
	_, err = db.Exec(db.Rebind(query), ts)
	return err
}
