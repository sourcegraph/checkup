package checkup

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

// GitHub is a way to store checkup results in a GitHub repository.
type GitHub struct {
	AccessToken     string `json:"access_token"`
	RepositoryOwner string `json:"repository_owner"`
	RepositoryName  string `json:"repository_name"`
	CommitterName   string `json:"committer_name"`
	CommitterEmail  string `json:"committer_email"`
	Branch          string `json:"branch"`
	Dir             string `json:"dir"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`

	client *github.Client
}

func (gh GitHub) fullPathName(filename string) string {
	if strings.HasPrefix(filename, gh.Dir) {
		return filename
	} else {
		return filepath.Join(gh.Dir, filename)
	}
}

func (gh GitHub) readFile(filename string) ([]byte, string, error) {
	contents, _, _, err := gh.client.Repositories.GetContents(
		context.Background(),
		gh.RepositoryOwner,
		gh.RepositoryName,
		gh.fullPathName(filename),
		&github.RepositoryContentGetOptions{Ref: "heads/" + gh.Branch},
	)
	if err != nil {
		return nil, "", err
	}

	decoded, err := contents.GetContent()
	return []byte(decoded), *contents.SHA, err
}

func (gh GitHub) writeFile(filename string, sha string, contents []byte) error {
	var err error
	var writeFunc func(context.Context, string, string, string, *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(fmt.Sprintf("[checkup] store %s [ci skip]", filename)),
		Content: contents,
		Committer: &github.CommitAuthor{
			Name:  &gh.CommitterName,
			Email: &gh.CommitterEmail,
		},
	}

	// If no SHA specified, then create the file.
	// Otherwise, update the file at the specified SHA.
	if sha == "" {
		opts.Branch = &gh.Branch
		writeFunc = gh.client.Repositories.CreateFile
	} else {
		opts.SHA = github.String(sha)
		writeFunc = gh.client.Repositories.UpdateFile
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

// deleteFile deletes a file from a Git tree and returns the new SHA for the ref
// and any applicable errors. If an error occurs, the input SHA is returned along
// with the error.
func (gh GitHub) deleteFile(filename string, sha string) (string, error) {
	commit, _, err := gh.client.Repositories.DeleteFile(
		context.Background(),
		gh.RepositoryOwner,
		gh.RepositoryName,
		gh.fullPathName(filename),
		&github.RepositoryContentFileOptions{
			Message: github.String(fmt.Sprintf("[checkup] delete %s [ci skip]", filename)),
			SHA:     github.String(sha),
			Committer: &github.CommitAuthor{
				Name:  &gh.CommitterName,
				Email: &gh.CommitterEmail,
			},
		},
	)
	if err != nil {
		return sha, err
	}

	return *commit.Commit.SHA, nil
}

func (gh GitHub) readIndex() (map[string]int64, string, error) {
	index := map[string]int64{}

	contents, sha, err := gh.readFile(indexName)
	if err != nil {
		return nil, "", err
	}

	err = json.Unmarshal(contents, &index)
	return index, sha, err
}

func (gh GitHub) writeIndex(index map[string]int64, sha string) error {
	contents, err := json.Marshal(index)
	if err != nil {
		return err
	}

	return gh.writeFile(indexName, sha, contents)
}

// Store stores results on filesystem according to the configuration in fs.
func (gh GitHub) Store(results []Result) error {
	// Write results to a new file
	name := *GenerateFilename()
	contents, err := json.Marshal(results)
	if err != nil {
		return err
	}
	err = gh.writeFile(name, "", contents)

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

// Maintain deletes check files that are older than fs.CheckExpiry.
func (gh GitHub) Maintain() error {
	if gh.CheckExpiry == 0 {
		return nil
	}

	ref, _, err := gh.client.Git.GetRef(context.Background(), gh.RepositoryOwner, gh.RepositoryName, "heads/"+gh.Branch)
	if err != nil {
		return err
	}
	tree, _, err := gh.client.Git.GetTree(context.Background(), gh.RepositoryOwner, gh.RepositoryName, *ref.Object.SHA, true)
	if err != nil {
		return err
	}
	var files []string
	for _, treeEntry := range tree.Entries {
		files = append(files, *treeEntry.Path)
	}

	index, sha, err := gh.readIndex()
	if err != nil {
		return err
	}

	for _, fileName := range files {
		if fileName == indexName {
			continue
		}
		if gh.Dir != "" && !strings.HasPrefix(fileName, gh.Dir) {
			continue
		}

		nsec, ok := index[fileName]
		if !ok {
			continue
		}

		if time.Since(time.Unix(0, nsec)) > gh.CheckExpiry {
			sha, err = gh.deleteFile(fileName, sha)
			if err != nil {
				return err
			}
			delete(index, fileName)
		}
	}

	return gh.writeIndex(index, sha)
}
