[![Latest Release](https://img.shields.io/github/v/release/fornellas/resonance)](https://github.com/fornellas/resonance/releases) [![Push](https://github.com/fornellas/resonance/actions/workflows/push.yaml/badge.svg)](https://github.com/fornellas/resonance/actions/workflows/push.yaml) [![Go Report Card](https://goreportcard.com/badge/github.com/fornellas/resonance)](https://goreportcard.com/report/github.com/fornellas/resonance) [![Coverage Status](https://coveralls.io/repos/github/fornellas/resonance/badge.svg)](https://coveralls.io/github/fornellas/resonance) [![Go Reference](https://pkg.go.dev/badge/github.com/fornellas/resonance.svg)](https://pkg.go.dev/github.com/fornellas/resonance) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) [![Buy me a beer: donate](https://img.shields.io/badge/Donate-Buy%20me%20a%20beer-yellow)](https://www.paypal.com/donate?hosted_button_id=AX26JVRT2GS2Q)

Status: experimental. Please check the [roadmap](./ROADMAP.md). Help welcome 🙏!

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
GOARCH=$(case $(uname -m) in i[23456]86) echo 386;; x86_64) echo amd64;; armv6l|armv7l) echo arm;; aarch64) echo arm64;; *) echo Unknown machine $(uname -m) 1>&2 ; exit 1 ;; esac) && wget -O- https://github.com/fornellas/resonance/releases/latest/download/resonance.linux.$GOARCH.gz | gunzip > resonance && chmod 755 resonance
./resonance --help
```

## Development

[Docker](https://www.docker.com/) is used to create a reproducible development environment on any machine:

```bash
git clone git@github.com:fornellas/resonance.git
cd resonance/
./build.sh
```

Typically you'll want to stick to `./build.sh rrb`, as it enables you to edit files as preferred, and the build will automatically be triggered on any file changes.