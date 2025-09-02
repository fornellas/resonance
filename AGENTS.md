# resonance

This is a host configuration tool, similar to Puppet, Chef or Ansible, but stateful like Terraform. Resources can be declared, and applied, and each resource state is tracked, to enable rollback to the original state or recover from failed / partial apply attempt.

## Layout

- `cmd/`: contains the CLI interface.
- `host/`: holds an interface to enable iteracting with hosts (ie: calling syscalls). There are multiple concrete implementations.
- `resources/`: holds resource definitions of what can be managed at a host.
- `state/`: host state management, as a collection of resources that can have the state read / applied from / to a host.
- `store/`: holds an interface for persistent storage of host state and logging. It contains concrete implementations.

## Dev Commands

Never run other commands to run linters, tests or build, other than the ones listed below: the `Makefile` contains a lot of logic that's required.

To run all linters do:

```shell
make lint LINT_GOVULNCHECK_DISABLE=1
```

To run all tests do:

```shell
make test LINT_GOVULNCHECK_DISABLE=1 GO_TEST_NO_COVER=1
```

To run all tests for a package:

```shell
make test LINT_GOVULNCHECK_DISABLE=1 GO_TEST_NO_COVER=1 GO_TEST_PACKAGES=github.com/fornellas/resonance/${package}
```

To run a single test:

```shell
make test LINT_GOVULNCHECK_DISABLE=1 GO_TEST_NO_COVER=1 GO_TEST_PACKAGES=github.com/fornellas/resonance/${package} GO_TEST_FLAGS='-run Test${name}'
```

To build the final binary:

```shell
make build GO_BUILD_AGENT_NATIVE_ONLY=1
```

To run the whole build (linters, tests and build):

```shell
make ci-dev
```
