#!/usr/bin/env bash

set -e
set -o pipefail

if [ $# != 0 ] ; then
	echo "Usage: $0"
	echo "Starts a local development shell where the build can be run as close as possible to the CI"
	exit 1
fi

DOCKER_PLATFORM="linux/amd64"

USER_ID="$(id -u)"
GROUP_ID="$(id -g)"
GO_VERSION="$(awk '/^go /{print $2}' go.mod)"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
if ! test -d "$GIT_ROOT"/.cache ; then
	mkdir "$GIT_ROOT"/.cache
fi

set -x

docker run \
	--platform ${DOCKER_PLATFORM} \
	--rm \
	--tty \
	--interactive \
	--volume /home/user \
	--volume ${GIT_ROOT}:/home/user/resonance \
	--volume ${GIT_ROOT}/.cache:/home/user/.cache \
	--volume /home/user/resonance/.cache \
	--workdir /home/user/resonance \
	golang:${GO_VERSION}-bullseye \
	/bin/bash -c "$(cat <<EOF
set -e

# User
addgroup --gid ${GROUP_ID} user
useradd --home-dir /home/user --gid ${GROUP_ID} --no-create-home --shell /bin/bash --uid ${USER_ID} user
ln -s /home/user/resonance/.bashrc /home/user/.bashrc

# Shell
exec su --group user --pty user sh -c "
export BINDIR=/home/user/.cache/bin
export PATH=/home/user/.cache/bin:\$PATH
export GOCACHE=/home/user/.cache/go-build
export GOMODCACHE=/home/user/.cache/go-mod
echo Available make targets: &&
make help &&
exec bash -i"
EOF
)"