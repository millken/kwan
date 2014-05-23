package store

var (
	Global Store
)

func init() {
	Global = NewYbcStore("kwan_cache")
}

func Get(key string) (data []byte, err error) {
	return Global.Get(key)
}

func Set(key string, expiretime int32, data []byte) (error) {
	return Global.Set(key, expiretime, data)
}