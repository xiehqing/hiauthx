package routes

import (
	"context"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
)

func (r *Router) registerSystemConfigRoutes(api *route.RouterGroup) {
	api.GET("/system-configs/site-settings", r.getSiteSettings)
	configs := api.Group("/system-configs", r.CheckLogin())
	configs.GET("", r.listSystemConfigs)
	configs.POST("", r.createSystemConfig)
	configs.GET("/enabled", r.listEnabledSystemConfigs)
	configs.GET("/enabled-map", r.getEnabledSystemConfigMap)
	configs.GET("/system-settings", r.getSystemSettings)
	configs.PUT("/batch", r.batchSaveSystemConfigs)
	configs.GET("/by-key/:key", r.getSystemConfigByKey)
	configs.GET("/:id", r.getSystemConfig)
	configs.PUT("/:id", r.updateSystemConfig)
	configs.DELETE("/:id", r.deleteSystemConfig)
}

func (r *Router) createSystemConfig(ctx context.Context, c *app.RequestContext) {
	var req authorization.CreateSystemConfigRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.CreateSystemConfig(ctx, req)
	handleData(c, data, err)
}

func (r *Router) updateSystemConfig(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req authorization.UpdateSystemConfigRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ID = id
	data, err := r.service.UpdateSystemConfig(ctx, req)
	handleData(c, data, err)
}

func (r *Router) batchSaveSystemConfigs(ctx context.Context, c *app.RequestContext) {
	var req authorization.BatchSaveSystemConfigsRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.BatchSaveSystemConfigs(ctx, req)
	handleData(c, data, err)
}

func (r *Router) getSystemConfig(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetSystemConfig(ctx, id)
	handleData(c, data, err)
}

func (r *Router) getSystemConfigByKey(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	data, err := r.service.GetSystemConfigByKey(ctx, key)
	handleData(c, data, err)
}

func (r *Router) listSystemConfigs(ctx context.Context, c *app.RequestContext) {
	req := authorization.ListSystemConfigsRequest{
		SystemConfigListFilter: queries.SystemConfigListFilter{
			Pagination: pagination(c),
			Group:      c.DefaultQuery("group", ""),
			Category:   c.DefaultQuery("category", ""),
			Enabled:    queryIntPtr(c, "enabled"),
		},
	}
	data, err := r.service.ListSystemConfigs(ctx, req)
	handleData(c, data, err)
}

func (r *Router) listEnabledSystemConfigs(ctx context.Context, c *app.RequestContext) {
	data, err := r.service.ListEnabledSystemConfigs(ctx, c.DefaultQuery("group", ""), c.DefaultQuery("category", ""))
	handleData(c, data, err)
}

func (r *Router) getEnabledSystemConfigMap(ctx context.Context, c *app.RequestContext) {
	data, err := r.service.GetEnabledSystemConfigMap(ctx, c.DefaultQuery("group", ""), c.DefaultQuery("category", ""))
	handleData(c, data, err)
}

func (r *Router) getSystemSettings(ctx context.Context, c *app.RequestContext) {
	data, err := r.service.GetEnabledSystemConfigMap(ctx, c.DefaultQuery("group", ""), entity.SystemConfigCategorySystem)
	handleData(c, data, err)
}

func (r *Router) getSiteSettings(ctx context.Context, c *app.RequestContext) {
	data, err := r.service.GetEnabledSystemConfigMap(ctx, c.DefaultQuery("group", ""), entity.SystemConfigCategorySite)
	handleData(c, data, err)
}

func (r *Router) deleteSystemConfig(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	handleMsg(c, "系统配置删除成功", r.service.DeleteSystemConfig(ctx, id))
}
