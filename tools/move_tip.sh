#!/bin/bash

tip=$(git log | head -1 | awk '{ print $2 }')
git tag --force tip $tip
git push --force --tags
