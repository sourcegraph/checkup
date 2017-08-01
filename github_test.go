package checkup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

var checkFileRegexp = regexp.MustCompile(`\A\d+-check.json\z`)

func TestGitHub(t *testing.T) {
	results := []Result{{Title: "Testing"}}
	resultsBytes := []byte(`[{"title":"Testing"}]` + "\n")

	// test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// github client configured to use test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
	client.UploadURL = url
	defer server.Close()

	// Our subject, our specimen.
	specimen := GitHub{
		RepositoryOwner: "o",
		RepositoryName:  "r",
		CommitterName:   "John Appleseed",
		CommitterEmail:  "appleseed@example.org",
		Branch:          "b",
		Dir:             "subdir/",
		client:          client,
	}

	// files := map[string]string{}
	index := map[string]int64{
		"1501523631505010894-check.json": time.Now().UnixNano(),
		"1501525202306053005-check.json": time.Now().UnixNano(),
	}

	mux.HandleFunc("/repos/o/r/git/refs/heads/b", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method; got != "GET" {
			t.Errorf("Request method: %v, want %v", got, "GET")
		}
		fmt.Fprint(w, `
		  {
		    "ref": "refs/heads/b",
		    "url": "https://api.github.com/repos/o/r/git/refs/heads/b",
		    "object": {
		      "type": "commit",
		      "sha": "aa218f56b14c9653891f9e74264a383fa43fefbd",
		      "url": "https://api.github.com/repos/o/r/git/commits/aa218f56b14c9653891f9e74264a383fa43fefbd"
		    }
		  }`)
	})

	mux.HandleFunc("/repos/o/r/git/trees/", func(w http.ResponseWriter, r *http.Request) {
		if got := r.FormValue("recursive"); got != "1" {
			t.Errorf("Expected recursive flag to be 1, got %v", got)
		}
		desiredSHA := "aa218f56b14c9653891f9e74264a383fa43fefbd"

		if sha := strings.TrimPrefix(r.URL.Path, "/repos/o/r/git/trees/"); sha != desiredSHA {
			t.Errorf("Expected to fetch tree %s, got: %v", desiredSHA, sha)
		}

		tree := struct {
			SHA     string              `json:"sha,omitempty"`
			Entries []map[string]string `json:"entries,omitempty"`
		}{SHA: desiredSHA}
		for filename, _ := range index {
			tree.Entries = append(tree.Entries, map[string]string{
				"sha":     desiredSHA,
				"path":    filepath.Join(specimen.Dir, filename),
				"content": string(resultsBytes),
			})
		}
		err := json.NewEncoder(w).Encode(tree)
		if err != nil {
			t.Errorf("Expected no error, got %+v", err)
		}
	})

	mux.HandleFunc("/repos/o/r/contents/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, filepath.Join("/repos/o/r/contents/", specimen.Dir)+"/")
		if got := r.Method; !(got == "GET" || got == "PUT") {
			t.Errorf("Request method: %v, want GET or PUT", got)
		}
		if got := r.FormValue("ref"); got != "" && got != "heads/b" {
			t.Errorf("Expected heads/b, got %v", got)
		}
		if r.Method == "GET" && path == "index.json" {
			encodedIndex, err := json.Marshal(index)
			if err != nil {
				t.Errorf("Expected no error, got %+v", err)
			}
			base64Index := base64.StdEncoding.EncodeToString(encodedIndex)
			fmt.Fprintf(w, `
                {
                  "type": "file",
                  "encoding": "base64",
                  "name": "index.json",
                  "path": "index.json",
                  "content": "%s",
                  "sha": "abcdef123456"
                }`, base64Index)
			return
		}
		if r.Method == "PUT" && path == "index.json" {
			// 1. Enforce body contains message, committer, content (base64-encoded), sha
			// 2. Return object based on https://developer.github.com/v3/repos/contents/#update-a-file
			return
		}
		if r.Method == "GET" && checkFileRegexp.MatchString(path) {
			_, ok := index[path]
			if ok {
				base64Index := base64.StdEncoding.EncodeToString(resultsBytes)
				fmt.Fprintf(w, `
                    {
                      "type": "file",
                      "encoding": "base64",
                      "content": "%s",
                      "sha": "abcdef123456"
                    }`, base64Index)
			} else {
				http.Error(w, path+" does not exist", 403)
			}
			return
		}
		if r.Method == "PUT" && checkFileRegexp.MatchString(path) {
			shortPath := strings.TrimPrefix(path, "subdir/")
			_, ok := index[shortPath]
			index[shortPath] = time.Now().UnixNano()
			if ok {
				// You're updating the file!
				// 2. Return object based on https://developer.github.com/v3/repos/contents/#update-a-file
			} else {
				// You're creating the file!
				// 2. Return object based on https://developer.github.com/v3/repos/contents/#create-a-file
			}
			return
		}
		http.Error(w, path+" is not handled", 403)
		t.Errorf("Cannot handle %s %s (path=%s)", r.Method, r.URL.Path, path)
	})

	// Here goes!

	if err := specimen.Store(results); err != nil {
		t.Fatalf("Expected no error from Store(), got: %v", err)
	}

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
	}

	// Make sure index has timestamp of check
	ts := time.Unix(0, nsec)
	if time.Since(ts) > 1*time.Second {
		t.Errorf("Timestamp of check is %s but expected something very recent", ts)
	}

	// Make sure check file bytes are correct
	checkfile := filepath.Join(specimen.Dir, name)
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
	if _, _, err := specimen.readFile(checkfile); err != nil {
		t.Fatalf("Expected checkfile to be deleted, but Stat() returned error: %v", err)
	}
}
