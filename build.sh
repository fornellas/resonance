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

DOCKER_PLATFORM_ARCH_NATIVE="$(docker system info --format '{{.Architecture}}')"
if [ -z "$DOCKER_PLATFORM" ] ; then
	DOCKER_PLATFORM="linux/$DOCKER_PLATFORM_ARCH_NATIVE"
fi
GOARCH_DOWNLOAD_ENV=""
# https://github.com/moby/moby/issues/42732
if [ "$DOCKER_PLATFORM" == "linux/386" ] && [ "$DOCKER_PLATFORM_ARCH_NATIVE" == "x86_64" ] ; then
	GOARCH_DOWNLOAD_ENV="--env GOARCH_DOWNLOAD=386"
fi

GID="$(id -g)"
GROUP="$(getent group $(getent passwd $USER | cut -d: -f4) | cut -d: -f1)"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
if ! test -d "$GIT_ROOT"/.cache ; then
	mkdir "$GIT_ROOT"/.cache
fi

DOCKER_IMAGE="$(docker build \
	--platform ${DOCKER_PLATFORM} \
	--build-arg USER="$USER" \
	--build-arg UID="$UID" \
	--build-arg GROUP="$GROUP" \
	--build-arg GID="$GID" \
	--build-arg HOME="$HOME" \
	--quiet \
	.
)"

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
	${GOARCH_DOWNLOAD_ENV} \
	${DOCKER_IMAGE} \
	make --no-print-directory "${@}"