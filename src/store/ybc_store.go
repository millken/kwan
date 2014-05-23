package store
import (
	"log"
	"time"
	"github.com/valyala/ybc/bindings/go/ybc"
)

type YbcStore struct {
  client *ybc.SimpleCache
}

// If item found, always return nil error
func (s *YbcStore) Get(key string) (data []byte, err error) {
  data, err = s.client.Get([]byte(key))
  return
}

func (s *YbcStore) Set(key string, expiretime int32, data []byte) (error) {
  err := s.client.Set([]byte(key), data, time.Duration(int64(expiretime))*time.Second)
  return err
}

func NewYbcStore (name string) (store *YbcStore) {
  	config := ybc.Config{
  		DataFile: "" + name + ".data",
  		IndexFile: "" + name + ".index",
		MaxItemsCount: ybc.SizeT(1000000),
		DataFileSize: ybc.SizeT(1000) * ybc.SizeT(1024*1024),
	}
	sc, err := config.OpenSimpleCache(100000, true)
	if err != nil {
	  log.Fatalf("Error when opening the cache: [%s]", err)
	}
	//defer sc.Close()
	store = &YbcStore{client: sc}
	return
}
