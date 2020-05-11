package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var about string

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Post a status message/update",
	Long: `The message subcommand allows you to post updates to
to your status page for a certain endpoint. This is
helpful (and responsible of you) when your service is
experiencing a disruption or you are starting planned
maintenance.

Checkup will always report the facts, even if the
disruption is planned. You can use status messages to
give clarity and transparency to your customers
or visitors.

If your checkup configuration specifies more than one
endpoint, you must use the --about flag to specify
the name of the endpoint as defined in your config
file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println(cmd.Long)
			os.Exit(1)
		}

		contents := args[0]

		c := loadCheckup()
		if c.Storage == nil {
			log.Fatal("no storage configured")
		}

		results, err := c.Check()
		if err != nil {
			log.Fatal(err)
		}

		if len(results) > 1 && about == "" {
			log.Fatal("more than one result; unable to guess which one to attach message to")
		}

		if len(results) == 1 && about == "" {
			results[0].Message = contents
		} else {
			found := false
			lowerAbout := strings.ToLower(about)
			for i, result := range results {
				if strings.ToLower(result.Title) == lowerAbout {
					results[i].Message = contents
					found = true
					break
				}
			}
			if !found {
				log.Fatalf("no result for endpoint with title '%s'", about)
			}
		}

		err = c.Storage.Store(results)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Message posted")
	},
}

func init() {
	RootCmd.AddCommand(messageCmd)
	messageCmd.Flags().StringVarP(&about, "about", "a", "", "The name/title of the endpoint this message is about")
}
