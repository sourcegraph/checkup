<img src="https://i.imgur.com/UWhSoQj.png" width="450" alt="Checkup">

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/sourcegraph/checkup) [![Sourcegraph](https://sourcegraph.com/github.com/sourcegraph/checkup/-/badge.svg)](https://sourcegraph.com/github.com/sourcegraph/checkup?badge)


**Checkup is distributed, lock-free, self-hosted health checks and status pages, written in Go.**

**It features an elegant, minimalistic CLI and an idiomatic Go library. They are completely interoperable and their configuration is beautifully symmetric.**

Checkup was created by Matt Holt, author of the [Caddy web server](https://caddyserver.com). It is maintained and sponsored by [Sourcegraph](https://sourcegraph.com). If you'd like to dive into the source, you can [start here](https://sourcegraph.com/github.com/sourcegraph/checkup/-/def/GoPackage/github.com/sourcegraph/checkup/-/Checkup).

This tool is a work-in-progress. Please use liberally (with discretion) and report any bugs!

## Recent changes

Due to recent development, some breaking changes have been introduced:

- providers: the json config field `provider` was renamed to `type` for consistency,
- notifiers: the json config field `name` was renamed to `type` for consistency,
- sql: by default the sqlite storage engine is disabled (needs build with `-tags sql` to enable),
- sql: storage engine is deprecated in favor of new storage engines postgres, mysql, sqlite3
- mailgun: the `to` parameter now takes a list of e-mail addresses (was a single recipient)
- LOGGING IS NOT SWALLOWED ANYMORE, DON'T PARSE `checkup` OUTPUT IN SCRIPTS
- default for status page config has been set to local source (use with `checkup serve`)

If you want to build the latest version, it's best to run:

- `make build` - builds checkup with mysql and postgresql support,
- `make build-sqlite3` - builds checkup with additional sqlite3 support

The resulting binary will be placed into `builds/checkup`.

## Intro

Checkup can be customized to check up on any of your sites or services at any time, from any infrastructure, using any storage provider of your choice (assuming an integration exists for your storage provider). The status page can be customized to your liking since you can do your checks however you want. The status page is also mobile-responsive.

Checkup currently supports these checkers:

- HTTP
- TCP (+TLS)
- DNS
- TLS

Checkup implements these storage providers:

- Amazon S3
- Local file system
- GitHub
- MySQL
- PostgreSQL
- SQLite3
- Azure Application Insights

*Currently the status page does not support SQL or Azure Application Insights storage back-ends.*

Checkup can even send notifications through your service of choice (if an integration exists).


## How it Works

There are 3 components:

1. **Storage.** You set up storage space for the results of the checks.
2. **Checks.** You run checks on whatever endpoints you have as often as you want.
3. **Status Page.** You (or GitHub) host the status page.


## Quick Start

[Download Checkup](https://github.com/sourcegraph/checkup/releases/latest) for your platform and put it in your PATH, or install from source:

```bash
$ go get -u github.com/sourcegraph/checkup/cmd/checkup
```

You'll need Go 1.8 or newer. Verify it's installed properly:

```bash
$ checkup --help
```

Then follow these instructions to get started quickly with Checkup.


### Create your Checkup config

You can configure Checkup entirely with a simple JSON document. You should configure storage and at least one checker. Here's the basic outline:

```js
{
    "checkers": [
        // checker configurations go here
    ],

    "storage": {
        // storage configuration goes here
    },

    "notifiers": [
        // notifier configuration goes here
    ]
}
```

Save the checkup configuration file as `checkup.json` in your working directory.

We will show JSON samples below, to get you started. **But please [refer to the godoc](https://godoc.org/github.com/sourcegraph/checkup) for a comprehensive description of each type of checker, storage, and notifier you can configure!**

Here are the configuration structures you can use, which are explained fully [in the godoc](https://godoc.org/github.com/sourcegraph/checkup). **Only the required fields are shown, so consult the godoc for more.**

#### HTTP Checker

**[godoc: check/http](https://godoc.org/github.com/sourcegraph/checkup/check/http)**

```js
{
    "type": "http",
    "endpoint_name": "Example HTTP",
    "endpoint_url": "http://www.example.com"
    // for more fields, see the godoc
}
```


#### TCP Checker

**[godoc: check/tcp](https://godoc.org/github.com/sourcegraph/checkup/check/tcp)**

```js
{
    "type": "tcp",
    "endpoint_name": "Example TCP",
    "endpoint_url": "example.com:80"
}
```

#### DNS Checkers

**[godoc: check/dns](https://godoc.org/github.com/sourcegraph/checkup/check/dns)**

```js
{
    "type": "dns",
    "endpoint_name": "Example of endpoint_url looking up host.example.com",
    "endpoint_url": "ns.example.com:53",
    "hostname_fqdn": "host.example.com"
}
```

#### TLS Checkers

**[godoc: check/tls](https://godoc.org/github.com/sourcegraph/checkup/check/tls)**

```js
{
    "type": "tls",
    "endpoint_name": "Example TLS Protocol Check",
    "endpoint_url": "www.example.com:443"
}
```

#### Exec Checkers

**[godoc: check/exec](https://godoc.org/github.com/sourcegraph/checkup/check/exec)**

The exec checker can run any command, and expects an zero-value exit code
on success. Non-zero exit codes are considered errors. You can configure
the check with `"raise":"warning"` if you want to consider a failing
service as DEGRADED. Additional options available on godoc link above.

```js
{
    "type": "exec",
    "name": "Example Exec Check",
    "command": "testdata/exec.sh"
}
```

#### Amazon S3 Storage

**[godoc: S3](https://godoc.org/github.com/sourcegraph/checkup/check/s3)**

```js
{
    "type": "s3",
    "access_key_id": "<yours>",
    "secret_access_key": "<yours>",
    "bucket": "<yours>",
    "region": "us-east-1"
}
```

To serve files for your status page from S3, copy `statuspage/config_s3.js` over `statuspage/config.js`, and fill out the required public, read-only credentials.

#### File System Storage

**[godoc: FS](https://godoc.org/github.com/sourcegraph/checkup/storage/fs)**

```js
{
    "type": "fs",
    "dir": "/path/to/your/check_files"
}
```

#### GitHub Storage

**[godoc: GitHub](https://godoc.org/github.com/sourcegraph/checkup/storage/github)**

```js
{
    "type": "github",
    "access_token": "some_api_access_token_with_repo_scope",
    "repository_owner": "owner",
    "repository_name": "repo",
    "committer_name": "Commiter Name",
    "committer_email": "you@example.com",
    "branch": "gh-pages",
    "dir": "updates"
}
```

Where "dir" is a subdirectory within the repo to push all the check files. Setup instructions:

1. Create a repository,
2. Copy the contents of `statuspage/` from this repo to the root of your new repo,
3. Update the URL in `config.js` to `https://your-username.github.com/dir/`,
4. Create `updates/.gitkeep`,
5. Enable GitHub Pages in your settings for your desired branch.

#### MySQL Storage

**[godoc: storage/mysql](https://godoc.org/github.com/sourcegraph/checkup/storage/mysql)**

A MySQL database can be configured as a storage backend.

Example configuration:

```js
{
    "type": "mysql",
    "create": true,
    "dsn": "checkup:checkup@tcp(mysql-checkup-db:3306)/checkup"
}
```

When `create` is set to true, checkup will issue `CREATE TABLE` statements required for storage.

#### SQLite3 Storage (requires CGO to build, not available as a default)

**[godoc: storage/sqlite3](https://godoc.org/github.com/sourcegraph/checkup/storage/sqlite3)**

A SQLite3 database can be configured as a storage backend.

Example configuration:

```js
{
    "type": "sqlite3",
    "create": true,
    "dsn": "/path/to/your/sqlite.db"
}
```

When `create` is set to true, checkup will issue `CREATE TABLE` statements required for storage.

#### PostgreSQL Storage

**[godoc: storage/postgres](https://godoc.org/github.com/sourcegraph/checkup/storage/postgres)**

A PostgreSQL database can be configured as a storage backend.

Example configuration:

```js
{
    "type": "postgres",
    "dsn": "host=postgres-checkup-db user=checkup password=checkup dbname=checkup sslmode=disable"
}
```

When `create` is set to true, checkup will issue `CREATE TABLE` statements required for storage.


#### Azure Application Insights Storage

**[godoc: appinsights](https://godoc.org/github.com/sourcegraph/checkup/storage/appinsights)**

Azure Application Insights can be used as a storage backend, enabling Checkup to be used as a source of custom availability tests and metrics.  An example use case is documented [here](https://docs.microsoft.com/en-us/azure/azure-monitor/app/availability-azure-functions).

A sample storage configuration with retries enabled:
```js
{
  "type": "appinsights",
  "test_location": "data center 1",
  "instrumentation_key": "11111111-1111-1111-1111-111111111111",
  "retry_interval": 1,
  "max_retries": 3,
  "tags": {
    "service": "front end",
    "product": "main web app"
  }
} 
```

The following keys are optional:

- `test_location` (default is **Checkup Monitor**)
- `retry_interval` (default is 0)
- `max_retries` (default is 0)
- `timeout` (defaults to 2 seconds if omitted or set to 0)
- `tags`

If retries are disabled, the plugin will wait up to `timeout` seconds to submit telemetry before closing.

When check results are sent to Application Insights, the following values are included in the logged telemetry:

- `success` is set to `1` if the check passes, `0` otherwise
- `message` is set to `Up`, `Down`, or `Degraded`
- `duration` is set to the average of all check result round-trip times and is displayed as a string in milliseconds
- `customMeasurements` is set to a JSON object including the number of the check as a string and the round-trip time of the check in nanoseconds
- If the check included a `threshold_rtt` setting, it will be added to the `customDimensions` JSON object as key `ThresholdRTT` with a time duration string value (ie: `200ms`)
- If any tags were included in the storage configuation, they will be added to the `customDimensions` JSON object

Currently the status page does not support Application Insights storage.

#### Slack notifier

Enable notifications in Slack with this Notifier configuration:
```js
{
    "type": "slack",
    "username": "username",
    "channel": "#channel-name",
    "webhook": "webhook-url"
}
```

Follow these instructions to [create a webhook](https://get.slack.help/hc/en-us/articles/115005265063-Incoming-WebHooks-for-Slack).

#### Mail notifier

Enable E-mail notifications with this Notifier configuration:
```js
{
    "type": "mail",
    "from": "from@example.com",
    "to": [ "support1@example.com", "support2@example.com" ],
    "subject": "Custom subject line",
    "smtp": {
        "server": "smtp.example.com",
        "port": 25,
        "username": "username",
        "password": "password"
    }
}
```

The settings for `subject`, `smtp.port` (default to 25), `smtp.username` and `smtp.password` are optional.

#### Mailgun notifier

Enable notifications using Mailgun with this Notifier configuration:
```js
{
    "type": "mailgun",
    "from": "sender@example.com",
    "to": [ "support1@example.com", "support2@example.com" ],
    "subject": "Custom subject line"
    "apikey": "mailgun-api-key",
    "domain": "mailgun-domain",
}
```

#### Pushover notifier

Enable notifications using Pushover with this Notifier configuration:
```js
{
    "type": "pushover",
    "token": "API_TOKEN",
    "recipient": "USER_KEY"
    "subject": "Custom subject line"
}
```

## Setting up storage on S3

The easiest way to do this is to give an IAM user these two privileges (keep the credentials secret):

- arn:aws:iam::aws:policy/**IAMFullAccess**
- arn:aws:iam::aws:policy/**AmazonS3FullAccess**

### Implicit Provisioning

If you give these permissions to the same user as with the credentials in your JSON config above, then you can simply run:

```bash
$ checkup provision
```

and checkup will read the config file and provision S3 for you. If the user is different, you may want to use explicit provisioning instead.

This command creates a new IAM user with read-only permission to S3 and also creates a new bucket just for your check files. The credentials of the new user are printed to your screen. **Make note of the Public Access Key ID and Public Access Key!** You won't be able to see them again.

**IMPORTANT SECURITY NOTE:** This new IAM user will have read-only permission to all S3 buckets in your AWS account, and its credentials will be visible to any visitor to your status page. If you do not want to grant visitors to your status page read access to all your S3 buckets, you need to modify this IAM user's permissions to scope its access to the Checkup bucket. If in doubt, restrict access to your status page to trusted visitors. It is recommended that you do NOT include ANY sensitive credentials on the machine running Checkup.


### Explicit Provisioning

If you do not prefer implicit provisioning using your `checkup.json` file, do this instead. Export the information to environment variables and run the provisioning command:

```bash
$ export AWS_ACCESS_KEY_ID=...
$ export AWS_SECRET_ACCESS_KEY=...
$ export AWS_BUCKET_NAME=...
$ checkup provision s3
```

### Manual Provisioning

If you'd rather do this manually, see the [instructions on the wiki](https://github.com/sourcegraph/checkup/wiki/Provisioning-S3-Manually) but keeping in mind the region must be **US Standard**.


## Checkup status page

Checkup now has a local HTTP server that supports serving checks stored in:

- FS (local filesystem storage),
- MySQL
- PostgreSQL
- SQLite3 (not enabled by default)

You can run `checkup serve` from the folder which contains `checkup.json`
and the `statuspage/` folder.

### Setting up the status page for GitHub

You will need to edit `

### Setting up the status page for S3

In statuspage/js, use the contents of [config_s3.js](https://github.com/sourcegraph/checkup/blob/master/statuspage/js/config_s3.js) to fill out [config.js](https://github.com/sourcegraph/checkup/blob/master/statuspage/js/config.js), which is used by the status page.
This is where you specify how to access the S3 storage bucket you just provisioned for check files.

As you perform checks, the status page will update every so often with the latest results. **Only checks that are stored will appear on the status page.**


## Performing checks

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


## Posting status messages

Site reliability engineers should post messages when there are incidents or other news relevant for a status page. This is also very easy:

```bash
$ checkup message --about=Example "Oops. We're trying to fix the problem. Stay tuned."
```

This stores a check file with your message attached to the result for a check named "Example" which you configured in `checkup.json` earlier.




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

You can implement your own Checker and Storage types. If it's general enough, feel free to submit a pull request so others can use it too!

### Building Locally

Requires Go v1.14 or newer. Building with the latest Go version is encouraged.

```bash
git clone git@github.com:sourcegraph/checkup.git
cd checkup
make
```

Building the SQLite3 enabled version is done with `make build-sqlite3`. PostgreSQL and MySQL are enabled by default.

### Building a Docker image

If you would like to run checkup in a docker container, building it is done by running `make docker`.
It will build the version without sql support. An SQL supported docker image is currently not provided,
but there's a plan to do that in the future.
