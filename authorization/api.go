package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/ormx"
)

type CreateAPIRequest struct {
	Name         string `json:"name"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Description  string `json:"description"`
	ResourceType string `json:"resourceType"`
	Status       *int   `json:"status"`
	Sort         int    `json:"sort"`
	Operator     string `json:"operator"`
}
type UpdateAPIRequest struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Description  string `json:"description"`
	ResourceType string `json:"resourceType"`
	Status       int    `json:"status"`
	Sort         int    `json:"sort"`
	Operator     string `json:"operator"`
}
type ListAPIsRequest struct{ queries.APIListFilter }

func (as *Service) CreateAPI(ctx context.Context, req CreateAPIRequest) (*entity.API, error) {
	status := entity.APIStatusEnabled
	if req.Status != nil {
		status = *req.Status
	}
	method, path, resource := normalizeAPI(req.Method, req.Path, req.ResourceType)
	if err := validateAPI(req.Name, method, path, req.Module, req.Action, status); err != nil {
		return nil, err
	}
	item := &entity.API{BaseModel: ormx.BaseModel{CreatedBy: normalizeString(req.Operator), UpdatedBy: normalizeString(req.Operator)}, Name: normalizeString(req.Name), Method: method, Path: path, Module: normalizeString(req.Module), Action: normalizeString(req.Action), Description: normalizeString(req.Description), ResourceType: resource, Status: status, Sort: req.Sort}
	if err := as.queries.CreateAPI(ctx, item); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetAPI(ctx, item.ID)
}

func (as *Service) UpdateAPI(ctx context.Context, req UpdateAPIRequest) (*entity.API, error) {
	if req.ID <= 0 {
		return nil, ErrInvalidID
	}
	method, path, resource := normalizeAPI(req.Method, req.Path, req.ResourceType)
	if err := validateAPI(req.Name, method, path, req.Module, req.Action, req.Status); err != nil {
		return nil, err
	}
	item := &entity.API{BaseModel: ormx.BaseModel{ID: req.ID, UpdatedBy: normalizeString(req.Operator)}, Name: normalizeString(req.Name), Method: method, Path: path, Module: normalizeString(req.Module), Action: normalizeString(req.Action), Description: normalizeString(req.Description), ResourceType: resource, Status: req.Status, Sort: req.Sort}
	if err := as.queries.UpdateAPI(ctx, item); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetAPI(ctx, req.ID)
}

func (as *Service) GetAPI(ctx context.Context, id int64) (*entity.API, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	item, err := as.queries.GetAPI(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	return item, nil
}
func (as *Service) ListAPIs(ctx context.Context, req ListAPIsRequest) (ormx.PageResult[entity.API], error) {
	return as.queries.ListAPIs(ctx, req.APIListFilter)
}
func (as *Service) DeleteAPI(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	return normalizeDBError(as.queries.DeleteAPI(ctx, id))
}

func normalizeAPI(method, path, resource string) (string, string, string) {
	method = strings.ToUpper(normalizeString(method))
	path = normalizeString(path)
	resource = normalizeString(resource)
	if resource == "" {
		resource = "request"
	}
	return method, path, resource
}
func validateAPI(name, method, path, module, action string, status int) error {
	if !required(name) || !required(module) || !required(action) {
		return fmt.Errorf("%w: name、module 和 action 不能为空", ErrInvalidArgument)
	}
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
	default:
		return fmt.Errorf("%w: HTTP method 不合法", ErrInvalidArgument)
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: path 必须以 / 开头", ErrInvalidArgument)
	}
	if status != entity.APIStatusDisabled && status != entity.APIStatusEnabled {
		return fmt.Errorf("%w: status 不合法", ErrInvalidArgument)
	}
	return nil
}
