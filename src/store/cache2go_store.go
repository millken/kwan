package store

import (
  "time"
  "github.com/millken/cache2go"
  "fmt"
)

type Cache2goStore struct {
  client *cache2go.CacheTable
}

type Cache2goStruct struct {
	text []byte
}

// If item found, always return nil error
func (s *Cache2goStore) Get(key string) (data []byte, err error) {
  item, err := s.client.Value(key)
  if err != nil {
    return
  }
  err = nil
  data = item.Data().(*Cache2goStruct).text
  return
}

func (s *Cache2goStore) Set(key string, expiretime int32, data []byte) (error) {
  fmt.Println("enter cache2go Set %s", key)
  val := Cache2goStruct{data}
  s.client.Add(key, time.Duration(expiretime) * time.Second, &val )
  return nil
}

func NewCache2goStore () (store *Cache2goStore) {
  store = &Cache2goStore{client: cache2go.Cache("kwan")}
  return
}
