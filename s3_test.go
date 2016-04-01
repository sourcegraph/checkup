package checkup

import (
	"bytes"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestS3Store(t *testing.T) {
	keyID, accessKey, region, bucket := "fakeKeyID", "fakeKey", "fakeRegion", "fakeBucket"
	fakes3 := new(s3Mock)
	results := []Result{{Title: "Testing"}}
	resultsBytes := []byte(`[{"title":"Testing","down":false}]`)
	newS3 = func(p client.ConfigProvider, cfgs ...*aws.Config) objectPutter {
		if len(cfgs) != 1 {
			t.Fatalf("Expected 1 aws.Config, got %d", len(cfgs))
		}
		creds, err := cfgs[0].Credentials.Get()
		if err != nil {
			t.Fatalf("Got an error when calling Get() on Credentials: %v", err)
		}
		if got, want := creds.AccessKeyID, keyID; got != want {
			t.Errorf("Expected AccessKeyID to be '%s', got '%s'", want, got)
		}
		if got, want := creds.SecretAccessKey, accessKey; got != want {
			t.Errorf("Expected SecretAccessKey to be '%s', got '%s'", want, got)
		}
		if got, want := *cfgs[0].Region, region; got != want {
			t.Errorf("Expected Region to be '%s', got '%s'", want, got)
		}
		return fakes3
	}

	specimen := S3{
		AccessKeyID:     keyID,
		SecretAccessKey: accessKey,
		Region:          region,
		Bucket:          bucket,
	}
	err := specimen.Store(results)
	if err != nil {
		t.Fatalf("Expected no error from Store(), got: %v", err)
	}

	// Make sure bucket name is right
	if got, want := *fakes3.input.Bucket, bucket; got != want {
		t.Errorf("Expected Bucket to be '%s', got '%s'", want, got)
	}

	// Make sure filename has timestamp of check
	key := *fakes3.input.Key
	hyphenPos := strings.Index(key, "-")
	if hyphenPos < 0 {
		t.Fatalf("Expected Key to have timestamp then hyphen, got: %s", key)
	}
	tsString := key[:hyphenPos]
	tsNs, err := strconv.ParseInt(tsString, 10, 64)
	if err != nil {
		t.Fatalf("Expected Key's timestamp to be integer; got error: %v", err)
	}
	ts := time.Unix(0, tsNs)
	if time.Since(ts) > 1*time.Second {
		t.Errorf("Timestamp of filename is %s but expected something very recent", ts)
	}

	// Make sure body bytes are correct
	bodyBytes, err := ioutil.ReadAll(fakes3.input.Body)
	if err != nil {
		t.Fatalf("Expected no error reading body, got: %v", err)
	}
	if bytes.Compare(bodyBytes, resultsBytes) != 0 {
		t.Errorf("Contents of file are wrong\nExpected %s\n     Got %s", resultsBytes, bodyBytes)
	}
}

// s3Mock mocks s3.S3.
type s3Mock struct {
	input *s3.PutObjectInput
}

func (s *s3Mock) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	s.input = input
	return nil, nil
}
