#!/usr/bin/env bash

set -e
echo "multi-part file upload test" 

SHATEST="08e24d13d2dc68035d93ccd461a7e6dcdac89f52c7ad82d8a64f9edf4b8ea494"
SOURCE="../isos/multipart.iso"
USER="rocketskates:r0cketsk8ts"
TARGET="drp-data/tftpboot/isos/multipart.iso"

echo "uploading ISO via multipart"
curl -X POST --user $USER -H "Content-Type: multipart/form-data" -F "file=@$SOURCE" --insecure https://127.0.0.1:8092/api/v3/isos/multipart.iso

echo "comparing SHAs"
echo "$SHATEST  $TARGET"
sha256sum $TARGET


echo "uploading ISO via CLI"
./drpcli isos upload $SOURCE as multipart.iso
echo "$SHATEST  $TARGET"
sha256sum $TARGET
