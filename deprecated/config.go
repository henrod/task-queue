package deprecated

import (
	"github.com/garyburd/redigo/redis"
)

type EnqueueOptions struct {
	RetryCount        int               `json:"retry_count,omitempty"`
	Retry             bool              `json:"retry,omitempty"`
	RetryMax          int               `json:"retry_max,omitempty"`
	At                float64           `json:"at,omitempty"`
	RetryOptions      RetryOptions      `json:"retry_options,omitempty"`
	ConnectionOptions map[string]string `json:"connection_options,omitempty"`
}

type RetryOptions struct {
	Exp      int `json:"exp"`
	MinDelay int `json:"min_delay"`
	MaxDelay int `json:"max_delay"`
	MaxRand  int `json:"max_rand"`
}

type config struct {
	Pool *redis.Pool
}
var Config *config
