package routes

import (
	"context"
	"github.com/xiehqing/hiauthx/authorization"
	"github.com/xiehqing/hiauthx/db/queries"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/xiehqing/infra/pkg/hertzx"
)

func (r *Router) registerAuditRoutes(api *route.RouterGroup) {
	audits := api.Group("/audit-logs", r.CheckLogin())
	audits.GET("", r.listAuditLogs)
	audits.GET("/:id", r.getAuditLog)
}

func (r *Router) listAuditLogs(ctx context.Context, c *app.RequestContext) {
	startTime, endTime, ok := auditTimeRange(c)
	if !ok {
		return
	}
	req := authorization.ListAuditLogsRequest{
		AuditLogListFilter: queries.AuditLogListFilter{
			Pagination:   pagination(c),
			OperatorName: c.DefaultQuery("operatorName", ""),
			Module:       c.DefaultQuery("module", ""),
			Action:       c.DefaultQuery("action", ""),
			ResourceType: c.DefaultQuery("resourceType", ""),
			ResourceID:   queryInt64(c, "resourceId"),
			Status:       c.DefaultQuery("status", ""),
			StartTime:    startTime,
			EndTime:      endTime,
		},
	}
	data, err := r.service.ListAuditLogs(ctx, req)
	handleData(c, data, err)
}

func auditTimeRange(c *app.RequestContext) (*time.Time, *time.Time, bool) {
	startTime, ok := auditQueryTime(c, "startTime")
	if !ok {
		return nil, nil, false
	}
	endTime, ok := auditQueryTime(c, "endTime")
	if !ok {
		return nil, nil, false
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		hertzx.Badf(c, "开始时间不能晚于结束时间")
		return nil, nil, false
	}
	return startTime, endTime, true
}

func auditQueryTime(c *app.RequestContext, name string) (*time.Time, bool) {
	value := c.DefaultQuery(name, "")
	if value == "" {
		return nil, true
	}
	result, err := time.ParseInLocation(responseTimeFormat, value, time.Local)
	if err != nil {
		hertzx.Badf(c, "%s 格式不合法，请使用 yyyy-MM-dd HH:mm:ss", name)
		return nil, false
	}
	return &result, true
}

func (r *Router) getAuditLog(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := r.service.GetAuditLog(ctx, id)
	handleData(c, data, err)
}
