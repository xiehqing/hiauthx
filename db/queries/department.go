package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"
	"gorm.io/gorm"
)

type DepartmentListFilter struct {
	Keyword  string `json:"keyword" form:"keyword"`
	ParentID *int64 `json:"parentId" form:"parentId"`
}

func (q *Queries) CreateDepartment(ctx context.Context, department *entity.Department) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(department).Error
	})
}

func (q *Queries) UpdateDepartment(ctx context.Context, department *entity.Department) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Model(&entity.Department{}).
			Where("id = ?", department.ID).
			Select("ParentID", "Name", "Sort", "UpdatedBy").
			Updates(department).
			Error
	})
}

func (q *Queries) GetDepartment(ctx context.Context, id int64) (*entity.Department, error) {
	var department entity.Department
	err := q.db.WithContext(ctx).First(&department, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &department, nil
}

func (q *Queries) DepartmentNameExists(ctx context.Context, parentID int64, name string, excludeID int64) (bool, error) {
	db := q.db.WithContext(ctx).Model(&entity.Department{}).
		Where("parent_id = ? AND name = ?", parentID, name)
	if excludeID > 0 {
		db = db.Where("id <> ?", excludeID)
	}

	var count int64
	err := db.Count(&count).Error
	return count > 0, err
}

func (q *Queries) DeleteDepartment(ctx context.Context, id int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.User{}).Where("department_id = ?", id).Update("department_id", 0).Error; err != nil {
			return err
		}

		department := entity.Department{BaseModel: ormx.BaseModel{ID: id}}
		result := tx.Delete(&department)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (q *Queries) ListDepartments(ctx context.Context, filter DepartmentListFilter) ([]entity.Department, error) {
	db := q.db.WithContext(ctx).Model(&entity.Department{})
	if ormx.KeywordPresent(filter.Keyword) {
		db = db.Where("name LIKE ?", ormx.LikeKeyword(filter.Keyword))
	}
	if filter.ParentID != nil {
		db = db.Where("parent_id = ?", *filter.ParentID)
	}

	var departments []entity.Department
	err := db.Order("sort asc, id asc").Find(&departments).Error
	return departments, err
}

func (q *Queries) ListAllDepartments(ctx context.Context) ([]entity.Department, error) {
	var departments []entity.Department
	err := q.db.WithContext(ctx).Order("sort asc, id asc").Find(&departments).Error
	return departments, err
}
