#!/bin/sh
export GOPATH=$(cd "$(dirname "$0")"; pwd)
export GOBIN=$GOPATH/bin
REVISION=`git rev-parse --short=5 HEAD`
go get github.com/millken/falcore 
go get github.com/valyala/ybc/bindings/go/ybc
go get github.com/bradfitz/gomemcache/memcache
go get github.com/millken/cache2go
go get github.com/vmihailenco/msgpack
go get github.com/garyburd/redigo/redis
go get github.com/aybabtme/color/brush
go get github.com/golang/groupcache/lru
go build  kwan
mv kwan ./bin/
#go install kwan

