#!/bin/bash

[[ $GOPATH ]] || export GOPATH="$HOME/go"
fgrep -q "$GOPATH/bin" <<< "$PATH" || export PATH="$PATH:$GOPATH/bin"

cd "$GOPATH/src/github.com/rackn/rocket-skates"
if ! which go &>/dev/null; then
        echo "Must have go installed"
        exit 255
fi

# Work out the GO version we are working with:
GO_VERSION=$(go version | awk '{ print $3 }' | sed 's/go//')
WANTED_VER=(1 8)
if ! [[ "$GO_VERSION" =~ (.*)\.(.*) ]]; then
    echo "Cannot figure out what version of Go is installed"
    exit 1
elif ! (( ${BASH_REMATCH[1]} > ${WANTED_VER[0]} || ${BASH_REMATCH[2]} >= ${WANTED_VER[1]} )); then
    echo "Go Version needs to be 1.8 or higher: currently $GO_VERSION"
    exit -1
fi

for tool in go-bindata swagger glide; do
    which "$tool" &>/dev/null && continue
    case $tool in
        go-bindata) go get -u github.com/jteeuwen/go-bindata/...;;
        swagger)    go get -u github.com/go-swagger/go-swagger/cmd/swagger;;
        glide)
            go get -v github.com/Masterminds/glide
            (cd "$GOPATH/src/github.com/Masterminds/glide" && git checkout tags/v0.12.3 && go install);;
        *) echo "Don't know how to install $tool"; exit 1;;
    esac
done

glide install
go generate server/assets.go
go build -o rocket-skates server/*
go build -o rscli cli/*

./tools/test.sh
echo "To rebuild after changes, rerun tools/build.sh"
