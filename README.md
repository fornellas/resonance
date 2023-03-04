[![Go Report Card](https://goreportcard.com/badge/github.com/fornellas/resonance)](https://goreportcard.com/report/github.com/fornellas/resonance) [![Go Reference](https://pkg.go.dev/badge/github.com/fornellas/resonance.svg)](https://pkg.go.dev/github.com/fornellas/resonance) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Status: experimental.

# resonance

A configuration management tool, somewhat similar to Ansible, Chef or Puppet, but with some notable features:

- Stateful: Host state is persisted, enabling:
  - Detecting external changes that may break automation.
  - Deletion of resources that are not required anymore.
  - Automatic rollback on failures: "all or nothing" approach.
- Transactional resource changes.
  - Resources such as packages (eg: APT) can conflict if applied individually.
  - Resonance merges all of such resources and applies them "all or nothing", preventing any conflicts.
- "Just Works ©" refresh.
  - In memory state (eg: a daemon) must be refreshed when its dependencies change (eg: its configuration file).
  - By simply declaring first the configuration then the service, the service will be automatically reloaded only when required.
  - Resources may subscribe to any resources it depends on:
    - Eg: an app service is dependant on any configuration file at `/etc/app/**/*.conf`.
    - It is not required to fiddle with multiple individual dependecies declaration.
  - No more "forgot to declare to restart service".
- "Just Works ©" dependencies.
  - Order in which resources are declared is used for applying them.
  - Merged resources are considered and do not break declared order.
- Speed
  - All read-only checks are run in parallel.
  - All actions that can happen concurrently are done like so (eg: changing multiple files).