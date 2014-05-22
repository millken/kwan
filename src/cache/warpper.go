package cache

var (
	Global Cache
)

func init() {
	Global, _ = NewCache("memory", "{interval:30}")
}

func Put(name string, value interface{}, expired int64) error {
	return Global.Put(name, value, expired)
}

func Get(name string) interface{} {
	return Global.Get(name)
}

func Delete(name string) error {
	return Global.Delete(name)
}

func Incr(key string) error {
	return Global.Incr(key)
}

func Decr(key string) error {
	return Global.Decr(key)
}

func IsExist(name string) bool {
	return Global.IsExist(name)
}

func ClearAll() error {
	return Global.ClearAll()
}

func Do(cmd string, args ...interface{}) (reply interface{}, err error) {
	return Global.Do(cmd, args...)
}