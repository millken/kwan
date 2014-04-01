#!/bin/sh
export GOPATH=$(cd "$(dirname "$0")"; pwd)
export GOBIN=$GOPATH/bin
REVISION=`git rev-parse --short=5 HEAD`
go get github.com/ParsePlatform/go.grace
go get github.com/valyala/ybc/bindings/go/ybc
go get github.com/bradfitz/gomemcache/memcache
go get github.com/millken/cache2go
go get github.com/vmihailenco/msgpack
go build -ldflags "-s -X main.gitVersion $REVISION" kwan
mv kwan ./bin/
#go install kwan

