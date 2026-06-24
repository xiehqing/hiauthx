package routes

import (
	"context"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
)

func (r *Router) registerMenuRoutes(api *route.RouterGroup) {
	menus := api.Group("/menus", r.CheckLogin())
	menus.GET("", r.listMenus)
	menus.POST("", r.createMenu)
	menus.GET("/:id", r.getMenu)
	menus.PUT("/:id", r.updateMenu)
	menus.DELETE("/:id", r.deleteMenu)
}

func (r *Router) createMenu(ctx context.Context, c *app.RequestContext) {
	var req authorization.CreateMenuRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.CreateMenu(ctx, req)
	handleData(c, data, err)
}

func (r *Router) updateMenu(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req authorization.UpdateMenuRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ID = id
	data, err := r.service.UpdateMenu(ctx, req)
	handleData(c, data, err)
}

func (r *Router) getMenu(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetMenu(ctx, id)
	handleData(c, data, err)
}

func (r *Router) listMenus(ctx context.Context, c *app.RequestContext) {
	req := authorization.ListMenusRequest{
		MenuListFilter: queries.MenuListFilter{
			Keyword:  c.DefaultQuery("keyword", ""),
			Type:     queryIntPtr(c, "type"),
			ParentID: queryInt64Ptr(c, "parentId"),
			Show:     queryIntPtr(c, "show"),
		},
	}
	data, err := r.service.ListMenus(ctx, req)
	handleData(c, data, err)
}

func (r *Router) deleteMenu(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	handleMsg(c, "菜单删除成功", r.service.DeleteMenu(ctx, id))
}
