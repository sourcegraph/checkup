<img src="https://i.imgur.com/UWhSoQj.png" width="450" alt="Checkup">

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/sourcegraph/checkup) [![Sourcegraph](https://sourcegraph.com/github.com/sourcegraph/checkup/-/badge.svg)](https://sourcegraph.com/github.com/sourcegraph/checkup?badge)


**Checkup is distributed, lock-free, self-hosted health checks and status pages, written in Go.**

**It features an elegant, minimalistic CLI and an idiomatic Go library. They are completely interoperable and their configuration is beautifully symmetric.**

Checkup was created by Matt Holt, author of the popular Caddy web
server. It is maintained and sponsored by
[Sourcegraph](https://sourcegraph.com). If you'd like to dive into the
source, you can
[start here](https://sourcegraph.com/github.com/sourcegraph/checkup/-/def/GoPackage/github.com/sourcegraph/checkup/-/Checkup).

This tool is a work-in-progress. Please use liberally with discretion
and report any bugs!


## Intro

Checkup can be customized to check up on any of your sites or services at any time, from any infrastructure, using any storage provider of your choice. The status page can be customized to your liking since you can do your checks however you want.

Out of the box, Checkup currently supports:

- Checking HTTP endpoints
- Checking TCP endpoints (TLS supported)
- Checking of DNS services & record existence  
- Storing results on S3
- Viewing results on a status page that is mobile-responsive and 100% static


## How it Works

There are 3 components:

1. **Storage** You set up storage space for the results of the checks.

2. **Checks** You run checks on whatever endpoints you have as often as you want.

3. **Status Page** You host the status page. [Caddy](https://caddyserver.com) makes this super easy. The status page downloads recent check files from storage and renders the results client-side.


## Quick Start

```bash
$ checkup --help
```

Follow these instructions to get started quickly with Checkup.


### Create your Checkup config

You can configure Checkup entirely with a simple JSON document. We recommend you configure storage and at least one checker:

```json
{
	"checkers": [{
		"type": "http",
		"endpoint_name": "Example HTTP",
		"endpoint_url": "http://www.example.com",
		"attempts": 5
	},
	{
		"type": "tcp",
		"endpoint_name": "Example TCP",
		"endpoint_url": "example.com:80",
		"attempts": 5
	},
	{
		"type": "tcp",
		"endpoint_name": "Example TCP with TLS enabled and a valid certificate chain",
		"endpoint_url": "example.com:443",
		"attempts": 5,
		"tls": true
	},
	{
		"type": "tcp",
		"endpoint_name": "Example TCP with TLS enabled and a self-signed certificate chain",
		"endpoint_url": "example.com:8443",
		"attempts": 5,
		"timeout": "2s",
		"tls": true,
		"tls_ca_file": "testdata/ca.pem"
	},
	{
		"type": "tcp",
		"endpoint_name": "Example TCP with TLS enabled and verification disabled",
		"endpoint_url": "example.com:8443",
		"attempts": 5,
		"timeout": "2s",
		"tls": true,
		"tls_skip_verify": true
	},
	{
		"type": "dns",
		"endpoint_name": "Example DNS test of endpoint_url looking up host.example.com",
		"endpoint_url": "ns.example.com:53",
		"hostname_fqdn": "host.example.com",
		"timeout": "2s"
	}],
	"storage": {
		"provider": "s3",
		"access_key_id": "<yours>",
		"secret_access_key": "<yours>",
		"bucket": "<yours>",
		"region": "us-east-1"
	}
}
```

**For the complete structure definition, please see [the godoc](https://godoc.org/github.com/sourcegraph/checkup).** There are many elements of checkers and storage you may wish to customize!

Save this file as `checkup.json` in your working directory.




### Setting up storage on S3

The easiest way to do this is to give an IAM user these two privileges (keep the credentials secret):

- arn:aws:iam::aws:policy/**IAMFullAccess**
- arn:aws:iam::aws:policy/**AmazonS3FullAccess**

#### Implicit Provisioning

If you give these permissions to the same user as with the credentials in your JSON config above, then you can simply run:

```bash
$ checkup provision
```

and checkup will read the config file and provision S3 for you. If the user is different, you may want to use explicit provisioning instead.

This command creates a new IAM user with read-only permission to S3 and also creates a new bucket just for your check files. The credentials of the new user are printed to your screen. **Make note of the Public Access Key ID and Public Access Key!** You won't be able to see them again.

**IMPORTANT SECURITY NOTE:** This new IAM user will have read-only permission to all S3 buckets in your AWS account, and its credentials will be visible to any visitor to your status page. If you do not want to grant visitors to your status page read access to all your S3 buckets, you need to modify this IAM user's permissions to scope its access to the Checkup bucket. If in doubt, restrict access to your status page to trusted visitors. It is recommended that you do NOT include ANY sensitive credentials on the machine running Checkup.


#### Explicit Provisioning

If you do not prefer implicit provisioning using your `checkup.json` file, do this instead. Export the information to environment variables and run the provisioning command:

```bash
$ export AWS_ACCESS_KEY_ID=...
$ export AWS_SECRET_ACCESS_KEY=...
$ export AWS_BUCKET_NAME=...
$ checkup provision s3
```

#### Manual Provisioning

If you'd rather do this manually, see the [instructions on the wiki](https://github.com/sourcegraph/checkup/wiki/Provisioning-S3-Manually) but keeping in mind the region must be **US Standard**.

### Setting up the status page

In statuspage/js, use the contents of [config_template.js](https://github.com/sourcegraph/checkup/blob/master/statuspage/js/config_template.js) to fill out [config.js](https://github.com/sourcegraph/checkup/blob/master/statuspage/js/config.js), which is used by the status page. This is where you put the *read-only* S3 credentials you just generated.

Then, the status page can be served over HTTPS by running `caddy -host status.mysite.com` on the command line. (You can use [getcaddy.com](https://getcaddy.com) to install Caddy.)

As you perform checks, the status page will update every so often with the latest results. **Only checks that are stored will appear on the status page.**


### Performing checks

You can run checks many different ways: cron, AWS Lambda, or a time.Ticker in your own Go program, to name a few. Checks should be run on a regular basis. How often you run checks depends on your requirements and how much time you render on the status page. 

For example, if you run checks every 10 minutes, showing the last 24 hours on the status page will require 144 check files to be downloaded on each page load. You can distribute your checks to help avoid localized network problems, but this multiplies the number of files by the number of nodes you run checks on, so keep that in mind.

Performing checks with the `checkup` command is very easy.

Just `cd` to the folder with your `checkup.json` from earlier, and checkup will automatically use it:

```bash
$ checkup
```

The vanilla checkup command runs a single check and prints the results to your screen, but does not save them to storage for your status page.

To store the results instead, use `--store`:

```bash
$ checkup --store
```

If you want Checkup to loop forever and perform checks and store them on a regular interval, use this:

```bash
$ checkup every 10m
```

And replace the duration with your own preference. In addition to the regular `time.ParseDuration()` formats, you can use shortcuts like `second`, `minute`, `hour`, `day`, or `week`.

You can also get some help using the `-h` option for any command or subcommand.


### Posting status messages

Site reliability engineers should post messages when there are incidents or other news relevant for a status page. This is also very easy:

```bash
$ checkup message --about=Example "Oops. We're trying to fix the problem. Stay tuned."
```

This stores a check file with your message attached to the result for "Example" which you configured in `checkup.json` earlier.




## Doing all that, but with Go

Checkup is as easy to use in a Go program as it is on the command line.


### Using Go to set up storage on S3

First, create an IAM user with credentials as described in the section above.

Then `go get github.com/sourcegraph/checkup` and import it.

Then replace `ACCESS_KEY_ID` and `SECRET_ACCESS_KEY` below with the actual values for that user. Keep those secret. You'll also replace `BUCKET_NAME` with the unique bucket name to store your check files:

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
fmt.Println(info) // don't lose this output!
```

This method creates a new IAM user with read-only permission to S3 and also creates a new bucket just for your check files. The credentials of the new user are printed to your screen. **Make note of the PublicAccessKeyID and PublicAccessKey!** You won't be able to see them again.



### Using Go to perform checks

First, `go get github.com/sourcegraph/checkup` and import it. Then configure it:

```go
c := checkup.Checkup{
	Checkers: []checkup.Checker{
		checkup.HTTPChecker{Name: "Example (HTTP)", URL: "http://www.example.com", Attempts: 5},
		checkup.HTTPChecker{Name: "Example (HTTPS)", URL: "https://www.example.com", Attempts: 5},
		checkup.TCPChecker{Name:  "Example (TCP)", URL:  "www.example.com:80", Attempts: 5},
		checkup.TCPChecker{Name:  "Example (TCP SSL)", URL:  "www.example.com:443", Attempts: 5, TLSEnabled: true},
		checkup.TCPChecker{Name:  "Example (TCP SSL, self-signed certificate)", URL:  "www.example.com:443", Attempts: 5, TLSEnabled: true, TLSCAFile: "testdata/ca.pem"},
		checkup.TCPChecker{Name:  "Example (TCP SSL, validation disabled)", URL:  "www.example.com:8443", Attempts: 5, TLSEnabled: true, TLSSkipVerify: true},
		checkup.DNSChecker{Name:  "Example DNS test of ns.example.com:53 looking up host.example.com", URL:  "ns.example.com:53", Host: "host.example.com", Attempts: 5},
	},
	Storage: checkup.S3{
		AccessKeyID:     "<yours>",
		SecretAccessKey: "<yours>",
		Bucket:          "<yours>",
		Region:          "us-east-1",
		CheckExpiry:     24 * time.Hour * 7,
	},
}
```

This sample checks 2 endpoints (HTTP and HTTPS). Each check consists of 5 attempts so as to smooth out the final results a bit. We will store results on S3. Notice the `CheckExpiry` value. The `checkup.S3` type is also `checkup.Maintainer` type, which means it can maintain itself and purge any status checks older than `CheckExpiry`. We chose 7 days.

Then, to run checks every 10 minutes:

```go
c.CheckAndStoreEvery(10 * time.Minute)
select {}
```

`CheckAndStoreEvery()` returns a `time.Ticker` that you can stop, but in this case we just want it to run forever, so we block forever using an empty `select`.


### Using Go to post status messages

Simply perform a check, add the message to the corresponding result, and then store it:

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


## Other topics

### Getting notified when there are problems

Uh oh, having some fires? 🔥 You can create a type that implements `checkup.Notifier`. Checkup will invoke `Notify()` after every check, where you can evaluate the results and decide if and how you want to send a notification or trigger some event.

### Other kinds of checks or storage providers

Need to check more than HTTP? S3 too Amazony for you? You can implement your own Checker and Storage types. If it's general enough, feel free to submit a pull request so others can use it too!

### Building with Docker

Linux binary:

```bash
git clone git@github.com:sourcegraph/checkup.git
cd checkup
docker run --rm \ 
-v `pwd`:/project \ 
-w /project golang bash \ 
-c "cd cmd/checkup; go get -v -d; go build -v -ldflags '-s' -o ../../checkup"
```

This will create a checkup binary in the root project folder.
