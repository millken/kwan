// from Beego (http://beego.me/)
package cache
import "fmt"

type Cache interface {

	// get cached value by key.
	Get(key string) interface{}
	// set cached value with key and expire time.
	Put(key string, val interface{}, timeout int64) error
	// delete cached value by key.
	Delete(key string) error
	// increase cached int value by key, as a counter.
	Incr(key string) error
	// decrease cached int value by key, as a counter.
	Decr(key string) error
	// check cached value is existed or not.
	IsExist(key string) bool
	// clear all cache.
	ClearAll() error
	// start gc routine via config string setting.
	StartAndGC(config string) error
	Do(cmd string, args ...interface{}) (reply interface{}, err error)
}

var adapters = make(map[string]Cache)

// Register makes a cache adapter available by the adapter name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, adapter Cache) {
	if adapter == nil {
		panic("cache: Register adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("cache: Register called twice for adapter " + name)
	}
	adapters[name] = adapter
}

// Create a new cache driver by adapter and config string.
// config need to be correct JSON as string: {"interval":360}.
// it will start gc automatically.
func NewCache(adapterName, config string) (Cache, error) {
	adapter, ok := adapters[adapterName]
	if !ok {
		return nil, fmt.Errorf("cache: unknown adaptername %q (forgotten import?)", adapterName)
	}
	err := adapter.StartAndGC(config)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}
