package checkup

import (
	"bytes"
	"encoding/json"

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

// newS3 calls s3.New(), but may be replaced for mocking in tests.
var newS3 = func(p client.ConfigProvider, cfgs ...*aws.Config) objectPutter {
	return s3.New(p, cfgs...)
}

// objectPutter is used for mocking the s3.S3 type.
type objectPutter interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}
