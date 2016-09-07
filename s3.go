package checkup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3 is a way to store checkup results in an S3 bucket.
type S3 struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region,omitempty"`
	Bucket          string `json:"bucket"`

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration `json:"check_expiry,omitempty"`
}

// Store stores results on S3 according to the configuration in s.
func (s S3) Store(results []Result) error {
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return err
	}
	svc := newS3(session.New(), &aws.Config{
		Credentials: credentials.NewStaticCredentials(s.AccessKeyID, s.SecretAccessKey, ""),
		Region:      &s.Region,
	})
	params := &s3.PutObjectInput{
		Bucket: &s.Bucket,
		Key:    GenerateFilename(),
		Body:   bytes.NewReader(jsonBytes),
	}
	_, err = svc.PutObject(params)
	return err
}

// Maintain deletes check files that are older than s.CheckExpiry.
func (s S3) Maintain() error {
	if s.CheckExpiry == 0 {
		return nil
	}

	svc := newS3(session.New(), &aws.Config{
		Credentials: credentials.NewStaticCredentials(s.AccessKeyID, s.SecretAccessKey, ""),
		Region:      &s.Region,
	})

	var marker *string
	for {
		listParams := &s3.ListObjectsInput{
			Bucket: &s.Bucket,
			Marker: marker,
		}
		listResp, err := svc.ListObjects(listParams)
		if err != nil {
			return err
		}

		var objsToDelete []*s3.ObjectIdentifier
		for _, o := range listResp.Contents {
			if o == nil || o.LastModified == nil {
				continue
			}
			if time.Since(*o.LastModified) > s.CheckExpiry {
				objsToDelete = append(objsToDelete, &s3.ObjectIdentifier{Key: o.Key})
			}
		}

		if len(objsToDelete) == 0 {
			break
		}

		delParams := &s3.DeleteObjectsInput{
			Bucket: &s.Bucket,
			Delete: &s3.Delete{
				Objects: objsToDelete,
				Quiet:   aws.Bool(true),
			},
		}

		_, err = svc.DeleteObjects(delParams)
		if err != nil {
			return err
		}

		if !*listResp.IsTruncated {
			break
		}

		marker = listResp.Contents[len(listResp.Contents)-1].Key
	}

	return nil
}

// Provision creates a new IAM user in the account specified
// by s, and configures a bucket according to the values in
// s. The credentials in s must have the IAMFullAccess and
// AmazonS3FullAccess permissions in order to succeed.
//
// The name of the created IAM user is "checkup-monitor-s3-public".
// It will have read-only permission to S3.
//
// Provision need only be called once per status page (bucket),
// not once per endpoint.
func (s S3) Provision() (ProvisionInfo, error) {
	const iamUser = "checkup-monitor-s3-public"
	var info ProvisionInfo

	// default region (required, but regions don't apply to S3, kinda weird)
	if s.Region == "" {
		s.Region = "us-east-1"
	}

	svcIam := iam.New(session.New(), &aws.Config{
		Credentials: credentials.NewStaticCredentials(s.AccessKeyID, s.SecretAccessKey, ""),
		Region:      &s.Region,
	})

	// Create a new user, just for reading the check files
	resp, err := svcIam.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(iamUser),
	})
	if err != nil {
		return info, fmt.Errorf("Error creating user: %s\n\nTry deleting the user in the AWS control panel and try again.", err)
	}
	info.Username = *resp.User.UserName
	info.UserID = *resp.User.UserId

	// Restrict the user to only reading S3 buckets
	_, err = svcIam.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"),
		UserName:  aws.String(iamUser),
	})
	if err != nil {
		return info, err
	}

	// Give the user a key (this will become public as it is read-only)
	resp3, err := svcIam.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(iamUser),
	})
	if err != nil {
		return info, err
	}
	info.PublicAccessKeyID = *resp3.AccessKey.AccessKeyId
	info.PublicAccessKey = *resp3.AccessKey.SecretAccessKey

	// Prepare to talk to S3
	svcS3 := s3.New(session.New(), &aws.Config{
		Credentials: credentials.NewStaticCredentials(s.AccessKeyID, s.SecretAccessKey, ""),
		Region:      &s.Region,
	})

	// Create a bucket to hold all the checks
	_, err = svcS3.CreateBucket(&s3.CreateBucketInput{
		Bucket: &s.Bucket,
	})
	if err != nil {
		return info, err
	}

	// Configure its CORS policy to allow reading from status pages
	_, err = svcS3.PutBucketCors(&s3.PutBucketCorsInput{
		Bucket: &s.Bucket,
		CORSConfiguration: &s3.CORSConfiguration{
			CORSRules: []*s3.CORSRule{
				{
					AllowedOrigins: []*string{aws.String("*")},
					AllowedMethods: []*string{aws.String("GET"), aws.String("HEAD")},
					ExposeHeaders:  []*string{aws.String("ETag")},
					AllowedHeaders: []*string{aws.String("*")},
					MaxAgeSeconds:  aws.Int64(3000),
				},
			},
		},
	})
	if err != nil {
		return info, err
	}

	// Set its policy to allow getting objects
	_, err = svcS3.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: &s.Bucket,
		Policy: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Sid": "PublicReadGetObject",
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + s.Bucket + `/*"
				}
			]
		}`),
	})
	if err != nil {
		return info, err
	}

	return info, nil
}

// newS3 calls s3.New(), but may be replaced for mocking in tests.
var newS3 = func(p client.ConfigProvider, cfgs ...*aws.Config) s3svc {
	return s3.New(p, cfgs...)
}

// s3svc is used for mocking the s3.S3 type.
type s3svc interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
	ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error)
}
