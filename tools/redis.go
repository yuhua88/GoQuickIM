package tools

import (
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var syncLock sync.Mutex
var RedisClientMap = map[string]*redis.Client{}

type RedisOption struct {
	Address  string
	Password string
	Db       int
}

func GetRedisInstance(redisOpt RedisOption) *redis.Client {
	address := redisOpt.Address
	passWord := redisOpt.Password
	db := redisOpt.Db

	syncLock.Lock()
	defer syncLock.Unlock()

	if redisCli, ok := RedisClientMap[address]; ok {
		return redisCli
	}
	client := redis.NewClient(&redis.Options{
		Addr:       address,
		Password:   passWord,
		DB:         db,
		MaxConnAge: 20 * time.Second,
	})
	RedisClientMap[address] = client

	return client
}
