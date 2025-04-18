FROM debian:bookworm

SHELL ["/bin/bash", "-c"]

# apt
RUN apt update
RUN apt upgrade
RUN apt -y --no-install-recommends install \
    ca-certificates \
    curl \
    gcc \
    git \
    less \
    libc6-dev \
    make \
    man \
    npm \
    openssh-server \
    shellcheck \
    shfmt \
    unzip
RUN rm -rf /var/cache/apt/archives/
RUN rm -rf /var/lib/apt/lists/

# bash-language-server
RUN npm i -g bash-language-server

# Go
ARG GO_VERSION
RUN set -ex; \
    set -o pipefail; \
    DPKG_ARCH="$(dpkg --print-architecture)"; \
    case "${DPKG_ARCH}" in \
    'i386') \
        GO_ARCH='386'; \
        ;; \
    'amd64') \
        GO_ARCH='amd64'; \
        ;; \
    'arm64') \
        GO_ARCH='arm64'; \
        ;; \
    *) \
        echo "unsupported dpkg architecture: ${DPKG_ARCH}"; \
        exit 1; \
    ;; \
    esac; \
    curl -sSfL https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz \
        | tar -zx -C /opt
RUN echo "GOROOT=/opt/go" >> /etc/environment

# protoc
ARG PROTOC_VERSION
RUN set -ex; \
    set -o pipefail; \
    DPKG_ARCH="$(dpkg --print-architecture)"; \
    case "${DPKG_ARCH}" in \
    'i386') \
        PROTOC_ARCH='x86_32'; \
        ;; \
    'amd64') \
        PROTOC_ARCH='x86_64'; \
        ;; \
    'arm64') \
        PROTOC_ARCH='aarch_64'; \
        ;; \
    *) \
        echo "Unknown machine ${DPKG_ARCH}"; \
        exit 1; \
        ;; \
    esac; \
    curl -sSfL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip \
        > /tmp/protoc.zip; \
    unzip -d /opt/protoc /tmp/protoc.zip; \
    rm -f /tmp/protoc.zip
RUN echo "PROTOC_BIN=/opt/protoc/bin" >> /etc/environment

# root
RUN passwd -d root

# group
ARG GID
ARG GROUP
RUN addgroup --gid ${GID} ${GROUP} > /dev/null

# user
ARG HOME
RUN mkdir ${HOME}
ARG UID
ARG USER
RUN useradd --home-dir ${HOME} --gid ${GID} --no-create-home --shell /bin/bash --uid ${UID} ${USER}
RUN chown ${UID}:${GID} ${HOME}
