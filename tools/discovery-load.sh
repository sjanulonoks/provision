#!/bin/bash

export RS_KEY=${RS_KEY:-rocketskates:r0cketsk8ts}
./drpcli bootenvs install bootenvs/local.yml
./drpcli bootenvs install bootenvs/discovery.yml
./drpcli bootenvs install bootenvs/sledgehammer.yml

./drpcli set prefs unknownBootEnv discovery defaultBootEnv sledgehammer

