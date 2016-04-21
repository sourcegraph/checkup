Checkup
========

Checkup is distributed, lock-free, self-hosted health checks and status pages, written in Go.

**Note: This is a _work in progress_ and is still in the prototype phase. Don't use it for anything too  important, but _do_ still use it, and report any bugzies!**


## Intro

Checkup can be customized to check up on any of your sites or services at any time, from any infrastructure, using any storage provider of your choice. The status page can be customized to your liking since you can do your checks however you want.

Out of the box, Checkup currently supports:

- Checking HTTP endpoints
- Storing results on S3


## How it Works

There are 3 components:

1. **Storage** You set up storage space for the results of the checks.

2. **Checks** You run checks whenever you want (usually on a regular basis).

3. **Status Page** You host the status page. [Caddy](https://caddyserver.com) makes this super easy. The status page downloads recent check files from storage and renders the results client-side.


## Quick Start

*TODO(mholt): Make setting up an S3 bucket easier (Provisioner interface?)*

Follow these instructions to get started quickly with Checkup.


### Setting up storage on S3

1. Create two IAM users and groups. One user and group has `AmazonS3FullAccess`. The other user and group has `AmazonS3ReadOnlyAccess`. (The credentials for the ReadOnlyAccess will be made public...)

2. Create an S3 bucket with the following CORS configuration:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <CORSRule>
        <AllowedOrigin>*</AllowedOrigin>
        <AllowedMethod>GET</AllowedMethod>
        <AllowedMethod>HEAD</AllowedMethod>
        <MaxAgeSeconds>3000</MaxAgeSeconds>
        <ExposeHeader>ETag</ExposeHeader>
        <AllowedHeader>*</AllowedHeader>
    </CORSRule>
</CORSConfiguration>
```

3. Give the bucket this policy (replace BUCKET_NAME):
```json
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "PublicReadGetObject",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::BUCKET_NAME/*"
		}
	]
}
```

This bucket must be used exclusively for checks for this status page.


### Setting up checks

You can run checks many different ways: cron, AWS Lambda, or a time.Ticker in your Go program, to name a few. Checks should be run on a regular basis. How often you run checks depends on your requirements and how much time you render on the status page. 

For example, if you run checks every 10 minutes, showing the last 24 hours on the status page will require 144 files to be downloaded on each page load. You can distribute your checks to help avoid localized network problems, but this multiplies the number of files by the number of nodes you run checks on, so keep that in mind.

Checks are configured in Go. First, get the package:

```bash
$ go get github.com/sourcegraph/checkup
```

Then import it:

```go
import "github.com/sourcegraph/checkup"
```

Then configure it:

```go
c := checkup.Checkup{
	Checkers: []checkup.Checker{
		checkup.HTTPChecker{Name: "Example (HTTP)", URL: "http://www.example.com", Attempts: 5},
		checkup.HTTPChecker{Name: "Example (HTTPS)", URL: "https://example.com", Attempts: 5},
	},
	Storage: checkup.S3{
		AccessKeyID:     "<yours>",
		SecretAccessKey: "<yours>",
		Region:          "us-east-1",
		Bucket:          "<yours>",
		CheckExpiry:     24 * time.Hour * 7,
	},
}
```

This sample checks 2 endpoints (HTTP and HTTPS). Each check consists of 5 attempts so as to smooth out the final results a bit. We will store results on S3. Notice the `CheckExpiry` value. The `checkup.S3` type is also `checkup.Maintainer` type, which means it can maintain itself and purge any status checks older than `CheckExpiry`. We chose 7 days.

Then, to run checks every 10 minutes:

```go
wait := make(chan struct{})
c.CheckAndStoreEvery(10 * time.Minute)
<-wait
```

The channel is only used to block forever, but your actual use case may be different. `CheckAndStoreEvery()` returns a `time.Ticker` that you can stop, but in this case we just want it to run forever.

Great! With regular checkups happening, we can now serve our status page.


### Setting up the status page

*TODO(mholt): Make the status page more easily configurable. Right now, values have to be edited at the top of the statuspage.js file.*

Once configured, the status page can be served over HTTPS by running `caddy -host status.mysite.com`.



## Posting status messages

Site reliability engineers should post messages when there are incidents or other news relevant for a status page. All you have to do is run a check, then add a message to the result before storing it:

```go
results, err := c.Check()
if err != nil {
	// handle err
}

results[0].Message = "We're investigating connectivity issues."

err = c.Storage.Store(results)
if err != nil {
	// handle err
}
```

Of course, real status messages should be as descriptive as possible. You can use HTML in them.

*TODO(mholt): Status messages should be allowed to be long (for post-mortems) and the status page should handle that gracefully by collapsing long messages.*

