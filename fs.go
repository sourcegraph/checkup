package checkup

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// FS is a way to store checkup results on the local filesystem.
type FS struct {
	Dir string `json:"dir"`
	Url string `json:"url"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

// Store stores results on filesystem according to the configuration in fs.
func (fs FS) Store(results []Result) error {
	f, err := os.Create(filepath.Join(fs.Dir, *GenerateFilename()))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(results)
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

	for _, f := range files {
		if time.Since(f.ModTime()) > fs.CheckExpiry {
			if err := os.Remove(f.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}
