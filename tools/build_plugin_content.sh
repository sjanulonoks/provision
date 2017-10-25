#!/usr/bin/env bash

if [ ! -e "$1/content" ] ; then
    cat > $1/content.go <<EOF
package main

var contentYamlString string = ""

EOF
    exit 0
fi

cd $1/content
drpcli contents bundle ../content.go --format=go
cd -

