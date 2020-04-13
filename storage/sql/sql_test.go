// +build sql

package sql

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sourcegraph/checkup/types"
)

func TestSQL(t *testing.T) {
	results := []types.Result{{Title: "Testing"}}

	// Create temporary directory for the tests
	dir, err := ioutil.TempDir("", "checkup")
	if err != nil {
		t.Fatalf("Cannot create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	dbFile := filepath.Join(dir, "checkuptest.db")

	specimen := Storage{
		SqliteDBFile: dbFile,
	}

	if err := specimen.initialize(); err != nil {
		t.Fatalf("Could not initialize test database, got: %v", err)
	}

	if err := specimen.Store(results); err != nil {
		t.Fatalf("Expected no error from Store(), got: %v", err)
	}

	// Test GetIndex (StorageReader interface)
	index, err := specimen.GetIndex()
	if err != nil {
		t.Fatalf("StoreReader: cannot read index: %v", err)
	}

	if len(index) != 1 {
		t.Fatalf("Expected length of index to be 1, but got %v", len(index))
	}

	var (
		name string
		nsec int64
	)
	for name, nsec = range index {
	}

	// Make sure index has timestamp of check
	ts := time.Unix(0, nsec)
	if time.Since(ts) > 1*time.Second {
		t.Errorf("Timestamp of check is %s but expected something very recent", ts)
	}

	// Make sure stored data are correct
	testResults, err := specimen.Fetch(name)
	if err != nil {
		t.Fatalf("Could not fetch data, got: %v", err)
	}
	if len(testResults) != 1 {
		t.Fatalf("StoreReader: expected length of []Result to be 1, but got %v", len(testResults))
	}

	if testResults[0].Title != results[0].Title {
		t.Fatalf("Expected test result title to be '%s', but got '%s'", results[0].Title, testResults[0].Title)
	}

	// Make sure the check is not deleted after maintain with CheckExpiry == 0
	if err := specimen.Maintain(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, err := specimen.Fetch(name); err != nil {
		t.Fatalf("Expected the check to be present in the DB, got: %v", err)
	}

	// Make sure the check is not deleted after maintain with CheckExpiry == 1 day
	specimen.CheckExpiry = 24 * time.Hour
	if err := specimen.Maintain(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, err := specimen.Fetch(name); err != nil {
		t.Fatalf("Expected the check to be present in the DB, got: %v", err)
	}

	// Make sure the check is deleted after maintain with CheckExpiry > 0
	specimen.CheckExpiry = 1 * time.Nanosecond
	if err := specimen.Maintain(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, err := specimen.Fetch(name); err == nil {
		t.Fatalf("Expected not to be able to fetch the result from the DB")
	}
}
