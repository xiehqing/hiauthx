package hitokenx

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/xiehqing/hitoken/core/adapter"
	"github.com/xiehqing/hitoken/storage/memory"
	hitokenredis "github.com/xiehqing/hitoken/storage/redis"
	"github.com/xiehqing/infra/pkg/logs"
)

const (
	defaultRedisPoolSize                = 10
	defaultRedisDialTimeoutSeconds      = 5
	defaultRedisReadTimeoutSeconds      = 3
	defaultRedisWriteTimeoutSeconds     = 3
	defaultRedisPoolTimeoutSeconds      = 4
	defaultRedisOperationTimeoutSeconds = 3
)

type StorageConfig struct {
	Type                    string `json:"type"`
	URL                     string `json:"url"`
	Host                    string `json:"host"`
	Port                    int    `json:"port"`
	Password                string `json:"password"`
	Database                int    `json:"database"`
	PoolSize                int    `json:"poolSize"`
	DialTimeoutSeconds      int    `json:"dialTimeoutSeconds"`
	ReadTimeoutSeconds      int    `json:"readTimeoutSeconds"`
	WriteTimeoutSeconds     int    `json:"writeTimeoutSeconds"`
	PoolTimeoutSeconds      int    `json:"poolTimeoutSeconds"`
	OperationTimeoutSeconds int    `json:"operationTimeoutSeconds"`
}

func NewStorage(value string) adapter.Storage {
	storage, err := newStorage(value)
	if err != nil {
		logs.Errorf("storage.New: build token storage failed.")
		return memory.NewStorage()
	}
	return storage
}

func newStorage(value string) (adapter.Storage, error) {
	if strings.TrimSpace(value) == "" {
		return memory.NewStorage(), nil
	}

	var config StorageConfig
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(config.Type)) {
	case "", "memory":
		return memory.NewStorage(), nil
	case "redis":
		return newRedisStorage(config)
	default:
		return memory.NewStorage(), nil
	}
}

func newRedisStorage(config StorageConfig) (adapter.Storage, error) {
	if strings.TrimSpace(config.URL) != "" {
		return hitokenredis.NewStorage(config.URL)
	}
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port <= 0 {
		config.Port = 6379
	}
	if config.PoolSize <= 0 {
		config.PoolSize = defaultRedisPoolSize
	}

	return hitokenredis.NewStorageFromConfig(&hitokenredis.Config{
		Host:             config.Host,
		Port:             config.Port,
		Password:         config.Password,
		Database:         config.Database,
		PoolSize:         config.PoolSize,
		DialTimeout:      secondsOrDefault(config.DialTimeoutSeconds, defaultRedisDialTimeoutSeconds),
		ReadTimeout:      secondsOrDefault(config.ReadTimeoutSeconds, defaultRedisReadTimeoutSeconds),
		WriteTimeout:     secondsOrDefault(config.WriteTimeoutSeconds, defaultRedisWriteTimeoutSeconds),
		PoolTimeout:      secondsOrDefault(config.PoolTimeoutSeconds, defaultRedisPoolTimeoutSeconds),
		OperationTimeout: secondsOrDefault(config.OperationTimeoutSeconds, defaultRedisOperationTimeoutSeconds),
	})
}

func secondsOrDefault(value, defaultValue int) time.Duration {
	if value <= 0 {
		value = defaultValue
	}
	return time.Duration(value) * time.Second
}
