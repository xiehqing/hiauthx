package routes

import (
	"context"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
)

func (r *Router) registerDepartmentRoutes(api *route.RouterGroup) {
	departments := api.Group("/departments", checkLogin())
	departments.GET("", r.listDepartments)
	departments.GET("/options", r.listDepartmentOptions)
	departments.POST("", r.createDepartment)
	departments.GET("/:id", r.getDepartment)
	departments.PUT("/:id", r.updateDepartment)
	departments.DELETE("/:id", r.deleteDepartment)
}

func (r *Router) createDepartment(ctx context.Context, c *app.RequestContext) {
	var req authorization.CreateDepartmentRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.CreateDepartment(ctx, req)
	handleData(c, data, err)
}

func (r *Router) updateDepartment(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req authorization.UpdateDepartmentRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ID = id
	data, err := r.service.UpdateDepartment(ctx, req)
	handleData(c, data, err)
}

func (r *Router) getDepartment(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetDepartment(ctx, id)
	handleData(c, data, err)
}

func (r *Router) listDepartments(ctx context.Context, c *app.RequestContext) {
	req := authorization.ListDepartmentsRequest{
		DepartmentListFilter: queries.DepartmentListFilter{
			Keyword:  c.DefaultQuery("keyword", ""),
			ParentID: queryInt64Ptr(c, "parentId"),
		},
	}
	data, err := r.service.ListDepartments(ctx, req)
	handleData(c, data, err)
}

func (r *Router) listDepartmentOptions(ctx context.Context, c *app.RequestContext) {
	data, err := r.service.ListAllDepartments(ctx)
	handleData(c, data, err)
}

func (r *Router) deleteDepartment(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	handleMsg(c, "部门删除成功", r.service.DeleteDepartment(ctx, id))
}
