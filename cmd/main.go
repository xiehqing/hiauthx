package main

import (
	"context"
	"github.com/alecthomas/kingpin/v2"
	"github.com/pkg/errors"
	"github.com/xiehqing/hiauthx/cmd/config"
	"github.com/xiehqing/hiauthx/routes"
	"github.com/xiehqing/infra/pkg/cfgx"
	"github.com/xiehqing/infra/pkg/hertzx"
	"github.com/xiehqing/infra/pkg/logs"
	"github.com/xiehqing/infra/pkg/ormx"
)

var (
	configDir  = kingpin.Flag("config-dir", "配置文件目录.(env:CONFIG_DIR)").Default("./etc").Envar("CONFIG_DIR").String()
	configFile = kingpin.Flag("config-file", "配置文件名称.(env:CONFIG_FILE)").Default("config.yml").Envar("CONFIG_FILE").String()
	configType = kingpin.Flag("config-type", "配置文件类型.(env:CONFIG_TYPE)").Default("yml").Envar("CONFIG_TYPE").String()
	timeZone   = kingpin.Flag("time-zone", "配置文件类型.(env:TZ)").Default("Asia/Shanghai").Envar("TZ").String()
)

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()
	var config *config.Config
	err := cfgx.LoadConfig(*configDir, *configFile, *configType, &config)
	if err != nil {
		logs.Fatalf("加载配置文件失败, err: %s", err)
	}
	err = Initialize(config)
	if err != nil {
		logs.Fatalf("服务初始化失败, err: %s", err)
	}
}

func Initialize(cfg *config.Config) error {
	err := logs.InitLogger(cfg.Log, "imh-server.log")
	if err != nil {
		return errors.WithMessagef(err, "初始化日志失败")
	}
	dbClient, err := ormx.NewDBClient(cfg.DB)
	if err != nil {
		return errors.WithMessagef(err, "初始化数据库失败")
	}
	// 初始化路由
	routes := routes.New(dbClient)
	routes.RefreshTokenManager(context.Background())
	webEngine := hertzx.WebEngine(cfg.Server)
	routes.Init(webEngine)
	hertzx.StartWebServer(webEngine)
	return nil
}
