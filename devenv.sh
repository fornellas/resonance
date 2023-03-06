#!/usr/bin/env bash

set -e
set -o pipefail

if [ $# != 0 ] ; then
	echo "Usage: $0"
	echo "Starts a local development shell where the build can be run as close as possible to the CI"
	exit 1
fi

DOCKER_PLATFORM="linux/amd64"

GID="$(id -g)"
GROUP="$(getent group $(getent passwd $USER | cut -d: -f4) | cut -d: -f1)"
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
	--volume ${GIT_ROOT}:${HOME}/resonance \
	--volume ${GIT_ROOT}/.cache:${HOME}/.cache \
	--volume ${HOME}/resonance/.cache \
	--workdir ${HOME}/resonance \
	golang:${GO_VERSION}-bullseye \
	/bin/bash -c "$(cat <<EOF
set -e

# User
addgroup --gid ${GID} ${GROUP}
useradd --home-dir ${HOME} --gid ${GID} --no-create-home --shell /bin/bash --uid ${UID} ${USER}
ln -s ${HOME}/resonance/.bashrc ${HOME}/.bashrc
chown ${UID}:${GID} ${HOME}

# Shell
exec su --group ${GROUP} --pty ${USER} sh -c "
	set -e

	export CACHE_DIR=${HOME}/.cache
	export BINDIR=\\\$(make BINDIR)
	PATH=\\\$BINDIR:\\\$PATH
	export GOBIN=\\\$(make GOBIN)
	PATH=\\\$GOBIN:\\\$PATH
	export GOCACHE=\\\$(make GOCACHE)
	export GOMODCACHE=\\\$(make GOMODCACHE)
  
	echo
	echo Available make targets:
	make help
	exec bash -i"
EOF
)"