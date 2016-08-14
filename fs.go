package checkup

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const indexName = "index.json"

// FS is a way to store checkup results on the local filesystem.
type FS struct {
	// The path to the directory where check files will be stored.
	Dir string `json:"dir"`
	// The URL corresponding to fs.Dir.
	URL string `json:"url"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

func (fs FS) readIndex() (index map[string]int64, err error) {
	f, err := os.Open(filepath.Join(fs.Dir, indexName))
	if os.IsNotExist(err) {
		index = map[string]int64{}
		err = nil
		return
	}
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&index)
	return
}

func (fs FS) writeIndex(index map[string]int64) error {
	f, err := os.Create(filepath.Join(fs.Dir, indexName))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(index)
}

// Store stores results on filesystem according to the configuration in fs.
func (fs FS) Store(results []Result) error {
	// Write results to a new file
	name := *GenerateFilename()
	f, err := os.Create(filepath.Join(fs.Dir, name))
	if err != nil {
		return err
	}
	err = json.NewEncoder(f).Encode(results)
	f.Close()
	if err != nil {
		return err
	}

	// Read current index file
	index, err := fs.readIndex()
	if err != nil {
		return err
	}

	// Add new file to index
	index[name] = time.Now().UnixNano()

	// Write new index
	return fs.writeIndex(index)
}

// Maintain deletes check files that are older than fs.CheckExpiry.
func (fs FS) Maintain() error {
	if fs.CheckExpiry == 0 {
		return nil
	}

	files, err := ioutil.ReadDir(fs.Dir)
	if err != nil {
		return err
	}

	index, err := fs.readIndex()
	if err != nil {
		return err
	}

	for _, f := range files {
		name, err := filepath.Rel(fs.Dir, f.Name())
		if err != nil || name == indexName {
			continue
		}

		nsec, ok := index[name]
		if !ok {
			continue
		}

		if time.Since(time.Unix(0, nsec)) > fs.CheckExpiry {
			if err := os.Remove(f.Name()); err != nil {
				return err
			}
			delete(index, name)
		}
	}

	return fs.writeIndex(index)
}
