#!/bin/sh
export GOPATH=$(cd "$(dirname "$0")"; pwd)
export GOBIN=$GOPATH/bin
REVISION=`git rev-parse --short=5 HEAD`
go get github.com/vmihailenco/msgpack
go get github.com/golang/groupcache/lru
go build  kwan
mv kwan ./bin/
#go install kwan

