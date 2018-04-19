#!/bin/bash

set -e

if ! [[ -x bin/linux/amd64/dr-provision || ! -d assets/startup ]]; then
    echo "dr-provision has not been built!"
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

tools/build-all-license.sh > ALL-LICENSE

tmpdir="$(mktemp -d /tmp/rs-bundle-XXXXXXXX)"
cp ALL-LICENSE "$tmpdir"
cp -a bin "$tmpdir"
mkdir -p "$tmpdir/assets"
cp -a assets/startup "$tmpdir/assets"
mkdir -p "$tmpdir/tools"
cp -a tools/install.sh "$tmpdir/tools"
(
    cd "$tmpdir"
    $shasum $(find . -type f) >sha256sums
    zip -p -r dr-provision.zip *
)

cp "$tmpdir/dr-provision.zip" .
$shasum dr-provision.zip > dr-provision.sha256
HASH=$(git rev-parse HEAD)
echo $HASH > dr-provision-hash.$HASH
rm -rf "$tmpdir"
