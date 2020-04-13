// +build sql

package sql

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"           // Enable postgresql beckend
	_ "github.com/mattn/go-sqlite3" // Enable sqlite3 backend

	"github.com/sourcegraph/checkup/storage/fs"
	"github.com/sourcegraph/checkup/types"
)

// schema is the table schema expected by the sqlite3 checkup storage.
const schema = `
CREATE TABLE checks (
    name TEXT NOT NULL PRIMARY KEY,
    timestamp INT8 NOT NULL,
    results TEXT
);
CREATE UNIQUE INDEX idx_checks_timestamp ON checks(timestamp);
`

// Storage is a way to store checkup results in a SQL database.
type Storage struct {
	// SqliteDBFile is the sqlite3 DB where check results will be stored.
	SqliteDBFile string `json:"sqlite_db_file,omitempty"`

	// PostgreSQL contains the Postgres connection settings.
	PostgreSQL *struct {
		Host     string `json:"host,omitempty"`
		Port     int    `json:"port,omitempty"`
		User     string `json:"user"`
		Password string `json:"password,omitempty"`
		DBName   string `json:"dbname"`
		SSLMode  string `json:"sslmode,omitempty"`
	} `json:"postgresql"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

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

func (sql Storage) dbConnect() (*sqlx.DB, error) {
	// Only one SQL backend can be present
	if sql.SqliteDBFile != "" && sql.PostgreSQL != nil {
		return nil, errors.New("several SQL backends are configured")
	}

	// SQLite3 configuration
	if sql.SqliteDBFile != "" {
		return sqlx.Connect("sqlite3", sql.SqliteDBFile)
	}

	// PostgreSQL configuration
	if sql.PostgreSQL != nil {
		var pgOptions string
		if sql.PostgreSQL.DBName == "" {
			return nil, errors.New("missing PostgreSQL database name")
		}
		if sql.PostgreSQL.User == "" {
			return nil, errors.New("missing PostgreSQL username")
		}
		if sql.PostgreSQL.Host != "" {
			pgOptions += " host=" + sql.PostgreSQL.Host
		}
		if sql.PostgreSQL.Port != 0 {
			pgOptions += " port=" + strconv.Itoa(sql.PostgreSQL.Port)
		}
		pgOptions += " user=" + sql.PostgreSQL.User
		if sql.PostgreSQL.Password != "" {
			pgOptions += " password=" + sql.PostgreSQL.Password
		}
		pgOptions += " dbname=" + sql.PostgreSQL.DBName
		if sql.PostgreSQL.SSLMode != "" {
			pgOptions += " sslmode=" + sql.PostgreSQL.SSLMode
		}
		return sqlx.Connect("postgres", pgOptions)
	}

	// TODO: MySQL backend?

	return nil, errors.New("no configured database backend")
}

// GetIndex returns the list of check results for the database.
func (sql Storage) GetIndex() (map[string]int64, error) {
	db, err := sql.dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	idx := make(map[string]int64)
	var check struct {
		Name      string `db:"name"`
		Timestamp int64  `db:"timestamp"`
	}

	rows, err := db.Queryx(`SELECT name,timestamp FROM "checks"`)
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
func (sql Storage) Fetch(name string) ([]types.Result, error) {
	db, err := sql.dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var checkResult []byte
	var results []types.Result

	err = db.Get(&checkResult, `SELECT results FROM "checks" WHERE name=$1 LIMIT 1`, name)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(checkResult, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// Store stores results in the database.
func (sql Storage) Store(results []types.Result) error {
	db, err := sql.dbConnect()
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
	const insertResults = `INSERT INTO "checks" (name, timestamp, results) VALUES (?, ?, ?)`
	_, err = db.Exec(insertResults, name, time.Now().UnixNano(), contents)
	return err
}

// Maintain deletes check files that are older than sql.CheckExpiry.
func (sql Storage) Maintain() error {
	if sql.CheckExpiry == 0 {
		return nil
	}

	db, err := sql.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	const st = `DELETE FROM "checks" WHERE timestamp < ?`
	ts := time.Now().Add(-1 * sql.CheckExpiry).UnixNano()
	_, err = db.Exec(st, ts)
	return err
}

// initialize creates the "checks" table in the database.
func (sql Storage) initialize() error {
	db, err := sql.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(schema)
	return err
}
