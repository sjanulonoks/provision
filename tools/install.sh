#!/usr/bin/env bash

case $(uname -s) in
    Darwin)
        binpath="bin/darwin/amd64"
        bindest="/usr/local/bin"
        # Someday, handle adding all the launchd stuff we will need.
        shasum="command shasum -a 256";;
    Linux)
        binpath="bin/linux/amd64"
        bindest="/usr/local/bin"
        if [[ -d /etc/systemd/system ]]; then
            # SystemD
            initfile="startup/rocketskates.service"
            initdest="/etc/systemd/system/rocketskates.service"
            starter="sudo systemctl daemon-reload && sudo systemctl start rocketskates"
        elif [[ -d /etc/init ]]; then
            # Upstart
            initfile="startup/rocketskates.unit"
            initdest="/etc/init/rocketskates.conf"
            starter="sudo service rocketskates start"
        elif [[ -d /etc/init.d ]]; then
            # SysV
            initfile="startup/rocketskates.sysv"
            initdest="/etc/init.d/rocketskates"
            starter="/etc/init.d/rocketskates start"
        else
            echo "No idea how to install startup stuff -- not using systemd, upstart, or sysv init"
            exit 1
        fi
        shasum="command sha256sum";;
    *)
        # Someday, support installing on Windows.  Service creation could be tricky.
        echo "No idea how to check sha256sums";;
esac

case $1 in
     install)
             $shasum -c sha256sums || exit 1
             sudo cp "$binpath"/* "$bindest"
             if [[ $initfile ]]; then
                 sudo cp "$initfile" "$initdest"
                 echo "You can start the RocketSkates service with:"
                 echo "$starter"
             fi;;
     remove)
         sudo rm "$bindest/rocket-skates" "$bindest/rscli" "$initdest";;
     *)
         echo 'Unknown action "$1". Please use "install" or "remove"';;
esac

