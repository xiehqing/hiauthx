package routes

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/xiehqing/hiauthx/authentication"
)

func (r *Router) registerAuthenticationRoutes(api *route.RouterGroup) {
	auth := api.Group("/auth")
	auth.GET("/encrypt-config", r.encryptConfig)
	auth.POST("/login", r.login)
	auth.POST("/logout", checkLogin(), r.logout)
	auth.GET("/me", checkLogin(), r.currentUser)
}

func (r *Router) encryptConfig(ctx context.Context, c *app.RequestContext) {
	data, err := r.authentication.GetEncryptConfig(ctx)
	handleData(c, data, err)
}

func (r *Router) login(ctx context.Context, c *app.RequestContext) {
	var req authentication.LoginRequest
	if !bindJSON(c, &req) {
		return
	}

	data, err := r.authentication.Login(ctx, req)
	handleData(c, data, err)
}

func (r *Router) logout(ctx context.Context, c *app.RequestContext) {
	handleMsg(c, "退出登录成功", r.authentication.Logout(ctx, authorizationToken(c)))
}

func (r *Router) currentUser(ctx context.Context, c *app.RequestContext) {
	data, err := r.authentication.CurrentUser(ctx, authorizationToken(c))
	handleData(c, data, err)
}
