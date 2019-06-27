package redisengine

import (
	redis "github.com/go-redis/redis"
	"github.com/morfien101/go-metrics-reciever/config"
)

var (
	defaultRedisOptions = redis.Options{
		MaxRetries: 3,
	}
)

// RedisEngine is used to handle requests to Redis
type RedisEngine struct {
	config config.RedisConfig
	client *redis.Client
}

// New returns a RedisEngine. When you are ready you are expected to call
// Start to make use of it, then Stop to tear down connections.
func New(c config.RedisConfig) *RedisEngine {
	return &RedisEngine{
		config: c,
	}
}

// Start will connect the redis engine to the redis server.
// It will also check that it can ping the server. It will return an error
// if the Ping fails.
func (re *RedisEngine) Start() error {
	redisOptions := defaultRedisOptions
	redisOptions.Addr = re.config.RedisHost + ":" + re.config.RedisPort
	c := redis.NewClient(&redisOptions)
	if err := c.Ping().Err(); err != nil {
		return err
	}
	re.client = c

	return nil
}

// Stop will shutdown the Redis Engine
// It will return a channel that receives an error
// if anything goes wrong. It will get a nil if its all good.
func (re *RedisEngine) Stop() <-chan error {
	c := make(chan error, 1)
	go func() {
		c <- re.client.Close()
		close(c)
	}()
	return c
}

func (re *RedisEngine) ValidateAuth(username, password string) (bool, error) {
	output, err := re.client.Get(username).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if output == password {
		return true, nil
	}
	return false, nil
}
