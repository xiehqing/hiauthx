package config

import (
	"github.com/xiehqing/infra/pkg/hertzx"
	"github.com/xiehqing/infra/pkg/logs"
	"github.com/xiehqing/infra/pkg/ormx"
)

// Config 服务配置
type Config struct {
	Server hertzx.WebConfig `json:"server" yaml:"server" mapstructure:"server"`
	DB     ormx.DBConfig    `json:"db" yaml:"db" mapstructure:"db"`
	Log    logs.LogConfig   `json:"log" yaml:"log" mapstructure:"log"`
}
