package store

import (
	"github.com/garyburd/redigo/redis"
	"config"
	"time"
)

type RedisStore struct {
  pool *redis.Pool
}

// If item found, always return nil error
func (s *RedisStore) Get(key string) (data []byte, err error) {
	c := s.pool.Get()
	if c.Err() != nil {
		data = nil
		err = c.Err()
	}else{
    	defer c.Close()
		s, err := redis.String(c.Do("GET", key))
		if err != nil {
			data = nil
		}else{
			data = []byte(s)
		}
	}
  return
}

func (s *RedisStore) Set(key string, expiretime int32, data []byte) (error) {
	c := s.pool.Get()
	if c.Err() != nil {
		return c.Err()
	}else{
    	defer c.Close()
		c.Do("SET", key, data)
	}
  return nil
}

func NewRedisStore () (store *RedisStore) {
	server := config.GetRedis().Addr
	password := config.GetRedis().Password
	if server == "" {
		return nil
	}
	
    return &RedisStore{
    	&redis.Pool{
	        MaxIdle: 3,
	        IdleTimeout: 240 * time.Second,
	        Dial: func () (redis.Conn, error) {
	            c, err := redis.Dial("tcp", server)
	            if err != nil {
	                return nil, err
	            }
	            if password != "" {
		            if _, err := c.Do("AUTH", password); err != nil {
		                c.Close()
		                return nil, err
		            }
	        	}
	            return c, err
	        },
	        TestOnBorrow: func(c redis.Conn, t time.Time) error {
	            _, err := c.Do("PING")
	            return err
	        },
	    },
	}

}
