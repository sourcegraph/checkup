package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sourcegraph/checkup/types"
)

func TestStorage(t *testing.T) {
	results := []types.Result{{Title: "Testing"}}
	resultsBytes := []byte(`[{"title":"Testing"}]` + "\n")

	dir, err := ioutil.TempDir("", "checkup")
	if err != nil {
		t.Fatalf("Cannot create temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	specimen := Storage{
		Dir: dir,
	}

	if err := specimen.Store(results); err != nil {
		t.Fatalf("Expected no error from Store(), got: %v", err)
	}

	// Make sure index has been created
	index, err := specimen.readIndex()
	if err != nil {
		t.Fatalf("Cannot read index: %v", err)
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

	// Make sure check file bytes are correct
	checkfile := filepath.Join(specimen.Dir, name)
	b, err := ioutil.ReadFile(checkfile)
	if err != nil {
		t.Fatalf("Expected no error reading body, got: %v", err)
	}
	if !bytes.Equal(b, resultsBytes) {
		t.Errorf("Contents of file are wrong\nExpected %s\nGot %s", resultsBytes, b)
	}

	// Test the StorageReader interface
	index, err = specimen.GetIndex()
	if err != nil {
		t.Fatalf("StoreReader: cannot read index: %v", err)
	}

	if len(index) != 1 {
		t.Fatalf("StoreReader: expected length of index to be 1, but got %v", len(index))
	}

	var indexKey string
	for k := range index {
		indexKey = k // Get first (and unique) key
		break
	}
	testResults, err := specimen.Fetch(indexKey)
	if err != nil {
		t.Fatalf("StoreReader: cannot fetch contents for '%s': %v", indexKey, err)
	}
	if len(testResults) != 1 {
		t.Fatalf("StoreReader: expected length of []Result to be 1, but got %v", len(testResults))
	}

	if testResults[0].Title != results[0].Title {
		t.Fatalf("StoreReader: expected test result title to be '%s', but got '%s'", results[0].Title, testResults[0].Title)
	}

	// Make sure check file is not deleted after maintain with CheckExpiry == 0
	if err := specimen.Maintain(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, err := os.Stat(checkfile); err != nil {
		t.Fatalf("Expected not error calling Stat() on checkfile, got: %v", err)
	}

	// Make sure checkfile is deleted after maintain with CheckExpiry > 0
	specimen.CheckExpiry = 1 * time.Nanosecond
	if err := specimen.Maintain(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, err := os.Stat(checkfile); !os.IsNotExist(err) {
		t.Fatalf("Expected checkfile to be deleted, but Stat() returned error: %v", err)
	}
}
