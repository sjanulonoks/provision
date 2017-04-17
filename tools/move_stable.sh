#!/bin/bash

if [[ $1 == "" ]] ; then
        echo "Missing version tag"
        exit 1
fi

stable=$(git log $1 | head -1 | awk '{ print $2 }')
git tag --force stable $stable
git push --force --tags
