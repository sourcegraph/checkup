package checkup

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/github"
)

var (
	results      = []Result{{Title: "Testing"}}
	resultsBytes = []byte(`[{"title":"Testing"}]` + "\n")

	checkFileRegexp = regexp.MustCompile(`\A\d+-check.json\z`)
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
	return fmt.Sprintf("%x", sha1.Sum(data))
}

func withGitHubServer(t *testing.T, specimen GitHub, f func(*github.Client)) {
	// test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// github client configured to use test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL)
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
		Files: map[string]bool{
			"subdir/1501523631505010894-check.json": true,
			"subdir/1501525202306053005-check.json": true,
			"subdir/index.json":                     true,
		},
	}

	serverSHAForRepo := sha(toJSON(gitRepo))

	mux.HandleFunc("/repos/o/r/git/refs/heads/"+specimen.Branch, func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Processing %s %s\n", r.Method, r.URL.Path)
		if got := r.Method; got != "GET" {
			t.Errorf("Request method: %v, want %v", got, "GET")
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
		for filename, _ := range index {
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
		path := strings.TrimPrefix(r.URL.Path, filepath.Join("/repos/o/r/contents/", specimen.Dir)+"/")
		if got := r.Method; !(got == "GET" || got == "PUT" || got == "DELETE") {
			t.Errorf("Request method: %v, want GET or PUT", got)
		}
		if got := r.FormValue("ref"); got != "" && got != "heads/b" {
			t.Errorf("Expected heads/b, got %v", got)
		}
		if r.Method == "GET" && path == "index.json" {
			mustWriteJSON(w, github.RepositoryContent{
				Type:     github.String("file"),
				Encoding: github.String("base64"),
				Size:     github.Int(len(base64Encoded(toJSON(index)))),
				Name:     github.String("index.json"),
				Path:     github.String(filepath.Join(specimen.Dir, "index.json")),
				Content:  github.String(base64Encoded(toJSON(index))),
				SHA:      github.String(serverSHAForRepo),
			})
			return
		}
		if r.Method == "PUT" && path == "index.json" {
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
			if stuff.Message != "[checkup] store index.json [ci skip]" {
				t.Errorf("Expected a certain commit message, got '%s'", stuff.Message)
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
				Content: &github.RepositoryContent{
					Type:     github.String("file"),
					Encoding: github.String("base64"),
					Size:     github.Int(len(base64Encoded(toJSON(index)))),
					Name:     github.String("index.json"),
					Path:     github.String(filepath.Join(specimen.Dir, "index.json")),
					Content:  github.String(base64Encoded(toJSON(index))),
					SHA:      github.String(sha(toJSON(index))),
				},
				Commit: github.Commit{
					SHA: github.String(serverSHAForRepo),
				},
			})
			return
		}
		if r.Method == "GET" && checkFileRegexp.MatchString(path) {
			_, ok := index[path]
			if ok {
				mustWriteJSON(w, github.RepositoryContent{
					Type:     github.String("file"),
					Encoding: github.String("base64"),
					Size:     github.Int(len(base64Encoded(resultsBytes))),
					Name:     github.String(path),
					Path:     github.String(filepath.Join(specimen.Dir, path)),
					Content:  github.String(base64Encoded(resultsBytes)),
					SHA:      github.String(sha(resultsBytes)),
				})
			} else {
				http.Error(w, path+" does not exist", 404)
			}
			return
		}
		if r.Method == "PUT" && checkFileRegexp.MatchString(path) {
			index[path] = time.Now().UnixNano()

			gitRepo.Files[filepath.Join(specimen.Dir, path)] = true
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
			if stuff.Message != fmt.Sprintf("[checkup] store %s [ci skip]", path) {
				t.Errorf("Expected a certain commit message, got '%s'", stuff.Message)
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
				Content: &github.RepositoryContent{
					Type:     github.String("file"),
					Encoding: github.String("base64"),
					Size:     github.Int(len(base64Encoded(resultsBytes))),
					Name:     github.String(path),
					Path:     github.String(filepath.Join(specimen.Dir, path)),
					Content:  github.String(base64Encoded(resultsBytes)),
					SHA:      github.String(sha(resultsBytes)),
				},
				Commit: github.Commit{
					SHA: github.String(serverSHAForRepo),
				},
			})
			return
		}
		if r.Method == "DELETE" && checkFileRegexp.MatchString(path) {
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
			if expected := fmt.Sprintf("[checkup] delete %s [ci skip]", path); stuff.Message != expected {
				t.Errorf("Expected commit message to be '%s', got '%s'", expected, stuff.Message)
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

			// Ok, start modifying the in-memory state.
			if _, ok := index[path]; !ok {
				http.Error(w, "no such file: "+path, 500)
			}
			delete(index, path)

			if _, ok := gitRepo.Files[filepath.Join(specimen.Dir, path)]; !ok {
				http.Error(w, "file was deleted", 401)
			}
			gitRepo.Files[filepath.Join(specimen.Dir, path)] = false
			gitRepo.LastUpdated = time.Now().UnixNano()
			serverSHAForRepo = sha(toJSON(gitRepo))

			// Response is the same if updating or creating.
			mustWriteJSON(w, github.RepositoryContentResponse{
				Commit: github.Commit{
					SHA: github.String(serverSHAForRepo),
				},
			})
			return
		}
		http.Error(w, path+" is not handled", 403)
		t.Errorf("Cannot handle %s %s (path=%s)", r.Method, r.URL.Path, path)
	})

	f(client)
}

func TestGitHubWithSubdir(t *testing.T) {
	// Our subject, our specimen.
	specimen := &GitHub{
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
		if bytes.Compare(b, resultsBytes) != 0 {
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
		if _, _, err := specimen.readFile(checkfile); err != nil && err != errFileNotFound {
			t.Fatalf("Expected checkfile to be deleted, but Stat() returned error: %v", err)
		}
	})
}
