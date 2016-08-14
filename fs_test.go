package checkup

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTempFS() (specimen FS, err error) {
	dir, err := ioutil.TempDir("", "checkup")
	if err != nil {
		return
	}

	specimen = FS{
		Dir: dir,
	}
	return
}

func TestFS_Store(t *testing.T) {
	results := []Result{{Title: "Testing"}}
	resultsBytes := []byte(`[{"title":"Testing"}]`+"\n")

	specimen, err := newTempFS()
	if err != nil {
		t.Fatalf("Cannot create temporary directory: %v", err)
	}
	defer os.RemoveAll(specimen.Dir)

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
	for name, nsec = range index {}

	// Make sure index has timestamp of check
	ts := time.Unix(0, nsec)
	if time.Since(ts) > 1*time.Second {
		t.Errorf("Timestamp of check is %s but expected something very recent", ts)
	}

	// Make sure check file bytes are correct
	b, err := ioutil.ReadFile(filepath.Join(specimen.Dir, name))
	if err != nil {
		t.Fatalf("Expected no error reading body, got: %v", err)
	}
	if bytes.Compare(b, resultsBytes) != 0 {
		t.Errorf("Contents of file are wrong\nExpected %s\nGot %s", resultsBytes, b)
	}
}
