#!/bin/bash
TFTPROOT=.
PROV_SLEDGEHAMMER_SIG=$1
if [[ ! $PROV_SLEDGEHAMMER_SIG ]] ; then
  echo "Sledgehammer Hash not specified"
  exit 1
fi

PROV_SLEDGEHAMMER_URL=$2
if [[ ! $PROV_SLEDGEHAMMER_URL ]] ; then
  echo "Sledgehammer URL not specified"
  exit 1
fi

SHA1SUM="sha1sum"
if [[ $(uname -s) == Darwin ]] ; then
    SHA1SUM="shasum -a 1"
fi

# Get sledgehammer
SS_URL=$PROV_SLEDGEHAMMER_URL/$PROV_SLEDGEHAMMER_SIG
SS_DIR=${TFTPROOT}/sledgehammer/$PROV_SLEDGEHAMMER_SIG
mkdir -p "$SS_DIR"
if [[ ! -e $SS_DIR/sha1sums ]]; then
    curl -fgL -o "$SS_DIR/sha1sums" "$SS_URL/sha1sums"
    while read f; do
        curl -fgL -o "$SS_DIR/$f" "$SS_URL/$f"
    done < <(awk '{print $2}' <"$SS_DIR/sha1sums")
    if ! (cd "$SS_DIR" && $SHA1SUM -c sha1sums); then
        echo "Download of sledgehammer failed or is corrupt!"
        rm -f "$SS_DIR/sha1sums"
        exit 1
    fi
fi

# Lift everything out of the discovery directory and replace it with a symlink
# Symlink is for backwards compatibility
if [[ -d $TFTPROOT/discovery && ! -L $TFTPROOT/discovery ]]; then
    for f in "${TFTPROOT}/discovery/"*; do
        [[ -e $f ]] || continue
        mv "$f" "${TFTPROOT}/${f##*/}"
    done
    rmdir "$TFTPROOT/discovery"
fi
[[ -L $TFTPROOT/discovery ]] || (cd "${TFTPROOT}"; ln -sf . discovery)
[[ -L $TFTPROOT/nodes ]] || (cd "${TFTPROOT}"; ln -sf machines nodes)

# Make it the discovery image
(cd "$TFTPROOT"; rm initrd0.img stage*.img vmlinuz0) || :
cp "$SS_DIR/"stage*.img "$SS_DIR/vmlinuz0" "$TFTPROOT"

