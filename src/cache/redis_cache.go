package cache

import (
	"github.com/garyburd/redigo/redis"
	//"config"
	"time"
)

type RedisCache struct {
  pool *redis.Pool
}

// If item found, always return nil error
func (rc *RedisCache) Get(key string) (data string, err error) {
	c := rc.pool.Get()
	if c.Err() != nil {
		data = ""
		err = c.Err()
	}else{
    	defer c.Close()
		data, err = redis.String(c.Do("GET", key))
	}
  return
}

func (rc *RedisCache) SetEx(key string, expiretime int32, data string) (error) {
	c := rc.pool.Get()
	if c.Err() != nil {
		return c.Err()
	}else{
    	defer c.Close()
		c.Do("SETEX", key, expiretime, data)
	}
  return nil
}

func (rc *RedisCache) Set(key string, data string) (error) {
	c := rc.pool.Get()
	if c.Err() != nil {
		return c.Err()
	}else{
    	defer c.Close()
		c.Do("SET", key, data)
	}
  return nil
}

func NewRedisCache (server , password string) *RedisCache {
	//server := config.GetRedis().Addr
	//password := config.GetRedis().Password
	if server == "" {
		return nil
	}
	
    return &RedisCache{
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
