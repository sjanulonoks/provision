#!/bin/bash

set -e

if ! [[ -x bin/linux/amd64/rocket-skates || ! -d assets/startup ]]; then
    echo "RocketSkates has not been built!"
    exit 1
fi

case $(uname -s) in
    Darwin)
        shasum="command shasum -a 256";;
    Linux)
        shasum="command sha256sum";;
    *)
        # Someday, support installing on Windows.  Service creation could be tricky.
        echo "No idea how to check sha256sums"
        exit 1;;
esac


tmpdir="$(mktemp -d /tmp/rs-bundle-XXXXXXXX)"
cp -a bin assets/startup assets/templates assets/bootenvs tools/install.sh "$tmpdir"
(
    cd "$tmpdir"
    $shasum $(find . -type f) >sha256sums
    zip -p -r rocketskates.zip *
)

cp "$tmpdir/rocketskates.zip" .
rm -rf "$tmpdir"
