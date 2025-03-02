#!/bin/bash
# shellcheck source=.env
source ~/.env
cd "${_GIT_ROOT}" || exit
PS1='$(EXIT_STATUS=$? && [ "${EXIT_STATUS}" -ne 0 ] && echo "\[\e[31m\](${EXIT_STATUS})\[\e[0m\]")\[\e[34m\]$_NAME\[\e[0m\]\[\e[1:37m\]:\[\e[0m\]\w\[\e[1:37m\]\$\[\e[0m\] '
