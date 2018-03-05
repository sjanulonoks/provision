#!/usr/bin/env bash

GOOS=linux GOARCH=amd64 go build -ldflags "$VERFLAGS" -o ../embedded/assets/drpcli.amd64.linux ../cmds/drpcli/drpcli.go
GOOS=windows GOARCH=amd64 go build -ldflags "$VERFLAGS" -o ../embedded/assets/drpcli.amd64.windows ../cmds/drpcli/drpcli.go
