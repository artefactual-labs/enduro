package objectevent

import "fmt"

type Config struct {
	Enabled      bool
	Listen       string
	RedisAddress string
	RedisList    string
	BucketsPath  string
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Listen == "" {
		return fmt.Errorf("objectEventWebhook.listen is required when object event webhook is enabled")
	}
	if c.RedisAddress == "" {
		return fmt.Errorf("objectEventWebhook.redisAddress is required when object event webhook is enabled")
	}
	if c.RedisList == "" {
		return fmt.Errorf("objectEventWebhook.redisList is required when object event webhook is enabled")
	}
	if c.BucketsPath == "" {
		return fmt.Errorf("objectEventWebhook.bucketsPath is required when object event webhook is enabled")
	}

	return nil
}
