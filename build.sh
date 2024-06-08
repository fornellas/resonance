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

case "$(uname -s)" in
	Linux)
		DOCKER_USER="$(id -un)"
		DOCKER_UID="$(id -u)"
		DOCKER_GROUP="$(id -gn)"
		DOCKER_GID="$(id -g)"
		;;
	Darwin)
		# Darwin runs containers as VMs, and mount volumes with the current user, so we use dummy
		# values here, which are known to be good for the container, and which are mapped to the
		# Darwin user on the volume.
		DOCKER_USER=resonance
		DOCKER_UID=1000
		DOCKER_GROUP=resonance
		DOCKER_GID=1000
		;;
esac

DOCKER_HOME="/home/$DOCKER_USER"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
if ! test -d "$GIT_ROOT"/.cache ; then
	mkdir "$GIT_ROOT"/.cache
fi

DOCKER_IMAGE="$(docker build \
	--platform ${DOCKER_PLATFORM} \
	--build-arg USER="$DOCKER_USER" \
	--build-arg UID="$DOCKER_UID" \
	--build-arg GROUP="$DOCKER_GROUP" \
	--build-arg GID="$DOCKER_GID" \
	--build-arg HOME="$DOCKER_HOME" \
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
	--user "${DOCKER_UID}:${DOCKER_GID}" \
	--rm \
	${TTY} \
	--interactive \
	--volume ${GIT_ROOT}:${DOCKER_HOME}/resonance \
	--volume ${GIT_ROOT}/.cache:${DOCKER_HOME}/resonance/.cache \
	--env XDG_CACHE_HOME=${DOCKER_HOME}/resonance/.cache \
	--workdir ${DOCKER_HOME}/resonance \
	${GOARCH_DOWNLOAD_ENV} \
	${DOCKER_IMAGE} \
	make --no-print-directory "${@}"