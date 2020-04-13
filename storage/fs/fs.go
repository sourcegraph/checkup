package fs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "fs"

// Storage is a way to store checkup results on the local filesystem.
type Storage struct {
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

// GetIndex returns the index from filesystem.
func (fs Storage) GetIndex() (map[string]int64, error) {
	return fs.readIndex()
}

func (fs Storage) readIndex() (map[string]int64, error) {
	index := map[string]int64{}

	f, err := os.Open(filepath.Join(fs.Dir, IndexName))
	if os.IsNotExist(err) {
		return index, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&index)
	return index, err
}

func (fs Storage) writeIndex(index map[string]int64) error {
	f, err := os.Create(filepath.Join(fs.Dir, IndexName))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(index)
}

// Fetch fetches results from filesystem for the specified index.
func (fs Storage) Fetch(name string) ([]types.Result, error) {
	f, err := os.Open(filepath.Join(fs.Dir, name))
	if err != nil {
		return nil, err
	}
	var results []types.Result
	err = json.NewDecoder(f).Decode(&results)
	f.Close()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Store stores results on filesystem according to the configuration in fs.
func (fs Storage) Store(results []types.Result) error {
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
func (fs Storage) Maintain() error {
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
		if f.Name() == IndexName {
			continue
		}

		nsec, ok := index[f.Name()]
		if !ok {
			continue
		}

		if time.Since(time.Unix(0, nsec)) > fs.CheckExpiry {
			if err := os.Remove(filepath.Join(fs.Dir, f.Name())); err != nil {
				return err
			}
			delete(index, f.Name())
		}
	}

	return fs.writeIndex(index)
}
