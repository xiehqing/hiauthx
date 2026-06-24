package routes

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/logs"
)

func (r *Router) registerAPIRoutes(api *route.RouterGroup) {
	items := api.Group("/apis", r.CheckLogin())
	items.GET("", r.listAPIs)
	items.POST("", r.createAPI)
	items.GET("/:id", r.getAPI)
	items.PUT("/:id", r.updateAPI)
	items.DELETE("/:id", r.deleteAPI)
}
func (r *Router) createAPI(ctx context.Context, c *app.RequestContext) {
	var req authorization.CreateAPIRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.CreateAPI(ctx, req)
	if err == nil {
		r.refreshAPICacheAfterMutation(ctx)
	}
	handleData(c, data, err)
}
func (r *Router) updateAPI(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req authorization.UpdateAPIRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ID = id
	data, err := r.service.UpdateAPI(ctx, req)
	if err == nil {
		r.refreshAPICacheAfterMutation(ctx)
	}
	handleData(c, data, err)
}
func (r *Router) getAPI(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetAPI(ctx, id)
	handleData(c, data, err)
}
func (r *Router) deleteAPI(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := r.service.DeleteAPI(ctx, id)
	if err == nil {
		r.refreshAPICacheAfterMutation(ctx)
	}
	handleMsg(c, "API 删除成功", err)
}
func (r *Router) listAPIs(ctx context.Context, c *app.RequestContext) {
	req := authorization.ListAPIsRequest{APIListFilter: queries.APIListFilter{Pagination: pagination(c), Method: c.DefaultQuery("method", ""), Module: c.DefaultQuery("module", ""), Action: c.DefaultQuery("action", ""), Status: queryIntPtr(c, "status")}}
	data, err := r.service.ListAPIs(ctx, req)
	handleData(c, data, err)
}

func (r *Router) refreshAPICacheAfterMutation(ctx context.Context) {
	if err := r.RefreshAPICache(ctx); err != nil {
		logs.Errorf("refresh API audit metadata cache failed: %v", err)
	}
}
