#!/bin/bash

file="$1"

# Read QUESTION_ANSWERS from env, split into pairs
IFS='
'
set -- $QUESTION_ANSWERS
questions=()
answers=()
while [ $# -gt 0 ]; do
    questions+=("$1")
    shift
    answers+=("$1")
    shift
done


for i in $(seq 0 $((${#questions[@]} - 1))); do
    q="${questions[$i]}"
    a="${answers[$i]}"
    # Escape q for regex, a for sed replacement
    q_escaped=$(printf '%s\n' "$q" | sed 's/[][\/.^$*]/\\&/g')
    a_escaped=$(printf '%s\n' "$a" | sed 's/[&/\]/\\&/g')
    # Use sed to update the answer in place
    sed -i -e "/^$q_escaped=\".*\"/s|^$q_escaped=\".*\"|$q_escaped=\"$a_escaped\"|" "$file"
done
