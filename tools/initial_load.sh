#!/bin/bash

while [ ! -e drp-data/tftpboot/files/jq ] ; do
        sleep 3
done

export RS_KEY=rocketskates:r0cketsk8ts
./drpcli bootenvs install bootenvs/local.yml
./drpcli bootenvs install bootenvs/discovery.yml
./drpcli bootenvs install bootenvs/sledgehammer.yml


