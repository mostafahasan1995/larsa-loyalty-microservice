package caching

import (
	"fmt"
	"log"
	"loyalty/pkg/util"
	"strconv"
	"time"

	"git.larsa.io/mahdawi/microservices-commons.git/cache"
)

var Rdb cache.Cache

func InitRedis() {
	clusterMode := util.GetEnv("REDIS_CLUSTER", "false")
	host := util.GetEnv("REDIS_HOST", "localhost")
	port := util.GetEnv("REDIS_PORT", "6379")
	password := util.GetEnv("REDIS_PASSWORD", "")
	db, err := strconv.Atoi(util.GetEnv("REDIS_DB", "0"))

	if err != nil {
		log.Fatal(err)
	}

	uri := host + ":" + port
	var redisCache cache.Cache

	if clusterMode == "true" {
		redisCache = cache.NewRedisClusterCache()
	} else {
		redisCache = cache.NewRedisCache()
	}

	errCache := redisCache.Connect(uri, password, db)
	if errCache != nil {
		log.Fatal(errCache)
	}
	fmt.Println("Redis successfully connected and pinged.")
	if clusterMode == "true" {
		fmt.Println("redis mode: cluster")
	} else {
		fmt.Println("redis mode: single instance")
	}

	Rdb = redisCache
}

func CacheByteByKey(key string, val []byte, ex time.Duration) {
	client, _ := Rdb.GetClient()
	client.Client.Set(client.Client.Context(), key, val, ex)
}
