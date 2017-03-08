#!/bin/bash

mkdir -p embedded/assets/provisioner
cd embedded/assets/provisioner

if [ ! -f busybox ] ; then 
    curl -sfgL https://s3-us-west-2.amazonaws.com/rackn-busybox/busybox -o busybox
fi
if [ ! -f ipxe.efi ] ; then
    curl -sfgL http://boot.ipxe.org/ipxe.efi -o ipxe.efi
fi
if [ ! -f ipxe.pxe ] ; then
    curl -sfgL http://boot.ipxe.org/ipxe.pxe -o ipxe.pxe
fi
if [ ! -f jq ] ; then
    curl -sfgL https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 -o jq
fi

curl -sfgL http://downloads.sourceforge.net/project/elilo/elilo/elilo-3.16/elilo-3.16-all.tar.gz -o elilo.tar.gz
tar xzf elilo.tar.gz ./elilo-3.16-x86_64.efi
tar xzf elilo.tar.gz ./elilo-3.16-ia32.efi
tar xzf elilo.tar.gz ./elilo-3.16-ia64.efi
mv elilo-3.16-x86_64.efi bootx64.efi
mv elilo-3.16-ia32.efi bootia32.efi
mv elilo-3.16-ia64.efi bootia64.efi
rm elilo.tar.gz

curl -sfgL https://s3.amazonaws.com/opencrowbar/provisioner/syslinux-6.03.tar.xz -o syslinux-6.03.tar.xz
for f in syslinux-6.03/bios/com32/elflink/ldlinux/ldlinux.c32 \
    syslinux-6.03/bios/core/lpxelinux.0 \
    syslinux-6.03/bios/com32/modules/pxechn.c32 \
    syslinux-6.03/bios/com32/libutil/libutil.c32; do
    tar xOJf syslinux-6.03.tar.xz "$f">"${f##*/}"
done
rm -rf syslinux-6.03.tar.xz syslinux-6.03

curl -sfgL https://s3.amazonaws.com/opencrowbar/provisioner/syslinux-3.86.tar.xz -o syslinux-3.86.tar.xz
tar xOJf syslinux-3.86.tar.xz syslinux-3.86/core/pxelinux.0 > esxi.0
rm -rf syslinux-3.86.tar.xz syslinux-3.86

if [ ! -f wimboot-2.5.2.tar.bz2 ] ; then
    curl -sfgL https://git.ipxe.org/releases/wimboot/wimboot-2.5.2.tar.bz2 -o wimboot-2.5.2.tar.bz2
fi

cd -

