#!/bin/bash

GO_TARGET_VER_MAJOR=1
GO_TARGET_VER_MINOR=8

export GOPATH=$(pwd)
export PATH=$PATH:$GOPATH/bin

if ! which go &>/dev/null; then
        echo "Must have go installed"
        exit -1
fi
# Work out the GO version we are working with:
GO_VERSION=$(go version | awk '{ print $3 }' | sed 's/go//')
GO_VER_MAJOR=""
GO_VER_MINOR=""
if [[ "$GO_VERSION" =~ (.*)\.(.*) ]]
then
        GO_VER_MAJOR=${BASH_REMATCH[1]}
        GO_VER_MINOR=${BASH_REMATCH[2]}
fi

if [[ $GO_VER_MAJOR -ne $GO_TARGET_VER_MAJOR || $GO_VER_MINOR -lt $GO_TARGET_VER_MINOR ]] ; then
        echo "Go Version needs to be 1.8 or higher: currently $GO_VERSION"
        exit -1
fi

if ! which go-bindata ; then
        go get -u github.com/jteeuwen/go-bindata/...
fi

if ! which swagger ; then
        go get -u github.com/go-swagger/go-swagger/cmd/swagger
fi

if ! which glide ; then
        go get -v github.com/Masterminds/glide
        cd $GOPATH/src/github.com/Masterminds/glide && git checkout tags/v0.12.3 && go install && cd -
fi

if [[ ! -d src/github.com/rackn/rocket-skates ]] ; then
        go get -v github.com/rackn/rocket-skates
fi

cd $GOPATH/src/github.com/rackn/rocket-skates

glide install

./tools/download-assets.sh

go generate server/main.go
go build -o rocket-skates server/*

./tools/test.sh

echo "To rebuild after changes:"
echo " export GOPATH=$GOPATH"
echo " export PATH=\$PATH:\$GOPATH/bin"
echo " cd $GOPATH/src/github.com/rackn/rocket-skates"
echo " go generate server/main.go"
echo " go build -o rocket-skates server/*"
