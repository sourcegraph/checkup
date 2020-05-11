package github

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/github"

	"github.com/sourcegraph/checkup/types"
)

const (
	GET    = "GET"
	DELETE = "DELETE"
	PUT    = "PUT"
)

var (
	results      = []types.Result{{Title: "Testing"}}
	resultsBytes = []byte(`[{"title":"Testing"}]`)
)

func mustWriteJSON(w io.Writer, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		panic(err)
	}
}

func base64Encoded(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

func toJSON(data interface{}) []byte {
	encoded, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return encoded
}

func sha(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func pathForGitRepo(path string) string {
	return strings.TrimPrefix(path, "/")
}

func pathForIndex(path string) string {
	return filepath.Base(path)
}

func repositoryContent(path, serverSHAForRepo string, data interface{}) *github.RepositoryContent {
	return &github.RepositoryContent{
		Type:     github.String("file"),
		Encoding: github.String("base64"),
		Size:     github.Int(len(base64Encoded(toJSON(data)))),
		Name:     github.String(filepath.Base(path)),
		Path:     github.String(path),
		Content:  github.String(base64Encoded(toJSON(data))),
		SHA:      github.String(serverSHAForRepo),
	}
}

func withGitHubServer(t *testing.T, specimen Storage, f func(*github.Client)) {
	// test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// github client configured to use test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url
	defer server.Close()

	// files := map[string]string{}
	index := map[string]int64{
		"1501523631505010894-check.json": time.Now().UnixNano(),
		"1501525202306053005-check.json": time.Now().UnixNano(),
	}
	gitRepo := struct {
		LastUpdated int64
		Files       map[string]bool
	}{
		LastUpdated: time.Now().UnixNano(),
		Files:       map[string]bool{},
	}

	fixtureFiles := []string{"1501523631505010894-check.json", "1501525202306053005-check.json", "index.json"}
	for _, file := range fixtureFiles {
		gitRepo.Files[filepath.Join(specimen.Dir, file)] = true
	}

	serverSHAForRepo := sha(toJSON(gitRepo))

	mux.HandleFunc("/repos/o/r/git/refs/heads/"+specimen.Branch, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Processing %s %s\n", r.Method, r.URL.Path)
		if got, want := r.Method, GET; got != want {
			t.Errorf("Request method: %v, want %v", got, want)
		}
		mustWriteJSON(w, github.Reference{
			Ref: github.String("refs/heads/" + specimen.Branch),
			Object: &github.GitObject{
				Type: github.String("commit"),
				SHA:  github.String(serverSHAForRepo),
			},
		})
	})

	mux.HandleFunc("/repos/o/r/git/trees/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Processing %s %s\n", r.Method, r.URL.Path)
		if got := r.FormValue("recursive"); got != "1" {
			t.Errorf("Expected recursive flag to be 1, got %v", got)
		}

		if sha := strings.TrimPrefix(r.URL.Path, "/repos/o/r/git/trees/"); sha != serverSHAForRepo {
			t.Errorf("Expected to fetch tree %s, got: %v", serverSHAForRepo, sha)
		}

		entries := []github.TreeEntry{
			{
				SHA:     github.String(sha(toJSON(index))),
				Path:    github.String(filepath.Join(specimen.Dir, "index.json")),
				Content: github.String(base64Encoded(toJSON(index))),
			},
		}
		for filename := range index {
			entries = append(entries, github.TreeEntry{
				SHA:     github.String(sha(resultsBytes)),
				Path:    github.String(filepath.Join(specimen.Dir, filename)),
				Content: github.String(string(resultsBytes)),
			})
		}

		mustWriteJSON(w, github.Tree{
			SHA:     github.String(serverSHAForRepo),
			Entries: entries,
		})
	})

	mux.HandleFunc("/repos/o/r/contents/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Processing %s %s\n", r.Method, r.URL.Path)
		path := strings.TrimPrefix(r.URL.Path, "/repos/o/r/contents")
		if got := r.Method; !(got == GET || got == PUT || got == DELETE) {
			t.Errorf("Request method: %v, want GET, PUT or DELETE", got)
		}
		if got := r.FormValue("ref"); got != "" && got != "heads/b" {
			t.Errorf("Expected heads/b, got %v", got)
		}
		if r.Method == GET && strings.HasSuffix(path, "/index.json") {
			mustWriteJSON(w, repositoryContent(path, serverSHAForRepo, index))
			return
		}
		if r.Method == PUT && strings.HasSuffix(path, "/index.json") {
			// 1. Enforce body contains message, committer, content (base64-encoded), sha
			stuff := struct {
				Message   string            `json:"message"`
				Content   string            `json:"content"`
				SHA       string            `json:"sha"`
				Committer map[string]string `json:"committer"`
			}{}
			if err := json.NewDecoder(r.Body).Decode(&stuff); err != nil {
				t.Errorf("Expected body to decode fine, but got %+v", err)
			}
			if expected := fmt.Sprintf("[checkup] store %s [ci skip]", strings.TrimPrefix(path, "/")); stuff.Message != expected {
				t.Errorf("Expected commit message '%s', got '%s'", expected, stuff.Message)
			}
			if stuff.SHA != serverSHAForRepo {
				t.Errorf("Expected SHA to be %s, got '%s'", serverSHAForRepo, stuff.SHA)
			}
			if stuff.Committer["email"] != specimen.CommitterEmail {
				t.Errorf("Expected email to be %s, got %s", specimen.CommitterEmail, stuff.Committer["email"])
			}
			if stuff.Committer["name"] != specimen.CommitterName {
				t.Errorf("Expected name to be %s, got %s", specimen.CommitterName, stuff.Committer["name"])
			}
			// 2. Return object based on https://developer.github.com/v3/repos/contents/#update-a-file
			gitRepo.LastUpdated = time.Now().UnixNano()
			serverSHAForRepo = sha(toJSON(gitRepo))
			mustWriteJSON(w, github.RepositoryContentResponse{
				Content: repositoryContent(path, serverSHAForRepo, index),
				Commit:  github.Commit{SHA: github.String(serverSHAForRepo)},
			})
			return
		}
		if r.Method == GET && strings.HasSuffix(path, "-check.json") {
			_, ok := index[pathForIndex(path)]
			if ok {
				mustWriteJSON(w, repositoryContent(path, serverSHAForRepo, results))
			} else {
				http.Error(w, filepath.Base(path)+" does not exist", 404)
			}
			return
		}
		if r.Method == PUT && strings.HasSuffix(path, "-check.json") {
			index[pathForIndex(path)] = time.Now().UnixNano()

			gitRepo.Files[pathForGitRepo(path)] = true
			gitRepo.LastUpdated = time.Now().UnixNano()
			serverSHAForRepo = sha(toJSON(gitRepo))

			// 1. Enforce body contains message, committer, content (base64-encoded), sha
			stuff := struct {
				Message   string            `json:"message"`
				Content   string            `json:"content"`
				SHA       string            `json:"sha"`
				Committer map[string]string `json:"committer"`
			}{}
			if err := json.NewDecoder(r.Body).Decode(&stuff); err != nil {
				t.Errorf("Expected body to decode fine, but got %+v", err)
			}
			if expected := fmt.Sprintf("[checkup] store %s [ci skip]", strings.TrimPrefix(path, "/")); stuff.Message != expected {
				t.Errorf("Expected commit message '%s', got '%s'", expected, stuff.Message)
			}
			if stuff.SHA != "" {
				t.Errorf("Expected SHA to be empty, got '%s'", stuff.SHA)
			}
			if stuff.Committer["email"] != specimen.CommitterEmail {
				t.Errorf("Expected email to be %s, got %s", specimen.CommitterEmail, stuff.Committer["email"])
			}
			if stuff.Committer["name"] != specimen.CommitterName {
				t.Errorf("Expected name to be %s, got %s", specimen.CommitterName, stuff.Committer["name"])
			}

			// Response is the same if updating or creating.
			mustWriteJSON(w, github.RepositoryContentResponse{
				Content: repositoryContent(path, serverSHAForRepo, results),
				Commit: github.Commit{
					SHA: github.String(serverSHAForRepo),
				},
			})
			return
		}
		if r.Method == DELETE && strings.HasSuffix(path, "-check.json") {
			// 1. Enforce body contains message, committer, content (base64-encoded), sha
			stuff := struct {
				Message   string            `json:"message"`
				Content   string            `json:"content"`
				SHA       string            `json:"sha"`
				Committer map[string]string `json:"committer"`
			}{}
			if err := json.NewDecoder(r.Body).Decode(&stuff); err != nil {
				t.Errorf("Expected body to decode fine, but got %+v", err)
			}
			if expected := fmt.Sprintf("[checkup] delete %s [ci skip]", strings.TrimPrefix(path, "/")); stuff.Message != expected {
				t.Errorf("Expected commit message to be '%s', got '%s'", expected, stuff.Message)
			}
			if stuff.SHA != sha(resultsBytes) {
				t.Errorf("Expected SHA to be %s, got '%s'", sha(resultsBytes), stuff.SHA)
			}
			if stuff.Committer["email"] != specimen.CommitterEmail {
				t.Errorf("Expected email to be %s, got %s", specimen.CommitterEmail, stuff.Committer["email"])
			}
			if stuff.Committer["name"] != specimen.CommitterName {
				t.Errorf("Expected name to be %s, got %s", specimen.CommitterName, stuff.Committer["name"])
			}

			// Ok, start modifying the in-memory state.
			if _, ok := index[pathForIndex(path)]; !ok {
				http.Error(w, "no such file: "+filepath.Base(path), 500)
			}
			delete(index, filepath.Base(path))

			if _, ok := gitRepo.Files[pathForGitRepo(path)]; !ok {
				fmt.Printf("path=%s index=%+v repo: %+v\n", path, index, gitRepo.Files)
				http.Error(w, "file was deleted", http.StatusUnauthorized)
				return
			}
			gitRepo.Files[pathForGitRepo(path)] = false
			gitRepo.LastUpdated = time.Now().UnixNano()
			// Don't update the server SHA.

			// Response is the same if updating or creating.
			mustWriteJSON(w, github.RepositoryContentResponse{
				Commit: github.Commit{
					SHA: github.String(serverSHAForRepo),
				},
			})
			return
		}
		http.Error(w, path+" is not handled", http.StatusForbidden)
		t.Errorf("Cannot handle %s %s (path=%s)", r.Method, r.URL.Path, path)
	})

	f(client)
}

