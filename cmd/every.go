package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var everyCmd = &cobra.Command{
	Use:   "every",
	Short: "Run checks indefinitely at an interval",
	Long: `The every subcommand runs checkups at the interval you
specify. The result of each check is saved to storage.
Additionally, if a Notifier is configured, it will be
called to analyze and potentially notify you of any
problems.

This command never unblocks, so you must signal the
program to exit.

Interval formats are the same as those for Go's
time.ParseDuration() syntax:
https://golang.org/pkg/time/#ParseDuration - with a
few shortcuts: second, minute, hour, day, and week.

Examples:

  $ checkup every 10m
  $ checkup every day
  $ checkup every 1h30m`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println(cmd.Long)
			os.Exit(1)
		}

		itvlStr := strings.ToLower(args[0])
		switch itvlStr {
		case "second":
			itvlStr = "1s"
		case "minute":
			itvlStr = "1m"
		case "hour":
			itvlStr = "1h"
		case "day":
			itvlStr = "24h"
		case "week":
			itvlStr = "168h"
		}

		interval, err := time.ParseDuration(itvlStr)
		if err != nil {
			log.Fatal(err)
		}

		c := loadCheckup()
		if len(c.Checkers) == 0 {
			log.Fatal("no checkers configured")
		}
		if c.Storage == nil {
			log.Fatal("no storage configured")
		}

		c.CheckAndStoreEvery(interval)
		select {}
	},
}

func init() {
	RootCmd.AddCommand(everyCmd)
}
