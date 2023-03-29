#!/usr/bin/env bash

set -e
set -o pipefail

if [ $# != 0 ] ; then
	echo "Usage: $0"
	echo "Starts a local development shell where the build can be run as close as possible to the CI"
	exit 1
fi

GID="$(id -g)"
GROUP="$(id -gn)"
GO_VERSION="$(awk '/^go /{print $2}' go.mod)"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
if ! test -d "$GIT_ROOT"/.cache ; then
	mkdir "$GIT_ROOT"/.cache
fi

NAME="resonance-$$"

HOME="/home/${USER}"

function kill_container() {
	docker kill --signal SIGKILL "${NAME}" &>/dev/null || true
}

trap kill_container EXIT

docker run \
	--name "${NAME}" \
	--platform linux \
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
set -o pipefail

passwd -d root > /dev/null

if getent group ${GID} > /dev/null; then
	groupmod --new-name ${GROUP} \$(getent group ${GID} | cut -d: -f1)
else
	addgroup --gid ${GID} ${GROUP} > /dev/null
fi

if getent passwd ${UID} > /dev/null; then
	usermod --badnames --home ${HOME} --gid ${GID} --login ${USER} --shell /bin/bash \$(getent passwd ${UID} | cut -d: -f1)
else
	useradd --badnames --home-dir ${HOME} --gid ${GID} --no-create-home --shell /bin/bash --uid ${UID} ${USER}
fi
ln -s ${HOME}/resonance/.bashrc ${HOME}/.bashrc

chown ${UID}:${GID} ${HOME}

# Shell
exec su --group ${GROUP} --pty ${USER} sh -c "
	set -e

	export BINDIR=\\\$(make BINDIR)
	PATH=\\\$BINDIR:\\\$PATH
	export GOBIN=\\\$(make GOBIN)
	PATH=\\\$GOBIN:\\\$PATH
	export GOCACHE=\\\$(make GOCACHE)
	export GOMODCACHE=\\\$(make GOMODCACHE)
  
	echo Available make targets:
	make help
	exec bash -i"
EOF
)"