package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/sourcegraph/checkup/storage/fs"
	"github.com/sourcegraph/checkup/types"
)

// Type should match the package name
const Type = "github"

var errFileNotFound = fmt.Errorf("file not found on github")

// Storage is a way to store checkup results in a GitHub repository.
type Storage struct {
	// AccessToken is the API token used to authenticate with GitHub (required).
	AccessToken string `json:"access_token"`

	// RepositoryOwner is the account which owns the repository on GitHub (required).
	// For https://github.com/octocat/kit, the owner is "octocat".
	RepositoryOwner string `json:"repository_owner"`

	// RepositoryName is the name of the repository on GitHub (required).
	// For https://github.com/octocat/kit, the name is "kit".
	RepositoryName string `json:"repository_name"`

	// CommitterName is the display name of the user corresponding to the AccessToken (required).
	// If the AccessToken is for user @octocat, then this might be "Mona Lisa," her name.
	CommitterName string `json:"committer_name"`

	// CommitterEmail is the email address of the user corresponding to the AccessToken (required).
	// If the AccessToken is for user @octocat, then this might be "mona@github.com".
	CommitterEmail string `json:"committer_email"`

	// Branch is the git branch to store the files to (required).
	Branch string `json:"branch"`

	// Dir is the subdirectory in the Git tree in which to store the files (required).
	// For example, to write to the directory "updates" in the Git repo, this should be "updates".
	Dir string `json:"dir"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`

	client *github.Client
}

// New creates a new Storage instance based on json config
func New(config json.RawMessage) (*Storage, error) {
	storage := new(Storage)
	err := json.Unmarshal(config, &storage)
	return storage, err
}

// Type returns the storage driver package name
func (Storage) Type() string {
	return Type
}

// ensureClient builds an GitHub API client if none exists and stores it on the struct.
func (gh *Storage) ensureClient() error {
	if gh.client != nil {
		return nil
	}

	if gh.AccessToken == "" {
		return fmt.Errorf("Please specify access_token in storage configuration")
	}

	gh.client = github.NewClient(oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: gh.AccessToken},
		),
	))

	return nil
}

// fullPathName ensures the configured Dir value is present in the filename and
// returns a filename with the Dir prefixed before the input filename if necessary.
func (gh *Storage) fullPathName(filename string) string {
	if strings.HasPrefix(filename, gh.Dir) {
		return filename
	}
	return filepath.Join(gh.Dir, filename)
}

// readFile reads a file from the Git repository at its latest revision.
// This method returns the plaintext contents, the SHA associated with the contents
// If an error occurs, the contents and sha will be nil & empty.
func (gh *Storage) readFile(filename string) ([]byte, string, error) {
	if err := gh.ensureClient(); err != nil {
		return nil, "", err
	}

	contents, _, resp, err := gh.client.Repositories.GetContents(
		context.Background(),
		gh.RepositoryOwner,
		gh.RepositoryName,
		gh.fullPathName(filename),
		&github.RepositoryContentGetOptions{Ref: "heads/" + gh.Branch},
	)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, "", errFileNotFound
		}
		return nil, "", err
	}

	decoded, err := contents.GetContent()
	return []byte(decoded), *contents.SHA, err
}

// writeFile commits the contents to the Git repo at the given filename & revision.
// If the Git repo does not yet have a file at this filename, it will create the file.
// Otherwise, it will simply update the file with the new contents.
func (gh *Storage) writeFile(filename string, sha string, contents []byte) error {
	if err := gh.ensureClient(); err != nil {
		return err
	}

	var err error
	var writeFunc func(context.Context, string, string, string, *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("[checkup] store %s [ci skip]", gh.fullPathName(filename))),
		Content: contents,
		Committer: &github.CommitAuthor{
			Name:  &gh.CommitterName,
			Email: &gh.CommitterEmail,
		},
	}

	if gh.Branch != "" {
		opts.Branch = &gh.Branch
	}

	// If no SHA specified, then create the file.
	// Otherwise, update the file at the specified SHA.
	if sha == "" {
		writeFunc = gh.client.Repositories.CreateFile
		log.Printf("github: creating %s on branch '%s'", gh.fullPathName(filename), gh.Branch)
	} else {
		opts.SHA = github.String(sha)
		writeFunc = gh.client.Repositories.UpdateFile
		log.Printf("github: updating %s on branch '%s'", gh.fullPathName(filename), gh.Branch)
	}

	_, _, err = writeFunc(
		context.Background(),
		gh.RepositoryOwner,
		gh.RepositoryName,
		gh.fullPathName(filename),
		opts,
	)
	return err
}

