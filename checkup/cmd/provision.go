package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sourcegraph/checkup"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision a storage service",
	Long: `Use the provision command to provision storage for your
check files on any supported provider. Provisioning relies
on environment variables for settings and credentials,
and only needs to be done once per status page.

Run 'checkup provision <provider>' with your provider of
choice after setting the required environment variables.

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
		if len(args) != 1 {
			fmt.Println(cmd.Long)
			os.Exit(1)
		}
		providerName := strings.ToLower(args[0])

		var prov checkup.Provisioner
		switch providerName {
		case "s3":
			keyID := os.Getenv("AWS_ACCESS_KEY_ID")
			secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
			bucket := os.Getenv("AWS_BUCKET_NAME")
			if keyID == "" || secretKey == "" || bucket == "" {
				fmt.Println(cmd.Long)
				os.Exit(1)
			}

			prov = checkup.S3{
				AccessKeyID:     keyID,
				SecretAccessKey: secretKey,
				Bucket:          bucket,
			}
		default:
			log.Fatal("unknown storage provider " + providerName)
		}

		fmt.Println("One sec...")

		info, err := prov.Provision()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Successfully provisioned %s\n\n", providerName)
		fmt.Printf("             User ID: %s\n", info.UserID)
		fmt.Printf("            Username: %s\n", info.Username)
		fmt.Printf("Public Access Key ID: %s\n", info.PublicAccessKeyID)
		fmt.Printf("   Public Access Key: %s\n\n", info.PublicAccessKey)

		fmt.Println(`IMPORTANT: Copy the Public Access Key ID and Public Access
Key into the config.js file for your status page. You will
not be shown these credentials again.`)
	},
}

func init() {
	RootCmd.AddCommand(provisionCmd)
}
