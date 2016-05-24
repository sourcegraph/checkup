<img src="https://i.imgur.com/UWhSoQj.png" width="450" alt="Checkup">

Checkup is distributed, lock-free, self-hosted health checks and status pages, written in Go.

**Note: This is a _work in progress_ and is still in the prototype phase. Don't use it for anything too important, but _do_ still use it, and report any bugzies!**


## Intro

Checkup can be customized to check up on any of your sites or services at any time, from any infrastructure, using any storage provider of your choice. The status page can be customized to your liking since you can do your checks however you want.

Out of the box, Checkup currently supports:

- Checking HTTP endpoints
- Storing results on S3
- Viewing results on a status page


## How it Works

There are 3 components:

1. **Storage** You set up storage space for the results of the checks.

2. **Checks** You run checks whenever you want (usually on a regular basis), to whatever endpoints you want.

3. **Status Page** You host the status page. [Caddy](https://caddyserver.com) makes this super easy. The status page downloads recent check files from storage and renders the results client-side.


## Quick Start

Follow these instructions to get started quickly with Checkup.


### Setting up storage on S3

The easiest way to do this is with a few lines of Go code. (If you'd rather do it manually, see the [instructions on the wiki](https://github.com/sourcegraph/checkup/wiki/Provisioning-S3-Manually).)

First you'll need an IAM user with at least two permissions:

- arn:aws:iam::aws:policy/**IAMFullAccess**
- arn:aws:iam::aws:policy/**AmazonS3FullAccess**

Then replace `ACCESS_KEY_ID` and `SECRET_ACCESS_KEY` below with the actual values for that user. You'll also replace `BUCKET_NAME` with the unique bucket name to store your check files:

```go
storage := checkup.S3{
	AccessKeyID:     "ACCESS_KEY_ID",
	SecretAccessKey: "SECRET_ACCESS_KEY",
	Bucket:          "BUCKET_NAME",
}
info, err := storage.Provision()
if err != nil {
	log.Fatal(err)
}
fmt.Printf("%+v\n", info)
```

This method creates a new IAM user with read-only permission to S3 and also creates a new bucket just for your check files.

**Take note of the output, since you won't see it again!** Especially important are the `PublicAccessKeyID` and the `PublicAccessKey`. They will be used by the status page to load check files.


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
