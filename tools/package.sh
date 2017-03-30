#!/bin/bash

if ! [[ -x bin/linux/amd64/rocket-skates || ! -d assets/startup ]]; then
    echo "RocketSkates has not been built!"
    exit 1
fi
tmpdir="$(mktemp -d /tmp/rs-bundle-XXXXXXXX)"
cp -a bin assets/startup assets/templates assets/bootenvs tools/install.sh "$tmpdir"
(
    cd "$tmpdir"
    sha256sum $(find -type f) >sha256sums
    zip -p -r rocketskates.zip *
)

cp "$tmpdir/rocketskates.zip" .
rm -rf "$tmpdir"
