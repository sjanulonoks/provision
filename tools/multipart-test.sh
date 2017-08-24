#!/usr/bin/env bash

set -e
echo "multi-part file upload test" 

SHATEST="9cf1f4a5f6b6eb21076609c560ae97bb6c802f76111a8a64c7105d09acb8c1ac"
SOURCE="../isos/multipart.pp"
USER="rocketskates:r0cketsk8ts"
TARGET="drp-data/plugins/multipart.pp"

echo "uploading Plugin Provider via multipart"
curl -X POST --user $USER -H "Content-Type: multipart/form-data" -F "file=@$SOURCE" --insecure https://127.0.0.1:8092/api/v3/plugin_providers/multipart.pp

echo "comparing SHAs"
echo "$SHATEST  $TARGET"
sudo sha256sum $TARGET

echo "uploading ISO via CLI"
./drpcli plugin_providers upload $SOURCE as slack
echo "$SHATEST  $TARGET"
sudo sha256sum $TARGET


echo "uploading ISO via multipart"

SHATEST="08e24d13d2dc68035d93ccd461a7e6dcdac89f52c7ad82d8a64f9edf4b8ea494"
SOURCE="../isos/multipart.iso"
USER="rocketskates:r0cketsk8ts"
TARGET="drp-data/tftpboot/isos/multipart.iso"

curl -X POST --user $USER -H "Content-Type: multipart/form-data" -F "file=@$SOURCE" --insecure https://127.0.0.1:8092/api/v3/isos/multipart.iso

echo "comparing SHAs"
echo "$SHATEST  $TARGET"
sha256sum $TARGET


echo "uploading ISO via CLI"
./drpcli isos upload $SOURCE as multipart.iso
echo "$SHATEST  $TARGET"
sha256sum $TARGET

