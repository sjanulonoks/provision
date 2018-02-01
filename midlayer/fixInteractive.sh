#!/bin/bash

prompt_to_copy() {
    local ans=""
    read -p "Copy $1 to $2? (y/n)" ans
    [[ $ans != y ]] && return
    cp "$1" "$2"
}

go test "$@"

for testcase in dhcp-tests/*/*.request; do
    [[ -f $testcase ]] || continue
    tc="${testcase%.request}"
    if ! diff -NZBu "$tc.response-expect" "$tc.response-actual"; then
        prompt_to_copy "$tc.response-actual" "$tc.response-expect"
    fi
    if ! diff -NZBu "$tc.logs-expect" "$tc.logs-actual"; then
        prompt_to_copy "$tc.logs-actual" "$tc.logs-expect"
    fi
done