// deleteFile deletes a file from a Git tree and returns any applicable errors.
// If an empty SHA is passed as an argument, errFileNotFound is returned.
func (gh *Storage) deleteFile(filename string, sha string) error {
	if err := gh.ensureClient(); err != nil {
		return err
	}

	if sha == "" {
		return errFileNotFound
	}

	log.Printf("github: deleting %s on branch '%s'", gh.fullPathName(filename), gh.Branch)

	_, _, err := gh.client.Repositories.DeleteFile(
		context.Background(),
		gh.RepositoryOwner,
		gh.RepositoryName,
		gh.fullPathName(filename),
		&github.RepositoryContentFileOptions{
			Message: github.String(fmt.Sprintf("[checkup] delete %s [ci skip]", gh.fullPathName(filename))),
			SHA:     github.String(sha),
			Branch:  &gh.Branch,
			Committer: &github.CommitAuthor{
				Name:  &gh.CommitterName,
				Email: &gh.CommitterEmail,
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// readIndex reads the index JSON from the Git repo into a map.
// It returns the populated map & the Git SHA associated with the contents.
// If the index file is not found in the Git repo, an empty index is returned with no error.
// If any error occurs, a nil index and empty SHA are returned along with the error.
func (gh *Storage) readIndex() (map[string]int64, string, error) {
	index := map[string]int64{}

	contents, sha, err := gh.readFile(fs.IndexName)
	if errors.Is(err, errFileNotFound) {
		return index, "", nil
	}
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(contents, &index)
	return index, sha, err
}

// writeIndex marshals the index into JSON and writes the file to the Git repo.
// It returns any errors associated with marshaling the data or writing the file.
func (gh *Storage) writeIndex(index map[string]int64, sha string) error {
	contents, err := json.Marshal(index)
	if err != nil {
		return err
	}

	return gh.writeFile(fs.IndexName, sha, contents)
}

// Store stores results in the Git repo & updates the index.
func (gh *Storage) Store(results []types.Result) error {
	// Write results to a new file
	name := *fs.GenerateFilename()
	contents, err := json.Marshal(results)
	if err != nil {
		return err
	}
	err = gh.writeFile(name, "", contents)
	if err != nil {
		return err
	}

	// Read current index file
	index, indexSHA, err := gh.readIndex()
	if err != nil {
		return err
	}

	// Add new file to index
	index[name] = time.Now().UnixNano()

	// Write new index
	return gh.writeIndex(index, indexSHA)
}

// Fetch returns a checkup record -- Not tested!
func (gh *Storage) Fetch(name string) ([]types.Result, error) {
	contents, _, err := gh.readFile(name)
	if err != nil {
		return nil, err
	}
	var r []types.Result
	err = json.Unmarshal(contents, &r)
	return r, err
}

// GetIndex returns the checkup index
func (gh *Storage) GetIndex() (map[string]int64, error) {
	m, _, e := gh.readIndex()
	return m, e
}

// Maintain deletes check files that are older than gh.CheckExpiry.
func (gh *Storage) Maintain() error {
	if gh.CheckExpiry == 0 {
		return nil
	}

	if err := gh.ensureClient(); err != nil {
		return err
	}

	index, indexSHA, err := gh.readIndex()
	if err != nil {
		return err
	}

	ref, _, err := gh.client.Git.GetRef(context.Background(), gh.RepositoryOwner, gh.RepositoryName, "heads/"+gh.Branch)
	if err != nil {
		return err
	}
	tree, _, err := gh.client.Git.GetTree(context.Background(), gh.RepositoryOwner, gh.RepositoryName, *ref.Object.SHA, true)
	if err != nil {
		return err
	}

	for _, treeEntry := range tree.Entries {
		fileName := treeEntry.GetPath()

		if fileName == filepath.Join(gh.Dir, fs.IndexName) {
			continue
		}
		if gh.Dir != "" && !strings.HasPrefix(fileName, gh.Dir) {
			log.Printf("github: maintain: skipping %s because it isn't in the configured subdirectory", fileName)
			continue
		}

		nsec, ok := index[filepath.Base(fileName)]
		if !ok {
			log.Printf("github: maintain: skipping %s because it's not in the index", fileName)
			continue
		}

		if time.Since(time.Unix(0, nsec)) > gh.CheckExpiry {
			log.Printf("github: maintain: deleting %s", fileName)
			if err = gh.deleteFile(fileName, treeEntry.GetSHA()); err != nil {
				return err
			}
			delete(index, fileName)
		}
	}

	return gh.writeIndex(index, indexSHA)
}
