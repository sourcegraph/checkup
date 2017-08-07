package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/sourcegraph/checkup"
	"github.com/spf13/cobra"
)

var configFile string
var storeResults bool
var printLogs bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "checkup",
	Short: "Perform checks on your services and sites",
	Long: `Checkup is distributed, lock-free, self-hosted health
checks and status pages.

Checkup will always look for a checkup.json file in
the current working directory by default and use it.
You can specify a different file location using the
--config/-c flag.

Running checkup without any arguments will invoke
a single checkup and print results to stdout. To
store the results of the check, use --store.`,

	Run: func(cmd *cobra.Command, args []string) {
		if printLogs {
			log.SetOutput(os.Stdout)
		}

		allHealthy := true
		c := loadCheckup()

		if storeResults {
			if c.Storage == nil {
				log.Fatal("no storage configured")
			}
		}

		results, err := c.Check()
		if err != nil {
			log.Fatal(err)
		}

		if storeResults {
			err := c.Storage.Store(results)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		for _, result := range results {
			fmt.Println(result)
			if !result.Healthy {
				allHealthy = false
			}
		}

		if !allHealthy {
			os.Exit(1)
		}
	},
}

func loadCheckup() checkup.Checkup {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	var c checkup.Checkup
	err = json.Unmarshal(configBytes, &c)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "checkup.json", "JSON config file")
	RootCmd.Flags().BoolVar(&storeResults, "store", false, "Store results")
	RootCmd.Flags().BoolVar(&printLogs, "v", false, "Enable logging to standard output")
}
