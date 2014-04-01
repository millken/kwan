package store

type Store interface {

  Get(key string) ([]byte, error)
  Set(key string, expiretime int32, data []byte) (error)

}
