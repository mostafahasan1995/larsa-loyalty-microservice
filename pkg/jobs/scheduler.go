package jobs

import (
	"context"
	"loyalty/pkg/caching"

	redislock "github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/redis/go-redis/v9"
)

var Scheduler gocron.Scheduler

// InitScheduler initializes a new scheduler
func InitScheduler() {
	ctx := context.TODO()
	redisClient, err := caching.Rdb.GetClient()
	if err != nil {
		panic(err)
	}
	locker, err := redislock.NewRedisLocker(
		redis.NewClient(&redis.Options{Addr: redisClient.Client.Options().Addr}),
		redislock.WithTries(1),
	)
	if err != nil {
		panic(err)
	}
	Scheduler, err := gocron.NewScheduler(gocron.WithDistributedLocker(locker))
	if err != nil {
		panic(err)
	}
	InitNotifyOldVouchersDailyJob(ctx, Scheduler)
	InitNotifyExpiredVouchersDailyJob(ctx, Scheduler)
	Scheduler.Start()
	defer func() { _ = Scheduler.Shutdown() }()
}
