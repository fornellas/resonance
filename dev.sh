#!/usr/bin/env bash

set -e
set -o pipefail

cd "$(dirname "$0")"

####################################################################################################
# Global variables
####################################################################################################

NAME="$(basename "$(pwd)")"

##
## uname
##

SYSTEM="$(uname -s)"

##
## Docker
##

if [ -z "$DOCKER_PLATFORM" ] ; then
    DOCKER_PLATFORM_ARCH_NATIVE="$(docker system info --format '{{.Architecture}}')"
    DOCKER_PLATFORM="linux/$DOCKER_PLATFORM_ARCH_NATIVE"
fi
# We fix user / group here, because while on Linux hosts, we can use the exact same values in the
# container, for Darwin hosts, we can't, as Darwin accepts names which aren't good for Linux.
# We fix these for both cases.
DOCKER_USER=resonance
DOCKER_GROUP=resonance
case "${SYSTEM}" in
    Linux)
        # For Linux hosts, we can map these 1:1.
        DOCKER_UID="$(id -u)"
        DOCKER_GID="$(id -g)"
        ;;
    Darwin)
        # Darwin runs containers on a VM, and volumes map to the host user UID/GID always. For
        # the container, we fix safe values here that won't clash with any system accounts.
        DOCKER_UID=1000
        DOCKER_GID=1000
        ;;
    *)
        echo "unsupported system: ${SYSTEM}"
        return 1
        ;;
esac
DOCKER_HOME="/home/${DOCKER_USER}"
DOCKER_IMAGE="${NAME}:local"
DOCKER_CONTAINER="${NAME}"

##
## Git
##

GIT_ROOT="$(cd $(dirname $0) && git rev-parse --show-toplevel)"
DOCKER_GIT_ROOT="${DOCKER_HOME}/${NAME}"
DOCKER_GIT_HOME="${DOCKER_GIT_ROOT}/.home"
GIT_HOME="${GIT_ROOT}/.home/${DOCKER_PLATFORM}"
GIT_HOME_ROOT="${GIT_HOME}/${NAME}"

##
## Ssh
##

SSH_HOST=127.0.0.1
SSH_PORT=2222
SSH_KNOWN_HOSTS="${HOME}/.ssh/known_hosts"
SSH_KNOWN_HOSTS_HOSTNAME="[${SSH_HOST}]:${SSH_PORT}"
SSH_CLIENT_PUBLIC_KEYS_GLOB="~/.ssh/id_*.pub"

####################################################################################################
# Functions
####################################################################################################

function start() {
    if [ $# -gt 0 ] ; then
        echo "invalid arguments: ${@}"
        return 1
    fi

    if status &>/dev/null ; then
        echo "🌱 Already running"
        return
    fi

    echo "🔧 Building image..."
    docker build \
        --platform "${DOCKER_PLATFORM}" \
        --build-arg "USER=${DOCKER_USER}" \
        --build-arg "GROUP=${DOCKER_GROUP}" \
        --build-arg "UID=${DOCKER_UID}" \
        --build-arg "GID=${DOCKER_GID}" \
        --build-arg "HOME=${DOCKER_HOME}" \
        --tag "${DOCKER_IMAGE}" \
        --quiet \
        .

    echo "🚀 Running container..."
    echo mkdir -p "${GIT_HOME_ROOT}"
    mkdir -p "${GIT_HOME_ROOT}"
    docker run \
        --name "${DOCKER_CONTAINER}" \
        --detach \
        --platform "${DOCKER_PLATFORM}" \
        --publish "${SSH_PORT}:22" \
        --rm \
        --volume "${GIT_HOME}:${DOCKER_HOME}" \
        --volume "${GIT_ROOT}:${DOCKER_GIT_ROOT}" \
        --volume "${DOCKER_GIT_HOME}" \
        "${DOCKER_IMAGE}" \
        sh -c "chown ${DOCKER_UID}:${DOCKER_GID} ${DOCKER_HOME} && mkdir /run/sshd && /usr/sbin/sshd -D"
    trap stop ERR

    echo "🔑 Setting up SSH keys"
    ssh-keygen -q -f "${SSH_KNOWN_HOSTS}" -R "${SSH_KNOWN_HOSTS_HOSTNAME}"
    docker exec resonance sh -c "cat /etc/ssh/ssh_host_*_key.pub" \
        | awk '{print "'"${SSH_KNOWN_HOSTS_HOSTNAME}"' "$0}' \
        >> "${SSH_KNOWN_HOSTS}"
    if ! eval "ls ${SSH_CLIENT_PUBLIC_KEYS_GLOB}" &>/dev/null ; then
        echo "❌ No public keys found: ${SSH_CLIENT_PUBLIC_KEYS_GLOB}"
        echo "You can generate public keys by running:"
        echo "\$ ssh-keygen"
        echo "then you can try again."
        return 1
    fi
    mkdir -p "${GIT_HOME}/.ssh"
    chmod 700 "${GIT_HOME}/.ssh"
    eval "cat ${SSH_CLIENT_PUBLIC_KEYS_GLOB}" \
        > "${GIT_HOME}/.ssh/authorized_keys"
    chmod 644 "${GIT_HOME}/.ssh/authorized_keys"

    status

    info
}

function stop() {
    if [ $# -gt 0 ] ; then
        echo "invalid arguments: ${@}"
        return 1
    fi
    if ! status &>/dev/null ; then
        echo "💀 Already stopped"
        return
    fi
    echo "💀 Stopping..."
    docker kill "${DOCKER_CONTAINER}"
}

function status() {
    if [ $# -gt 0 ] ; then
        echo "invalid arguments: ${@}"
        return 1
    fi
    if docker exec "${DOCKER_CONTAINER}" true &>/dev/null ; then
        echo "🌱 Running"
    else
        echo "💀 Stopped"
        return 1
    fi
}

function shell() {
    if [ $# -gt 0 ] ; then
        echo "invalid arguments: ${@}"
        return 1
    fi
    ssh -p "${SSH_PORT}" "${DOCKER_USER}@${SSH_HOST}"
}

function run() {
    echo "TODO run"
    return 1
}

function info() {
    echo "ℹ️ Information"
    echo "Shell:"
    echo "\$ $0 shell"
    echo "Zed:"
    echo "\$ zed -n ssh://${DOCKER_USER}@${SSH_HOST}:${SSH_PORT}${DOCKER_GIT_ROOT}"
}

function help() {
    if [ $# -gt 0 ] ; then
        echo "invalid arguments: ${@}"
        return 1
    fi
    echo "TODO help"
    return 1
}

ACTION="$1"
shift
case "${ACTION}" in
    start)
        start "${@}"
        ;;
    stop)
        stop "${@}"
        ;;
    status)
        status "${@}"
        ;;
    shell)
        shell "${@}"
        ;;
    run)
        run "${@}"
        ;;
    info)
        info "${@}"
        ;;
    help)
        help "${@}"
        ;;
    *)
        help "${@}"
        return 1
        ;;
esac


# # https://github.com/moby/moby/issues/42732
# if [ "$DOCKER_PLATFORM" == "linux/386" ] && [ "$DOCKER_PLATFORM_ARCH_NATIVE" == "x86_64" ] ; then
#   GO_ENV_ARGS="$GO_ENV_ARGS --env GOARCH_DOWNLOAD=386"
# fi
