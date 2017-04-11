#!/bin/bash

set -e

if [ ! -e assets ] ; then
        echo "No assets directory to work from."
        exit 1
fi

export PATH=$PATH:`pwd`

cd assets

export RS_KEY=${RS_KEY:-rocketskates:r0cketsk8ts}
drpcli bootenvs install bootenvs/local.yml
drpcli bootenvs install bootenvs/discovery.yml
drpcli bootenvs install bootenvs/sledgehammer.yml

drpcli preferences set unknownBootEnv discovery defaultBootEnv sledgehammer

