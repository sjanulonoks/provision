#!/bin/bash

if [[ $1 != "" ]] ; then
    cd ..
    exec > $2
fi

echo
echo "DigitalRebar Provision License"
echo

cat LICENSE

echo
echo "TODO: Get the downloaded Assets Licenses"
echo

cd vendor
find . | grep LICENSE | while read line ; do
    echo
    echo "GO Package License: $line"
    echo
    cat $line
done
cd ..

echo

