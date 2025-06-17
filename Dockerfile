FROM debian:bookworm-slim
RUN apt-get update && apt-get -y --no-install-recommends install \
    curl \
    build-essential \
    ca-certificates \
    git \
    less \
    unzip
RUN passwd -d root
ARG USER
ARG GROUP
ARG UID
ARG GID
ARG HOME
RUN addgroup --gid ${GID} ${GROUP} > /dev/null
RUN mkdir ${HOME}
RUN useradd --home-dir ${HOME} --gid ${GID} --no-create-home --shell /bin/bash --uid ${UID} ${USER}
RUN chown ${UID}:${GID} ${HOME}
# This is required on Darwin, as the UID/GID mappping is not 1:1.
RUN git config --global --add safe.directory ${HOME}/resonance
