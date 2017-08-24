package checkup

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Enable sqlite3 backend
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

// SQL is a way to store checkup results in a SQL database.
type SQL struct {
	// The sqlite3 DB where check results will be stored.
	SqliteDBFile string `json:"sqlite_db_file"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

func (sql SQL) dbConnect() (*sqlx.DB, error) {
	// TODO: support other databases
	return sqlx.Connect("sqlite3", sql.SqliteDBFile)
}

// GetIndex returns the list of check results for the database.
func (sql SQL) GetIndex() (map[string]int64, error) {
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
func (sql SQL) Fetch(name string) ([]Result, error) {
	db, err := sql.dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var checkResult []byte
	var results []Result

	err = db.Get(&checkResult, `SELECT results FROM "checks" WHERE name=? LIMIT 1`, name)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(checkResult, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// Store stores results in the database.
func (sql SQL) Store(results []Result) error {
	db, err := sql.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	name := *GenerateFilename()
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
func (sql SQL) Maintain() error {
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
func (sql SQL) initialize() error {
	db, err := sql.dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(schema)
	return err
}
