package checkup

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3 is a way to store checkup results in an S3 bucket.
type S3 struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string

	// Check files older than CheckExpiry will be
	// deleted on calls to Maintain(). If this is
	// the zero value, no old check files will be
	// deleted.
	CheckExpiry time.Duration
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
