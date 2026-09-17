package planner

import (
	"github.com/gen0cide/laforge/server/utils"
	"github.com/go-redis/redis/v8"
)

var (
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
)

func init() {
	laforgeConfig, err := utils.LoadServerConfig()
	if err != nil {
		return
	}

	if laforgeConfig.Graphql.RedisServerUri != "" {
		if laforgeConfig.Graphql.RedisPassword != "" {
			rdb = redis.NewClient(&redis.Options{
				Addr:     laforgeConfig.Graphql.RedisServerUri,
				Password: laforgeConfig.Graphql.RedisPassword,
				DB:       0, // use default DB
			})
		} else {
			rdb = redis.NewClient(&redis.Options{
				Addr:     laforgeConfig.Graphql.RedisServerUri,
				Password: "", // no password set
				DB:       0,  // use default DB
			})
		}
	} else {
		if laforgeConfig.Graphql.RedisPassword != "" {
			rdb = redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: laforgeConfig.Graphql.RedisPassword,
				DB:       0, // use default DB
			})
		} else {
			rdb = redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: "", // no password set
				DB:       0,  // use default DB
			})
		}
	}

}
