#!/usr/bin/env bash

set -e
set -o pipefail

function usage() {
	echo "Usage: [DOCKER_PLATFORM=os/arch] $0 target"
	echo "Runs the build in a predictable Docker environment."
	echo "'target' is a build target (try 'help')."
	echo "DOCKER_PLATFORM can optionally be set."
	exit 1
}

if [ $# -lt 1 ] ; then
	usage
fi
if [ "$1"  == "-h" ] || [ "$1" == "--help"  ] ; then
	usage
fi
TTY=""
if [ -t 0 ]; then
	TTY="--tty"
fi

if [ -z "$DOCKER_PLATFORM" ] ; then
	DOCKER_PLATFORM="linux/$(docker system info --format '{{.Architecture}}')"
fi
GID="$(id -g)"
GROUP="$(getent group $(getent passwd $USER | cut -d: -f4) | cut -d: -f1)"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
if ! test -d "$GIT_ROOT"/.cache ; then
	mkdir "$GIT_ROOT"/.cache
fi

docker build \
	--platform ${DOCKER_PLATFORM} \
	--build-arg USER="$USER" \
	--build-arg UID="$UID" \
	--build-arg GROUP="$GROUP" \
	--build-arg GID="$GID" \
	--build-arg HOME="$HOME" \
	--tag resonance:local \
	--quiet \
	. > /dev/null

NAME="resonance-$$"

function kill_container() {
	docker kill --signal SIGKILL "${NAME}" &>/dev/null || true
}

trap kill_container EXIT

docker run \
	--name "${NAME}" \
	--platform ${DOCKER_PLATFORM} \
	--user "${UID}:${GID}" \
	--rm \
	${TTY} \
	--interactive \
	--volume ${GIT_ROOT}:${HOME}/resonance \
	--volume ${GIT_ROOT}/.cache:${HOME}/.cache \
	--volume ${HOME}/resonance/.cache \
	--workdir ${HOME}/resonance \
	resonance:local \
	make --no-print-directory "${@}"