[![Latest Release](https://img.shields.io/github/v/release/fornellas/resonance)](https://github.com/fornellas/resonance/releases) [![Push](https://github.com/fornellas/resonance/actions/workflows/push.yaml/badge.svg)](https://github.com/fornellas/resonance/actions/workflows/push.yaml) [![Go Report Card](https://goreportcard.com/badge/github.com/fornellas/resonance)](https://goreportcard.com/report/github.com/fornellas/resonance) [![codecov](https://codecov.io/gh/fornellas/resonance/branch/master/graph/badge.svg?token=XIF06NYSWO)](https://app.codecov.io/gh/fornellas/resonance) [![Go Reference](https://pkg.go.dev/badge/github.com/fornellas/resonance.svg)](https://pkg.go.dev/github.com/fornellas/resonance) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) [![Buy me a beer: donate](https://img.shields.io/badge/Donate-Buy%20me%20a%20beer-yellow)](https://www.paypal.com/donate?hosted_button_id=AX26JVRT2GS2Q)

Status: experimental. Please check the [roadmap](./ROADMAP.md). Help welcome 🙏!

# resonance

A configuration management tool, somewhat similar to Ansible, Chef or Puppet, but with some notable features:

- Stateful: Host state is persisted, enabling:
  - Detecting external changes that may break automation.
  - Deletion of resources that are not required anymore.
  - Automatic rollback on failures: apply all changes successfully or rollback to previous state.
- Transactional resource changes.
  - Resources such as packages (eg: APT) can conflict if applied individually.
  - Resonance merges all of such resources and applies them together, preventing any conflicts.
- Painless refresh.
  - In memory state (eg: a daemon) must be refreshed when its dependencies change (eg: its configuration file).
  - By simply declaring first the configuration then the service, the service will be automatically reloaded only when required.
  - Resources may subscribe to any resources it depends on:
    - Eg: an app service is dependant on any configuration file at `/etc/app/**/*.conf`.
    - It is not required to fiddle with multiple individual dependecies declaration.
  - No more "forgot to declare to restart service".
- Painless dependencies.
  - Order in which resources are declared is used for applying them.
  - Merged resources are considered and do not break declared order.
- Speed
  - All read-only checks are run in parallel.
  - All actions that can happen concurrently are done like so (eg: changing multiple files).

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
./devshell.sh
```

Typically you'll want to stick to `make rrb`, as it enables you to edit files as preferred, and the build will automatically be triggered on any file changes.