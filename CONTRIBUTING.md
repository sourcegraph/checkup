# How to Contribute

This Sourcegraph project is [MIT licensed](LICENSE) and accepts
contributions via GitHub pull requests. This document outlines some of
the conventions on development workflow, commit message formatting,
contact points and other resources to make it easier to get your
contribution accepted.

# Certificate of Origin

By contributing to this project you agree to the Developer Certificate of Origin
(DCO). This document was created by the Linux Kernel community and is a simple
statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

## Getting Started

You'll need Go 1.8 or newer installed.

1. [Fork this repo](https://github.com/sourcegraph/checkup). This makes a copy of the code you can write to.
2. If you don't already have this repo (sourcegraph/checkup.git) repo on your computer, get it with `go get github.com/sourcegraph/checkup/cmd/checkup`.
3. Tell git that it can push the sourcegraph/checkup.git repo to your fork by adding a remote: `git remote add myfork https://github.com/you/checkup.git`
4. Make your changes in the sourcegraph/checkup.git repo on your computer.
5. Push your changes to your fork: `git push myfork`
6. [Create a pull request](https://github.com/sourcegraph/checkup/pull/new/master) to merge your changes into sourcegraph/checkup @ master. (Click "compare across forks" and change the head fork.)

You can test your changes with `go run main.go` or `go build` if you want a binary plopped on disk. Use `go test -race ./...` from the root of the repo to run tests and make sure they pass!


## Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where you want to base your work (usually master).
- Make commits of logical units.
- Make sure your commit messages are in the proper format (see below).
- Push your changes to a topic branch in your fork of the repository.
- Make sure the tests pass, and add any new tests as appropriate.
- Submit a pull request to the original repository.

Thanks for your contributions!

### Format of the Commit Message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
scripts: add the test-cluster command

this uses tmux to setup a test cluster that you can easily kill and
start for debugging.

Fixes #38
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the
second line is always blank, and other lines should be wrapped at 80 characters.
This allows the message to be easier to read on Sourcegraph and GitHub as well
as in various git tools.
