#!/bin/bash

echo "Explode iso $1 $2 $3 $4"

rhelish_re='^(redhat|centos|fedora)'

os_name="$1"
tftproot="$2"
iso="$3"
os_install_dir="$4"
expected_sha="$5"

tftproot_cwd="`pwd`/"
if [[ "$tftproot" = /* ]] ; then
    tftproot_cwd=""
fi

iso_cwd="`pwd`/"
if [[ "$iso" = /* ]] ; then
    iso_cwd=""
fi

oid_cwd="`pwd`/"
if [[ "$os_install_dir" = /* ]] ; then
    oid_cwd=""
fi

extract() {
    if [[ $(uname -s) == Darwin ]] ; then
        # This is bsdtar under the covers.
        # If this errors, you may need to use homebrew to update
        # libarchive to get the latest version.
        tar -xf "$@"
    else
        bsdtar -xf "$@"
    fi
}

echo "Extracting $iso for $os_name"
[[ -d "${os_install_dir}.extracting" ]] && rm -rf "${os_install_dir}.extracting"
mkdir -p "${os_install_dir}.extracting"
case $os_name in
    esxi*)
        # ESXi needs some special love extracting the files from the image.
        # Specifically, bsdtar/hdiutil extracts everything in UPPERCASE,
        # where everything else expects lowercase
        (
            cd "${oid_cwd}${os_install_dir}.extracting"
            extract "${iso_cwd}${iso}"
            changed=true
            while [[ $changed = true ]]; do
                changed=false
                while read d; do
                    [[ $d = . || $d = ${d,,} ]] && continue
                    mv "$d" "${d,,}"
                    changed=true
                done < <(find . -type d |sort)
            done
            while read d; do
                [[ $d = ${d,,} ]] && continue
                mv "${d}" "${d,,}"
            done < <(find . -type f |sort)
            # ESX needs an exact version of pxelinux, so add it.
            cp "$tftproot_cwd"/esxi.0 pxelinux.0
        );;
    windows*)
        # bsdtar does not extract the UDF part of the ISO image, so
        # we will use 7zip to do it.
        (
            cd "${oid_cwd}${os_install_dir}.extracting"
            7z x "${iso_cwd}${iso}"
            # Windows needs wimboot, so extract it.  This must be kept in sync with
            # the version of wimboot we have made available in the Dockerfile.
            cp "$tftproot_cwd"/wimboot wimboot
            # Fix up permissions so things can execute
            chmod -R 555 .
        )
        ;;
    sledgehammer/*)
        # For Sledgehammer, we also check the sha1sums that were
        # extracted and die if they do not match
        (
            cd "${oid_cwd}${os_install_dir}.extracting"
            extract "${iso_cwd}${iso}"
            SHA1SUM="sha1sum"
            if [[ $(uname -s) == Darwin ]] ; then
                SHA1SUM="shasum -a 1"
            fi
            if ! $SHA1SUM -c sha1sums; then
                echo "Sha1 check failed, invalid download."
                exit 1
            fi
        ) || exit 1;;

    *)
        # Everything else just needs bsdtar/hdiutil
        (cd "${oid_cwd}${os_install_dir}.extracting"; extract "${iso_cwd}${iso}");;
esac
if [[ $os_name =~ $rhelish_re ]]; then
    # Rewrite local package metadata.  This allows for properly
    # handling the case where we only use disc 1 of a multi-disc set
    # for initial install purposes.
    (
        cd "${oid_cwd}${os_install_dir}.extracting"
        groups=($(echo repodata/*comps*.xml))
        createrepo -g "${groups[-1]}" .
    )
fi
printf '%s' "$expected_sha" > "${os_install_dir}.extracting/.${os_name}.rebar_canary"
[[ -d "${os_install_dir}" ]] && mv "${os_install_dir}" "${os_install_dir}.deleting"
mv "${os_install_dir}.extracting" "${os_install_dir}"
rm -rf "${os_install_dir}.deleting"

if which selinuxenabled && selinuxenabled; then
    restorecon -R -F "$tftproot"
fi