func TestGitHubWithoutSubdir(t *testing.T) {
	// Our subject, our specimen.
	specimen := &Storage{
		RepositoryOwner: "o",
		RepositoryName:  "r",
		CommitterName:   "John Appleseed",
		CommitterEmail:  "appleseed@example.org",
		Branch:          "b",
		Dir:             "",
	}

	withGitHubServer(t, *specimen, func(client *github.Client) {
		specimen.client = client

		if err := specimen.Store(results); err != nil {
			t.Fatalf("Expected no error from Store(), got: %v", err)
		}

		fmt.Println("Done with Store()")

		// Make sure index has been created
		index, _, err := specimen.readIndex()
		if err != nil {
			t.Fatalf("Cannot read index: %v", err)
		}

		if len(index) != 3 {
			t.Fatalf("Expected length of index to be 3, but got %v", len(index))
		}

		var (
			name string
			nsec int64
		)
		for name, nsec = range index {
			break
		}

		// Make sure index has timestamp of check
		ts := time.Unix(0, nsec)
		if time.Since(ts) > 1*time.Second {
			t.Errorf("Timestamp of check is %s but expected something very recent", ts)
		}

		// Make sure check file bytes are correct
		fmt.Printf("checking %s\n", name)
		b, _, err := specimen.readFile(name)
		if err != nil {
			t.Fatalf("Expected no error reading body, got: %v", err)
		}
		if !bytes.Equal(b, resultsBytes) {
			t.Errorf("Contents of file are wrong\nExpected %s\nGot %s", resultsBytes, b)
		}

		// Make sure check file is not deleted after maintain with CheckExpiry == 0
		if err := specimen.Maintain(); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if _, _, err := specimen.readFile(name); err != nil {
			t.Fatalf("Expected not error calling Stat() on checkfile, got: %v", err)
		}

		// Make sure checkfile is deleted after maintain with CheckExpiry > 0
		specimen.CheckExpiry = 1 * time.Nanosecond
		if err := specimen.Maintain(); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if _, _, err := specimen.readFile(name); err != nil && !errors.Is(err, errFileNotFound) {
			t.Fatalf("Expected checkfile to be deleted, but Stat() returned error: %v", err)
		}
	})
}

