package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"
	"gorm.io/gorm"
)

type MenuListFilter struct {
	Keyword  string `json:"keyword" form:"keyword"`
	Type     *int   `json:"type" form:"type"`
	ParentID *int64 `json:"parentId" form:"parentId"`
	Show     *int   `json:"show" form:"show"`
}

func (q *Queries) CreateMenu(ctx context.Context, menu *entity.Menu) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roleIDs, err := firstChildInheritRoleIDs(tx, menu.ParentID)
		if err != nil {
			return err
		}
		if err := tx.Create(menu).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		if err := inheritMenuRoles(tx, menu.ID, roleIDs); err != nil {
			return err
		}
		return q.AuditAssociation(ctx, tx, "role_menus", "role_menus", menu.ID, map[string]any{
			"menuId":  menu.ID,
			"roleIds": []int64{},
		}, map[string]any{
			"menuId":  menu.ID,
			"roleIds": roleIDs,
		})
	})
}

func (q *Queries) UpdateMenu(ctx context.Context, menu *entity.Menu) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Model(&entity.Menu{}).
			Where("id = ?", menu.ID).
			Select("Type", "ParentID", "Name", "Route", "Sort", "Icon", "Show", "UpdatedBy").
			Updates(menu).
			Error
	})
}

func (q *Queries) GetMenu(ctx context.Context, id int64) (*entity.Menu, error) {
	var menu entity.Menu
	err := q.db.WithContext(ctx).First(&menu, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &menu, nil
}

func (q *Queries) HasChildMenus(ctx context.Context, parentID int64) (bool, error) {
	var count int64
	err := q.db.WithContext(ctx).Model(&entity.Menu{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count > 0, err
}

func (q *Queries) DeleteMenu(ctx context.Context, id int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM role_menus WHERE menu_id = ?", id).Error; err != nil {
			return err
		}

		menu := entity.Menu{BaseModel: ormx.BaseModel{ID: id}}
		result := tx.Delete(&menu)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (q *Queries) ListMenus(ctx context.Context, filter MenuListFilter) ([]entity.Menu, error) {
	db := q.db.WithContext(ctx).Model(&entity.Menu{})
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("name LIKE ? OR route LIKE ?", keyword, keyword)
	}
	if filter.Type != nil {
		db = db.Where("type = ?", *filter.Type)
	}
	if filter.ParentID != nil {
		db = db.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.Show != nil {
		db = db.Where("show = ?", *filter.Show)
	}

	var menus []entity.Menu
	err := db.Order("sort asc, id asc").Find(&menus).Error
	return menus, err
}

func (q *Queries) ListAllMenus(ctx context.Context) ([]entity.Menu, error) {
	var menus []entity.Menu
	err := q.db.WithContext(ctx).Order("sort asc, id asc").Find(&menus).Error
	return menus, err
}

func firstChildInheritRoleIDs(tx *gorm.DB, parentID int64) ([]int64, error) {
	if parentID <= 0 {
		return nil, nil
	}

	var childCount int64
	if err := tx.Model(&entity.Menu{}).Where("parent_id = ?", parentID).Count(&childCount).Error; err != nil {
		return nil, err
	}
	if childCount > 0 {
		return nil, nil
	}

	var roleIDs []int64
	err := tx.Table("role_menus").
		Where("menu_id = ?", parentID).
		Order("role_id asc").
		Pluck("role_id", &roleIDs).
		Error
	return roleIDs, err
}

func inheritMenuRoles(tx *gorm.DB, menuID int64, roleIDs []int64) error {
	for _, roleID := range roleIDs {
		if roleID <= 0 {
			continue
		}
		if err := tx.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
			return err
		}
	}
	return nil
}
