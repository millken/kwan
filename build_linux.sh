#!/bin/sh
export GOPATH=$(cd "$(dirname "$0")"; pwd)
export GOBIN=$GOPATH/bin
REVISION=`git rev-parse --short=5 HEAD`
go get github.com/ParsePlatform/go.grace
go build -ldflags "-s -X main.gitVersion $REVISION" -v
go install kwan

