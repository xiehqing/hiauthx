package routes

import (
	"context"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
)

func (r *Router) registerUserRoutes(api *route.RouterGroup) {
	users := api.Group("/users", checkLogin())
	users.GET("", r.listUsers)
	users.POST("", r.createUser)
	users.GET("/:id", r.getUser)
	users.PUT("/:id", r.updateUser)
	users.DELETE("/:id", r.deleteUser)
}

func (r *Router) createUser(ctx context.Context, c *app.RequestContext) {
	var req authorization.CreateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := r.service.CreateUser(ctx, req)
	handleData(c, data, err)
}

func (r *Router) updateUser(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req authorization.UpdateUserRequest
	if !bindJSON(c, &req) {
		return
	}
	req.ID = id
	data, err := r.service.UpdateUser(ctx, req)
	handleData(c, data, err)
}

func (r *Router) getUser(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetUser(ctx, id)
	handleData(c, data, err)
}

func (r *Router) listUsers(ctx context.Context, c *app.RequestContext) {
	req := authorization.ListUsersRequest{
		UserListFilter: queries.UserListFilter{
			Pagination:   pagination(c),
			Status:       queryIntPtr(c, "status"),
			RoleID:       queryInt64(c, "roleId"),
			DepartmentID: queryInt64(c, "departmentId"),
		},
	}
	data, err := r.service.ListUsers(ctx, req)
	handleData(c, data, err)
}

func (r *Router) deleteUser(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	handleMsg(c, "用户删除成功", r.service.DeleteUser(ctx, id))
}
