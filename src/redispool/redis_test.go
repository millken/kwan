//http://my.oschina.net/golang/blog/161923
package redispool

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
	"conn"
)

func main() {
	fmt.Println()

	c := InitRedis("tcp", "127.0.0.1:6379")
	
	//test uuid
	fmt.Println(time.Now())
	startTime := time.Now()
	
	var Success, Failure int
	for i := 0; i < 100000; i++ {
		if ok, _ := redis.Bool(c.Do("HSET", "payVerify:session", uuid.New(), "aaaa")); ok {
			Success++
			// break
		} else {
			Failure++
		}
	}
	fmt.Println(time.Now())
	fmt.Println("用时：", time.Now().Sub(startTime), "总计：100000,成功：", Success, "失败：", Failure)
}