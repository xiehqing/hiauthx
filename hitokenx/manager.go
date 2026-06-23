package hitokenx

import (
	"context"
	"github.com/xiehqing/hiauthx/configx"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/hitoken/core"
	"github.com/xiehqing/hitoken/htputil"
	"github.com/xiehqing/infra/pkg/logs"
)

func RefreshManager(ctx context.Context, q *queries.Queries) {
	config := configx.New(q)
	manager := core.NewBuilder().
		Storage(NewStorage(
			config.String(ctx, entity.SecurityTokenStorageType, ""),
			config.String(ctx, entity.SecurityTokenRedisConfig, ""),
			config.String(ctx, entity.SecurityTokenStorage, ""),
		)).
		TokenName("accessToken").
		AutoRenew(true).
		Timeout(config.Int64(ctx, entity.SecurityTokenExpireMinutes, 1440) * 60).
		TokenStyle(core.TokenStyleJWT).
		JwtSecretKey(config.String(ctx, entity.SecurityTokenJwtSecretKey, "www.zorktech.com")).
		IsConcurrent(config.Bool(ctx, entity.SecurityLoginConcurrentEnable, false)).
		IsPrintBanner(false).
		IsLog(false).
		Build()
	// 监听所有事件（通配符）
	manager.RegisterFunc(core.EventAll, func(data *core.EventData) {
		logs.Infof("[%s] %s", data.Event, data.LoginID)
	})
	htputil.SetManager(manager)
}
