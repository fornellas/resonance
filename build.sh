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

case "$(uname -s)" in
	Linux)
		# Under Linux, as docker run containers as... containers, we can map user and group 1:1.
		DOCKER_USER="$(id -un)"
		DOCKER_GROUP="$(id -gn)"
		DOCKER_UID="$(id -u)"
		DOCKER_GID="$(id -g)"
		;;
	Darwin)
		# Darwin runs containers as VMs, and mount volumes with the current user, so we use dummy
		# values here, which are known to be good for the container, and which are mapped to the
		# Darwin user on the volume.
		DOCKER_USER=resonance
		DOCKER_GROUP=resonance
		# Darwin regular users UID/GID may also conflict with the container's, so we also use dummy
		# values here.
		DOCKER_UID=1000
		DOCKER_GID=1000
		;;
esac

DOCKER_HOME="/home/${DOCKER_USER}"

DOCKER_XDG_CACHE_HOME="${DOCKER_HOME}/.cache"

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"

DOCKER_IMAGE="$(docker build \
	--platform "${DOCKER_PLATFORM}" \
	--build-arg "USER=${DOCKER_USER}" \
	--build-arg "GROUP=${DOCKER_GROUP}" \
	--build-arg "UID=${DOCKER_UID}" \
	--build-arg "GID=${DOCKER_GID}" \
	--build-arg "HOME=${DOCKER_HOME}" \
	--quiet \
	.
)"

SYSTEM="$(uname -s)"
case "${SYSTEM}" in
	Linux)
		if [ -z "$XDG_CACHE_HOME" ] ;then
			XDG_CACHE_HOME="${HOME}/.cache"
		fi
		;;
	Darwin)
		if [ -z "$XDG_CACHE_HOME" ] ;then
			XDG_CACHE_HOME="${HOME}/Library/Caches"
		fi
		;;
	*)
		echo "Unsupported system ${SYSTEM}"
		exit 1
		;;
esac
mkdir -p "${XDG_CACHE_HOME}/resonance"

NAME="resonance-$$"

function kill_container() {
	docker kill --signal SIGKILL "${NAME}" &>/dev/null || true
}

trap kill_container EXIT

GO_ENV_ARGS=()
if [ -n "${GOOS}" ] ; then
	GO_ENV_ARGS=($GO_ENV_ARGS --env "GOOS=${GOOS}")
fi
if [ -n "${GOARCH}" ] ; then
	GO_ENV_ARGS=($GO_ENV_ARGS --env "GOARCH=${GOARCH}")
fi
if [ -n "${GO_TEST_BINARY_FLAGS_EXTRA}" ] ; then
	GO_ENV_ARGS=($GO_ENV_ARGS --env "GO_TEST_BINARY_FLAGS_EXTRA=${GO_TEST_BINARY_FLAGS_EXTRA}")
fi
# https://github.com/moby/moby/issues/42732
if [ "$DOCKER_PLATFORM" == "linux/386" ] && [ "$DOCKER_PLATFORM_ARCH_NATIVE" == "x86_64" ] ; then
	GO_ENV_ARGS=($GO_ENV_ARGS --env GOARCH_DOWNLOAD=386)
fi

docker run \
	--name "${NAME}" \
	--platform "${DOCKER_PLATFORM}" \
	--user "${DOCKER_UID}:${DOCKER_GID}" \
	--rm \
	${TTY} \
	--interactive \
	--volume "${GIT_ROOT}:${DOCKER_HOME}/resonance" \
	--volume "${XDG_CACHE_HOME}/resonance:${DOCKER_XDG_CACHE_HOME}/resonance" \
	--env "XDG_CACHE_HOME=${DOCKER_XDG_CACHE_HOME}" \
	"${GO_ENV_ARGS[@]}" \
	--workdir ${DOCKER_HOME}/resonance \
	${DOCKER_IMAGE} \
	make --no-print-directory "${@}"