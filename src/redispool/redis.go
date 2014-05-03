package redispool

import (
//	"fmt"
	"github.com/garyburd/redigo/redis"
)
var MAX_POOL_SIZE = 20

var redisPool chan redis.Conn

func putRedis(conn redis.Conn) {
	if redisPool == nil {
		redisPool = make(chan redis.Conn, MAX_POOL_SIZE)
	}
	if len(redisPool) >= MAX_POOL_SIZE {
		conn.Close()
		return
	}
	redisPool <- conn
}

func InitRedis(network, address string) redis.Conn {
	redisPool = make(chan redis.Conn, MAX_POOL_SIZE)
	if len(redisPool) == 0 {
		go func() {
			for i := 0; i < MAX_POOL_SIZE/2; i++ {
				c, err := redis.Dial(network, address)
				if err != nil {
					panic(err)
				}
				putRedis(c)
			}
		}()
	}
	return <-redisPool
}