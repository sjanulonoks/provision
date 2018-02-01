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
    for item in response logs; do
        ex="$tc.$item-expect"
        ac="$tc.$item-actual"
        [[ -f $ex ]] || touch "$ex"
        if ! diff -NwBu "$ex" "$ac"; then
            prompt_to_copy "$ac" "$ex"
        fi
    done
done
