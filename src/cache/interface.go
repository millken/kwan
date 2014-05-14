package cache

type Cache interface {

  Get(key string) (string, error)
  Set(key string, data string) (error)
  SetEx(key string, expiretime int32, data string) (error)
  Do(cmd string, args ...interface{}) (reply interface{}, err error)
}
