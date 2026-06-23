package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"
	"gorm.io/gorm"
)

type RoleListFilter struct {
	ormx.Pagination
	BuiltIn *int
}

func (q *Queries) CreateRole(ctx context.Context, role *entity.Role) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Omit("Menus").Create(role).Error
	})
}

func (q *Queries) UpdateRole(ctx context.Context, role *entity.Role) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Model(&entity.Role{}).
			Where("id = ?", role.ID).
			Select("DisplayName", "Description", "UpdatedBy").
			Updates(role).
			Error
	})
}

func (q *Queries) GetRole(ctx context.Context, id int64) (*entity.Role, error) {
	var role entity.Role
	err := q.db.WithContext(ctx).Preload("Menus").First(&role, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &role, nil
}

func (q *Queries) DeleteRole(ctx context.Context, id int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role := entity.Role{BaseModel: ormx.BaseModel{ID: id}}
		if err := tx.Exec("DELETE FROM role_menus WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		result := tx.Delete(&role)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (q *Queries) ListRoles(ctx context.Context, filter RoleListFilter) (ormx.PageResult[entity.Role], error) {
	db := q.db.WithContext(ctx).Model(&entity.Role{})
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("display_name LIKE ? OR name LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	if filter.BuiltIn != nil {
		db = db.Where("built_in = ?", *filter.BuiltIn)
	}
	return ormx.Paginate[entity.Role](db, filter.Pagination, map[string]string{
		"id":          "id",
		"displayName": "display_name",
		"name":        "name",
		"builtIn":     "built_in",
		"createdAt":   "created_at",
		"updatedAt":   "updated_at",
	})
}

func (q *Queries) ListAllRoles(ctx context.Context) ([]entity.Role, error) {
	var roles []entity.Role
	err := q.db.WithContext(ctx).Order("id asc").Find(&roles).Error
	return roles, err
}

func (q *Queries) UpdateRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		beforeMenus, err := findRoleMenus(tx, roleID)
		if err != nil {
			return err
		}
		role := entity.Role{BaseModel: ormx.BaseModel{ID: roleID}}
		if err := replaceRoleMenus(tx, &role, menuIDs); err != nil {
			return err
		}
		return q.AuditAssociation(ctx, tx, "role_menus", "role_menus", roleID, map[string]any{
			"roleId":  roleID,
			"menuIds": menuIDsFromMenus(beforeMenus),
		}, map[string]any{
			"roleId":  roleID,
			"menuIds": normalizeAuditIDs(menuIDs),
		})
	})
}

func (q *Queries) GetRoleMenus(ctx context.Context, roleID int64) ([]entity.Menu, error) {
	var role entity.Role
	if err := q.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	var menus []entity.Menu
	err := q.db.WithContext(ctx).
		Model(&role).
		Order("sort asc, id asc").
		Association("Menus").
		Find(&menus)
	return menus, err
}

func replaceRoleMenus(tx *gorm.DB, role *entity.Role, menuIDs []int64) error {
	if role == nil || role.ID <= 0 {
		return nil
	}
	if err := tx.Exec("DELETE FROM role_menus WHERE role_id = ?", role.ID).Error; err != nil {
		return err
	}
	for _, menuID := range normalizeDataIDs(menuIDs) {
		if err := tx.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", role.ID, menuID).Error; err != nil {
			return err
		}
	}
	return nil
}

func findRoleMenus(tx *gorm.DB, roleID int64) ([]entity.Menu, error) {
	role := entity.Role{BaseModel: ormx.BaseModel{ID: roleID}}
	var menus []entity.Menu
	err := tx.Model(&role).Order("sort asc, id asc").Association("Menus").Find(&menus)
	return menus, err
}

func menuIDsFromMenus(menus []entity.Menu) []int64 {
	result := make([]int64, 0, len(menus))
	for _, menu := range menus {
		if menu.ID > 0 {
			result = append(result, menu.ID)
		}
	}
	return result
}
