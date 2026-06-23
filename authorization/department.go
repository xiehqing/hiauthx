package authorization

import (
	"context"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/ormx"
)

type CreateDepartmentRequest struct {
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
	Operator string `json:"operator"`
}

type UpdateDepartmentRequest struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
	Operator string `json:"operator"`
}

type ListDepartmentsRequest struct {
	queries.DepartmentListFilter
}

type DepartmentTree struct {
	entity.Department
	Children []DepartmentTree `json:"children"`
}

func (as *Service) CreateDepartment(ctx context.Context, req CreateDepartmentRequest) (*entity.Department, error) {
	if !required(req.Name) {
		return nil, fmt.Errorf("%w: 部门名称不能为空", ErrInvalidArgument)
	}
	if req.ParentID < 0 {
		return nil, fmt.Errorf("%w: 上级部门不合法", ErrInvalidArgument)
	}
	name := normalizeString(req.Name)
	if err := as.validateDepartmentNameUnique(ctx, req.ParentID, name, 0); err != nil {
		return nil, err
	}

	department := &entity.Department{
		BaseModel: ormx.BaseModel{
			CreatedBy: normalizeString(req.Operator),
			UpdatedBy: normalizeString(req.Operator),
		},
		ParentID: req.ParentID,
		Name:     name,
		Sort:     req.Sort,
	}

	if err := as.queries.CreateDepartment(ctx, department); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.queries.GetDepartment(ctx, department.ID)
}

func (as *Service) UpdateDepartment(ctx context.Context, req UpdateDepartmentRequest) (*entity.Department, error) {
	if req.ID <= 0 {
		return nil, ErrInvalidID
	}
	if req.ParentID < 0 || req.ParentID == req.ID {
		return nil, fmt.Errorf("%w: 上级部门不合法", ErrInvalidArgument)
	}
	if !required(req.Name) {
		return nil, fmt.Errorf("%w: 部门名称不能为空", ErrInvalidArgument)
	}
	name := normalizeString(req.Name)
	if err := as.validateDepartmentNameUnique(ctx, req.ParentID, name, req.ID); err != nil {
		return nil, err
	}

	department := &entity.Department{
		BaseModel: ormx.BaseModel{
			ID:        req.ID,
			UpdatedBy: normalizeString(req.Operator),
		},
		ParentID: req.ParentID,
		Name:     name,
		Sort:     req.Sort,
	}

	if err := as.queries.UpdateDepartment(ctx, department); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetDepartment(ctx, req.ID)
}

func (as *Service) GetDepartment(ctx context.Context, id int64) (*entity.Department, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	department, err := as.queries.GetDepartment(ctx, id)
	if err != nil {
		return nil, err
	}
	if department == nil {
		return nil, ErrNotFound
	}
	return department, nil
}

func (as *Service) ListDepartments(ctx context.Context, req ListDepartmentsRequest) ([]DepartmentTree, error) {
	req.Keyword = normalizeString(req.Keyword)
	departments, err := as.queries.ListDepartments(ctx, req.DepartmentListFilter)
	if err != nil {
		return nil, err
	}
	if req.Keyword != "" || req.ParentID != nil {
		allDepartments, err := as.queries.ListAllDepartments(ctx)
		if err != nil {
			return nil, err
		}
		departments = withDepartmentAncestors(departments, allDepartments)
	}
	return buildDepartmentTree(departments), nil
}

func (as *Service) ListAllDepartments(ctx context.Context) ([]DepartmentTree, error) {
	departments, err := as.queries.ListAllDepartments(ctx)
	if err != nil {
		return nil, err
	}
	return buildDepartmentTree(departments), nil
}

func (as *Service) DeleteDepartment(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	return normalizeDBError(as.queries.DeleteDepartment(ctx, id))
}

func (as *Service) validateDepartmentNameUnique(ctx context.Context, parentID int64, name string, excludeID int64) error {
	exists, err := as.queries.DepartmentNameExists(ctx, parentID, name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: 同级部门名称已存在", ErrInvalidArgument)
	}
	return nil
}

func buildDepartmentTree(departments []entity.Department) []DepartmentTree {
	childrenByParent := make(map[int64][]entity.Department)
	for _, department := range departments {
		childrenByParent[department.ParentID] = append(childrenByParent[department.ParentID], department)
	}

	var build func(parentID int64) []DepartmentTree
	build = func(parentID int64) []DepartmentTree {
		children := childrenByParent[parentID]
		result := make([]DepartmentTree, 0, len(children))
		for _, department := range children {
			result = append(result, DepartmentTree{
				Department: department,
				Children:   build(department.ID),
			})
		}
		return result
	}

	return build(0)
}

func withDepartmentAncestors(matched []entity.Department, all []entity.Department) []entity.Department {
	byID := make(map[int64]entity.Department, len(all))
	for _, department := range all {
		byID[department.ID] = department
	}

	selected := make(map[int64]entity.Department, len(matched))
	for _, department := range matched {
		selected[department.ID] = department
		for parentID := department.ParentID; parentID > 0; {
			parent, ok := byID[parentID]
			if !ok {
				break
			}
			selected[parent.ID] = parent
			parentID = parent.ParentID
		}
	}

	result := make([]entity.Department, 0, len(selected))
	for _, department := range all {
		if item, ok := selected[department.ID]; ok {
			result = append(result, item)
		}
	}
	return result
}