func TestGitHubWithSubdir(t *testing.T) {
	// Our subject, our specimen.
	specimen := &Storage{
		RepositoryOwner: "o",
		RepositoryName:  "r",
		CommitterName:   "John Appleseed",
		CommitterEmail:  "appleseed@example.org",
		Branch:          "b",
		Dir:             "subdir/",
	}

	withGitHubServer(t, *specimen, func(client *github.Client) {
		specimen.client = client

		if err := specimen.Store(results); err != nil {
			t.Fatalf("Expected no error from Store(), got: %v", err)
		}

		fmt.Println("Done with Store()")

		// Make sure index has been created
		index, _, err := specimen.readIndex()
		if err != nil {
			t.Fatalf("Cannot read index: %v", err)
		}

		if len(index) != 3 {
			t.Fatalf("Expected length of index to be 3, but got %v", len(index))
		}

		var (
			name string
			nsec int64
		)
		for name, nsec = range index {
			break
		}

		// Make sure index has timestamp of check
		ts := time.Unix(0, nsec)
		if time.Since(ts) > 1*time.Second {
			t.Errorf("Timestamp of check is %s but expected something very recent", ts)
		}

		// Make sure check file bytes are correct
		checkfile := filepath.Join(specimen.Dir, name)
		fmt.Printf("checking %s\n", checkfile)
		b, _, err := specimen.readFile(checkfile)
		if err != nil {
			t.Fatalf("Expected no error reading body, got: %v", err)
		}
		if !bytes.Equal(b, resultsBytes) {
			t.Errorf("Contents of file are wrong\nExpected %s\nGot %s", resultsBytes, b)
		}

		// Make sure check file is not deleted after maintain with CheckExpiry == 0
		if err := specimen.Maintain(); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if _, _, err := specimen.readFile(checkfile); err != nil {
			t.Fatalf("Expected not error calling Stat() on checkfile, got: %v", err)
		}

		// Make sure checkfile is deleted after maintain with CheckExpiry > 0
		specimen.CheckExpiry = 1 * time.Nanosecond
		if err := specimen.Maintain(); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if _, _, err := specimen.readFile(checkfile); err != nil && !errors.Is(err, errFileNotFound) {
			t.Fatalf("Expected checkfile to be deleted, but Stat() returned error: %v", err)
		}
	})
}
