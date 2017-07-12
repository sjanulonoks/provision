#!/usr/bin/env bash

set -e
echo "mode: atomic" > coverage.txt

packages="github.com/digitalrebar/provision,\
github.com/digitalrebar/provision/backend,\
github.com/digitalrebar/provision/backend/index,\
github.com/digitalrebar/provision/midlayer,\
github.com/digitalrebar/provision/frontend,\
github.com/digitalrebar/provision/embedded,\
github.com/digitalrebar/provision/server,\
github.com/digitalrebar/provision/cli\
"

for d in $(go list ./... 2>/dev/null | grep -v cmds | grep -v vendor | grep -v github.com/digitalrebar/provision/client  | grep -v github.com/digitalrebar/provision/models) ; do
    tdir=$PWD
    dir=${d//github.com\/digitalrebar\/provision}
    echo "----------- TESTING $dir -----------"
    rm -f test.bin
    go test -o test.bin -c -race -covermode=atomic -coverpkg=$packages "$d"
    if [ -e test.bin ] ; then
        (cd .$dir; time "$tdir/test.bin" -test.coverprofile="$tdir/profile.out")
        rm -f test.bin
    fi

    if [ -f profile.out ]; then
        grep -h -v "^mode:" profile.out >> coverage.txt
        rm profile.out
    fi
done

