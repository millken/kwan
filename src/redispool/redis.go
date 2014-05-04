package redispool

import (
	"config"
	"github.com/garyburd/redigo/redis"
	"time"
	"log"
)

var pool *redis.Pool
func newPool() *redis.Pool {
	server := config.GetRedis().Addr
	password := config.GetRedis().Password
	if pool != nil {
		return pool
	}
	if server == "" {
		return nil
	}
	
    return &redis.Pool{
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
    }
}


func Set(key , value string)  {
	pool = newPool()
	c := pool.Get()
	if c.Err() != nil {
		log.Printf("redis pool: %s", c.Err())
	}else{
    defer c.Close()
	c.Do("SET", key, value)
	}

}