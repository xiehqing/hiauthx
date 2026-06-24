package routes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xiehqing/hitoken/htputil"
	"github.com/xiehqing/infra/pkg/hertzx"
)

const (
	CtxKeyOfUser            = "user"
	CtxKeyOfUserID          = "userId"
	CtxKeyOfUserName        = "username"
	CtxKeyOfIsSystemManager = "isSystemManager"
)

const defaultSystemRole = "role_admin"

func (r *Router) CheckLogin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		token := normalizeAuthorizationToken(authorizationToken(c))
		if token == "" {
			hertzx.Unauthorized(c, "请先登录")
			return
		}
		if err := htputil.CheckLogin(token); err != nil {
			hertzx.Unauthorized(c, "登录状态已失效，请重新登录")
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			hertzx.Unauthorized(c, "未获取到当前登录用户信息，请重新登录")
			return
		}
		user, err := r.service.GetUser(ctx, userID)
		if err != nil {
			hertzx.Unauthorized(c, "获取当前登录用户信息失败，请重新登录")
			return
		}
		if user == nil {
			hertzx.Unauthorized(c, "未获取到当前登录用户，请重新登录")
			return
		}
		c.Set(CtxKeyOfUser, user)
		c.Set(CtxKeyOfUserID, userID)
		c.Set(CtxKeyOfUserName, user.Username)
		c.Set(CtxKeyOfIsSystemManager, isSystemManager(user))
		c.Next(ctx)
	}
}

func (r *Router) auditContext() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		requestID := string(c.GetHeader("X-Request-Id"))
		if requestID == "" {
			requestID = newRequestID()
		}

		value := audit.Context{
			RequestID: requestID,
			Method:    string(c.Method()),
			Path:      string(c.Path()),
			IP:        c.ClientIP(),
			UserAgent: string(c.UserAgent()),
		}
		routePath := c.FullPath()
		if routePath == "" {
			routePath = value.Path
		}
		if apiInfo, ok := r.apiCache.get(value.Method, routePath); ok {
			value.Module = apiInfo.Module
			value.Action = apiInfo.Action
			value.Description = apiInfo.Description
			value.ResourceType = apiInfo.ResourceType
		}

		token := normalizeAuthorizationToken(authorizationToken(c))
		if token != "" {
			if loginID, err := htputil.GetLoginID(token); err == nil {
				if userID, err := strconv.ParseInt(loginID, 10, 64); err == nil {
					value.OperatorID = userID
					if user, err := r.q.GetUser(ctx, userID); err == nil && user != nil {
						value.OperatorName = user.Username
					}
				}
			}
		} else if string(c.Path()) == "/api/v1/auth/login" {
			value.OperatorName = loginUsername(c)
		}

		auditCtx := audit.WithContext(ctx, value)
		c.Next(auditCtx)
		r.recordPlainRequestAudit(auditCtx, c, time.Since(start).Milliseconds())
	}
}

func (r *Router) recordPlainRequestAudit(ctx context.Context, c *app.RequestContext, durationMs int64) {
	method := string(c.Method())
	path := string(c.Path())
	if skipPlainRequestAudit(method, path, c.Response.StatusCode()) {
		return
	}

	request := requestAuditFromRoute(method, path)
	if value, ok := audit.FromContext(ctx); ok {
		if value.Module != "" {
			request.Module = value.Module
		}
		if value.Action != "" {
			request.Action = value.Action
		}
		if value.Description != "" {
			request.Description = value.Description
		}
		if value.ResourceType != "" {
			request.ResourceType = value.ResourceType
		}
	}
	request.DurationMs = durationMs
	statusCode := c.Response.StatusCode()
	if statusCode >= 400 {
		request.Status = entity.AuditStatusFail
		request.ErrorMessage = fmt.Sprintf("HTTP %d", statusCode)
	} else {
		request.Status = entity.AuditStatusSuccess
	}
	_ = r.q.CreateRequestAudit(ctx, request)
}

func skipPlainRequestAudit(method, path string, statusCode int) bool {
	if path == "/api/v1/health" || strings.HasPrefix(path, "/api/v1/audit-logs") {
		return true
	}
	if method == "GET" {
		return false
	}
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/logout" {
		return false
	}
	return statusCode < 400
}

func requestAuditFromRoute(method, path string) queries.RequestAudit {
	request := queries.RequestAudit{
		Action:       entity.AuditOperationQuery,
		ResourceType: "request",
	}
	switch {
	case path == "/api/v1/auth/login":
		request.Module = "认证管理"
		request.Action = entity.AuditOperationLogin
		request.ResourceType = "auth"
		request.Description = "用户登录"
	case path == "/api/v1/auth/logout":
		request.Module = "认证管理"
		request.Action = entity.AuditOperationLogout
		request.ResourceType = "auth"
		request.Description = "退出登录"
	case strings.HasPrefix(path, "/api/v1/auth/me"):
		request.Module = "认证管理"
		request.ResourceType = "auth"
		request.Description = "查询当前登录用户"
	case strings.HasPrefix(path, "/api/v1/users"):
		request.Module = "用户管理"
		request.ResourceType = "user"
		request.Description = plainRequestDescription(method, "用户")
	case strings.HasPrefix(path, "/api/v1/roles"):
		request.Module = "角色管理"
		request.ResourceType = "role"
		request.Description = plainRequestDescription(method, "角色")
	case strings.HasPrefix(path, "/api/v1/departments"):
		request.Module = "部门管理"
		request.ResourceType = "department"
		request.Description = plainRequestDescription(method, "部门")
	case strings.HasPrefix(path, "/api/v1/menus"):
		request.Module = "菜单管理"
		request.ResourceType = "menu"
		request.Description = plainRequestDescription(method, "菜单")
	case strings.HasPrefix(path, "/api/v1/system-configs"):
		request.Module = "系统配置"
		request.ResourceType = "system_config"
		request.Description = plainRequestDescription(method, "系统配置")
	default:
		request.Module = "系统"
		request.Description = fmt.Sprintf("访问接口：%s", path)
	}
	if method != "GET" && request.Action == entity.AuditOperationQuery {
		request.Action = strings.ToLower(method)
	}
	return request
}

func plainRequestDescription(method, resourceName string) string {
	if method == "GET" {
		return "查询" + resourceName
	}
	return "操作" + resourceName
}

func newRequestID() string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(random))
}

func loginUsername(c *app.RequestContext) string {
	var req struct {
		Username string `json:"username"`
	}
	body, err := c.Body()
	if err != nil || len(body) == 0 {
		return ""
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return strings.TrimSpace(req.Username)
}

func normalizeAuthorizationToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token)
}

func authorizationToken(c *app.RequestContext) string {
	return string(c.GetHeader("Authorization"))
}

// currentUserID 从登录令牌中解析当前用户 ID。
func currentUserID(c *app.RequestContext) (int64, bool) {
	token := normalizeAuthorizationToken(authorizationToken(c))
	if token == "" {
		return 0, false
	}
	loginID, err := htputil.GetLoginID(token)
	if err != nil {
		return 0, false
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(loginID), 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}

// isSystemManager 判断用户是否具备系统级空间管理权限。
func isSystemManager(user *entity.User) bool {
	if user == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(user.Username), "admin") {
		return true
	}
	for _, role := range user.Roles {
		if isSystemAdminRole(role) {
			return true
		}
	}
	return false
}

// isSystemAdminRole 判断角色是否为系统管理员角色。
func isSystemAdminRole(role entity.Role) bool {
	normalized := strings.ToLower(strings.TrimSpace(role.Name))
	return normalized == defaultSystemRole
}
