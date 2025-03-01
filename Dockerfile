FROM debian:bookworm

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
