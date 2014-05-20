set GOPATH=%~dp0;
go get github.com/bradfitz/gomemcache/memcache
go get github.com/millken/cache2go
go get github.com/vmihailenco/msgpack
go build -ldflags "-s" kwan
mv kwan ./bin/

