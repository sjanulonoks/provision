#!/bin/bash

set -e

if [ ! -e assets ] ; then
        echo "No assets directory to work from."
        exit 1
fi

export PATH=$PATH:`pwd`

cd assets

export RS_KEY=${RS_KEY:-rocketskates:r0cketsk8ts}

for i in local discovery sledgehammer ;
do
        if ! drpcli bootenvs exists $i ; then
                echo "Installing bootenv: $i"
                drpcli bootenvs install bootenvs/$i.yml
        fi
done

echo "Setting default preferences for discovery"
drpcli prefs set unknownBootEnv discovery defaultBootEnv sledgehammer

