package mysql

import (
	"time"
)

// Type should match the package name
const Type = "mysql"

// Storage is a way to store checkup results in a MySQL database.
type Storage struct {
	DSN string `json:"dsn"`

	// Issue create statements for database schema
	Create bool `json:"create"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

// schema is the expected table schema (can be re-applied)
const schema = "CREATE TABLE IF NOT EXISTS `checks` (`name` VARCHAR(512) NOT NULL, `timestamp` BIGINT NOT NULL, `results` TEXT NULL, PRIMARY KEY (`name`), UNIQUE (`timestamp`)) ENGINE = InnoDB;"
