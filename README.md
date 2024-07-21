[![Latest Release](https://img.shields.io/github/v/release/fornellas/resonance)](https://github.com/fornellas/resonance/releases) [![Push](https://github.com/fornellas/resonance/actions/workflows/push.yaml/badge.svg)](https://github.com/fornellas/resonance/actions/workflows/push.yaml) [![Go Report Card](https://goreportcard.com/badge/github.com/fornellas/resonance)](https://goreportcard.com/report/github.com/fornellas/resonance) [![Coverage Status](https://coveralls.io/repos/github/fornellas/resonance/badge.svg?branch=master)](https://coveralls.io/github/fornellas/resonance?branch=master) [![Go Reference](https://pkg.go.dev/badge/github.com/fornellas/resonance.svg)](https://pkg.go.dev/github.com/fornellas/resonance) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) [![Buy me a beer: donate](https://img.shields.io/badge/Donate-Buy%20me%20a%20beer-yellow)](https://www.paypal.com/donate?hosted_button_id=AX26JVRT2GS2Q)

**Status**: under active development, targetting a [v1 milestone](https://github.com/fornellas/resonance/milestone/1).

**🙏Help wanted🙏**: There are [plenty of issues to work on](https://github.com/fornellas/resonance/issues), of all sizes and difficulty. Getting a dev environment setup [takes less than 5 minutes](#development), so you can get your first PR easily!

# resonance

A configuration management tool with novel features to Ansible, Chef or Puppet:

- **Stateful**: Persistent host state enables deletion of old resources, rollback to previous state (on failures) and detection of external changes.
- **Transactional changes**: things such as APT packages are done "all together or nothing" (single `apt` call), eliminating isses with conflicting packages.
- **Painless refresh**: no need to manually tell "please restart the service after changes" as these are implicitly declared so things "just work".
- **Painless dependencies**: declared order is always respected.
- **Speed**: read-only checks and possible changes happen concurrently; a lightweight agent is used so things fly even via SSH.

## Install

Pick the [latest release](https://github.com/fornellas/resonance/releases) with:

```bash
GOARCH=$(case $(uname -m) in i[23456]86) echo 386;; x86_64) echo amd64;; armv6l|armv7l) echo arm;; aarch64) echo arm64;; *) echo Unknown machine $(uname -m) 1>&2 ; exit 1 ;; esac) && wget -O- https://github.com/fornellas/resonance/releases/latest/download/resonance.$(uname -s | tr A-Z a-z).$GOARCH.gz | gunzip > resonance && chmod 755 resonance
./resonance --help
```

## Development

[Docker](https://www.docker.com/) is used to create a reproducible development environment on any machine:

```bash
git clone git@github.com:fornellas/resonance.git
cd resonance/
./build.sh ci
```

This will execute the build exactly as it happens on CI. You can get a development shell with `./build.sh shell`, from where you can run `make help` and see other options.

### Linux

Under Linux, not only you can have fast local builds exactly as it happens on CI, but you can also have the build run automatically on file changes (via [rrb](https://github.com/fornellas/rrb)):

```bash
./build.sh rrb
```

Just edit files with your preferred editor and as soon as you save them, the bulid will be executed automatically.

### MacOS / Darwin

While the Docker build runs fine under MacOS / Darwin, sadly the performance is notoriously bad (Linux containers run under a VM). It is possible to run the build locally without a container by:

- Install GNU Make (eg: `brew install make`)*[^1]
- Run the build `gmake ci`.

Note that a lot of the tests are Linux only (all `*_linux_test.go` files), so while build signal should be representative, test results aren't.

[^1]: Apple ships an ancient Make version which will NOT work, you must use a recent GNU Make.