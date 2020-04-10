package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sourcegraph/checkup"
	"github.com/sourcegraph/checkup/storage/s3"
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision a storage service",
	Long: `Use the provision command to provision storage for your
check files on any supported provider. Provisioning need 
only be done once per status page. After provisioning,
you will be provided some credentials; use those to
configure your status page and/or checker. 

By default, checkup.json will be loaded and used, if it 
exists in the current working directory. Otherwise, you
may provision your storage manually according to the
instructions below.

To do it manually, run 'checkup provision <provider>'
with your provider of choice after setting the required
environment variables.

PROVIDERS

s3
    Create an IAM user with at least these two permissions:
    
       - arn:aws:iam::aws:policy/IAMFullAccess
       - arn:aws:iam::aws:policy/AmazonS3FullAccess

    Then set these env variables:

       - AWS_ACCESS_KEY_ID=<AccessKeyID of user> 
       - AWS_SECRET_ACCESS_KEY=<SecretAccessKey of user>
       - AWS_BUCKET_NAME=<unique bucket name>`,
	Run: func(cmd *cobra.Command, args []string) {
		var prov checkup.Provisioner
		var err error

		switch len(args) {
		case 0:
			prov, err = provisionerConfig()
		case 1:
			prov, err = provisionerEnvVars(cmd, args)
		default:
			fmt.Println(cmd.Long)
			os.Exit(1)
		}
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("One sec...")

		info, err := prov.Provision()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(info)
	},
}

func provisionerConfig() (checkup.Provisioner, error) {
	c := loadCheckup()
	if c.Storage == nil {
		return nil, fmt.Errorf("no storage configuration found")
	}
	prov, ok := c.Storage.(checkup.Provisioner)
	if !ok {
		return nil, fmt.Errorf("configured storage type does not have provisioning capabilities")
	}
	return prov, nil
}

func provisionerEnvVars(cmd *cobra.Command, args []string) (checkup.Provisioner, error) {
	providerName := strings.ToLower(args[0])
	switch providerName {
	case "s3":
		keyID := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		bucket := os.Getenv("AWS_BUCKET_NAME")
		if keyID == "" || secretKey == "" || bucket == "" {
			fmt.Println(cmd.Long)
			os.Exit(1)
		}
		return s3.Storage{
			AccessKeyID:     keyID,
			SecretAccessKey: secretKey,
			Bucket:          bucket,
		}, nil
	default:
		return nil, fmt.Errorf("unknown storage provider '%s'", providerName)
	}
}

func init() {
	RootCmd.AddCommand(provisionCmd)
}
