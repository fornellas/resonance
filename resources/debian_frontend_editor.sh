#!/bin/sh

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

file="$1"

for i in $(seq 0 $((${#questions[@]} - 1))); do
    q="${questions[$i]}"
    a="${answers[$i]}"
    # Use sed to update the answer in place
    sed -i -e "/^$q=\".*\"/s|^$q=\".*\"|$q=\"$a\"|" "$file"
done
