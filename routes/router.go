package routes

import (
	"context"
	"errors"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/authentication"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/hiauthx/hitokenx"
	"github.com/xiehqing/infra/pkg/logs"
	"github.com/xiehqing/infra/pkg/ormx"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/xiehqing/infra/pkg/hertzx"
	"gorm.io/gorm"
)

type Router struct {
	q              *queries.Queries
	service        *authorization.Service
	authentication *authentication.Service
	apiCache       *apiMetadataCache
}

func New(db *gorm.DB) *Router {
	audit := audit.NewAudit(db)
	q := queries.New(db, audit)
	q.AutoMigrate(context.Background())
	router := &Router{
		q:              q,
		service:        authorization.New(q),
		authentication: authentication.New(q),
		apiCache:       newAPIMetadataCache(),
	}
	if err := router.RefreshAPICache(context.Background()); err != nil {
		logs.Errorf("initialize API audit metadata cache failed: %v", err)
	}
	return router
}

func (r *Router) Authentication() *authentication.Service {
	return r.authentication
}

func (r *Router) Authorization() *authorization.Service {
	return r.service
}

func (r *Router) RefreshTokenManager(ctx context.Context) {
	hitokenx.RefreshManager(ctx, r.q)
}

func (r *Router) Init(server *server.Hertz) {
	api := server.Group("/api/v1")
	api.Use(r.auditContext())
	api.GET("/health", health)

	r.registerAuthenticationRoutes(api)
	r.registerUserRoutes(api)
	r.registerRoleRoutes(api)
	r.registerDepartmentRoutes(api)
	r.registerMenuRoutes(api)
	r.registerSystemConfigRoutes(api)
	r.registerAPIRoutes(api)
	r.registerAuditRoutes(api)
}

func (r *Router) RegisterRoutes(api *route.RouterGroup) {
	api.Use(r.auditContext())
	api.GET("/health", health)
	r.registerAuthenticationRoutes(api)
	r.registerUserRoutes(api)
	r.registerRoleRoutes(api)
	r.registerDepartmentRoutes(api)
	r.registerMenuRoutes(api)
	r.registerSystemConfigRoutes(api)
	r.registerAPIRoutes(api)
	r.registerAuditRoutes(api)
}

func bindJSON(c *app.RequestContext, req any) bool {
	if err := c.BindJSON(req); err != nil {
		hertzx.Badf(c, "请求体解析失败: %v", err)
		return false
	}
	return true
}

func pathID(c *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		hertzx.Badf(c, "参数 id 不合法")
		return 0, false
	}
	return id, true
}

func handleData(c *app.RequestContext, data any, err error) {
	if err != nil {
		handleError(c, err)
		return
	}
	hertzx.Data(c, formatResponseData(data))
}

func handleMsg(c *app.RequestContext, message string, err error) {
	if err != nil {
		handleError(c, err)
		return
	}
	hertzx.Msg(c, message)
}

func handleError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, authorization.ErrInvalidID),
		errors.Is(err, authorization.ErrInvalidArgument),
		errors.Is(err, authorization.ErrNotFound),
		errors.Is(err, authorization.ErrBuiltInRole),
		errors.Is(err, authentication.ErrInvalidLogin),
		errors.Is(err, authentication.ErrInvalidRSAKey),
		errors.Is(err, authentication.ErrUserDisabled),
		errors.Is(err, authentication.ErrUserLocked),
		errors.Is(err, authentication.ErrTokenRequired),
		errors.Is(err, authentication.ErrInvalidToken),
		errors.Is(err, authentication.ErrInvalidArgument):
		hertzx.Badf(c, "%v", err)
	default:
		hertzx.Errorf(c, "系统异常，请联系管理员")
	}
}

func pagination(c *app.RequestContext) ormx.Pagination {
	return ormx.Pagination{
		Keyword:   c.DefaultQuery("keyword", ""),
		PageNo:    queryInt(c, "pageNo"),
		PageSize:  queryInt(c, "pageSize"),
		SortField: c.DefaultQuery("sortField", ""),
		SortOrder: c.DefaultQuery("sortOrder", ""),
	}
}

func queryInt(c *app.RequestContext, name string) int {
	result, err := hertzx.QueryInt(c, name)
	if err != nil {
		return 0
	}
	return result
}

func queryIntPtr(c *app.RequestContext, name string) *int {
	result, err := hertzx.QueryIntPtr(c, name)
	if err != nil {
		return nil
	}
	return result
}

func queryInt64(c *app.RequestContext, name string) int64 {
	result, err := hertzx.QueryInt64(c, name)
	if err != nil {
		return 0
	}
	return result
}

func queryInt64Ptr(c *app.RequestContext, name string) *int64 {
	result, err := hertzx.QueryInt64Ptr(c, name)
	if err != nil {
		return nil
	}
	return result
}

func health(ctx context.Context, c *app.RequestContext) {
	hertzx.Data(c, map[string]string{"status": "ok"})
}
