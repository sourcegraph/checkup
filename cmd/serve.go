package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sourcegraph/checkup"
	"github.com/sourcegraph/checkup/storage/fs"
)

var listenAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve files from a local/remote storage service",
	Long: `Use the serve command to start a minimal http server that will serve
files from the configured storage provider. The intended use is to
provide a web server that will read any stored checks from storages like
fs, mysql, postgresql, sqlite3....

By default, checkup.json configuration file will be loaded and used.`,
	Run: func(cmd *cobra.Command, args []string) {
		var prov checkup.StorageReader
		var err error

		prov, err = storageReaderConfig()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("OK...")

		statuspage := http.FileServer(http.Dir("./statuspage/"))

		mux := http.NewServeMux()
		for _, folder := range []string{"js", "css", "images"} {
			mux.Handle("/"+folder+"/", statuspage)
		}
		mux.HandleFunc("/", serveHandler(prov))

		if err := http.ListenAndServe(listenAddr, mux); err != nil {
			log.Fatal(err)
		}
	},
}

func serveHandler(reader checkup.StorageReader) http.HandlerFunc {
	writeError := func(w http.ResponseWriter, err error) {
		response := struct {
			Error struct {
				Message string
			}
		}{}
		response.Error.Message = err.Error()
		json.NewEncoder(w).Encode(response)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		requestedFile := strings.TrimLeft(r.URL.Path, "/")
		if requestedFile == "" || requestedFile == "index.html" {
			http.ServeFile(w, r, "statuspage/index.html")
		}
		index, err := reader.GetIndex()
		if err != nil {
			writeError(w, err)
			return
		}
		if requestedFile == fs.IndexName {
			json.NewEncoder(w).Encode(index)
			return
		}
		if _, ok := index[requestedFile]; ok {
			file, err := reader.Fetch(requestedFile)
			if err != nil {
				writeError(w, err)
				return
			}
			json.NewEncoder(w).Encode(file)
			return
		}
		writeError(w, fmt.Errorf("file not found: %s", requestedFile))
	}
}

func storageReaderConfig() (checkup.StorageReader, error) {
	c := loadCheckup()
	if c.Storage == nil {
		return nil, fmt.Errorf("no storage configuration found")
	}
	prov, ok := c.Storage.(checkup.StorageReader)
	if !ok {
		return nil, fmt.Errorf("configured storage type does not have reading capabilities")
	}
	return prov, nil
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVarP(&listenAddr, "listen", "", ":3000", "The listen address for the HTTP server")
}
