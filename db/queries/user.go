package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"

	"gorm.io/gorm"
)

type UserListFilter struct {
	ormx.Pagination
	Status       *int  `json:"status" form:"status"`
	RoleID       int64 `json:"roleId" form:"roleId"`
	DepartmentID int64 `json:"departmentId" form:"departmentId"`
}

func (q *Queries) CreateUser(ctx context.Context, user *entity.User, roleIDs []int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Roles", "Department").Create(user).Error; err != nil {
			return err
		}
		if err := replaceUserRoles(tx, user, roleIDs); err != nil {
			return err
		}
		return q.AuditAssociation(ctx, tx, "user_roles", "user_roles", user.ID, map[string]any{
			"userId":  user.ID,
			"roleIds": []int64{},
		}, map[string]any{
			"userId":  user.ID,
			"roleIds": roleIDs,
		})
	})
}

func (q *Queries) UpdateUser(ctx context.Context, user *entity.User, roleIDs []int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before entity.User
		if err := tx.Preload("Roles").Preload("Department").First(&before, "id = ?", user.ID).Error; err != nil {
			return err
		}
		beforeRoleIDs := roleIDsFromRoles(before.Roles)

		fields := []string{"Username", "Nickname", "Phone", "Email", "Status", "DepartmentID", "UpdatedBy"}
		if user.Password != "" {
			fields = append(fields, "Password")
		}

		if err := tx.Model(&entity.User{}).Where("id = ?", user.ID).Select(fields).Updates(user).Error; err != nil {
			return err
		}
		if err := replaceUserRoles(tx, user, roleIDs); err != nil {
			return err
		}
		return q.AuditAssociation(ctx, tx, "user_roles", "user_roles", user.ID, map[string]any{
			"userId":  user.ID,
			"roleIds": beforeRoleIDs,
		}, map[string]any{
			"userId":  user.ID,
			"roleIds": normalizeAuditIDs(roleIDs),
		})
	})
}

func (q *Queries) GetUser(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := q.db.WithContext(ctx).
		Preload("Roles").
		Preload("Department").
		First(&user, "id = ?", id).
		Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &user, nil
}

func (q *Queries) GetUsers(ctx context.Context, ids []int64) ([]entity.User, error) {
	var user []entity.User
	err := q.db.WithContext(ctx).
		Preload("Roles").
		Preload("Department").
		Find(&user, "id in ?", ids).
		Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return user, nil
}

func (q *Queries) GetUserForAuth(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := q.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Menus").
		Preload("Department").
		First(&user, "id = ?", id).
		Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &user, nil
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := q.db.WithContext(ctx).
		Preload("Roles").
		Preload("Roles.Menus").
		Preload("Department").
		First(&user, "username = ?", username).
		Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &user, nil
}

func (q *Queries) DeleteUser(ctx context.Context, id int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := entity.User{StatusAbleModel: ormx.StatusAbleModel{BaseModel: ormx.BaseModel{ID: id}}}
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", id).Error; err != nil {
			return err
		}

		result := tx.Delete(&user)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (q *Queries) ListUsers(ctx context.Context, filter UserListFilter) (ormx.PageResult[entity.User], error) {
	db := q.db.WithContext(ctx).Model(&entity.User{}).Preload("Roles").Preload("Department")
	db = db.Where("LOWER(username) <> ?", "admin")
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ? OR email LIKE ?", keyword, keyword, keyword, keyword)
	}
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	if filter.RoleID > 0 {
		db = db.Where("id IN (SELECT user_id FROM user_roles WHERE role_id = ?)", filter.RoleID)
	}
	if filter.DepartmentID > 0 {
		departmentIDs, err := q.departmentDescendantIDs(ctx, filter.DepartmentID)
		if err != nil {
			return ormx.PageResult[entity.User]{}, err
		}
		db = db.Where("department_id IN ?", departmentIDs)
	}

	return ormx.Paginate[entity.User](db, filter.Pagination, map[string]string{
		"id":        "id",
		"username":  "username",
		"nickname":  "nickname",
		"status":    "status",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	})
}

func replaceUserRoles(tx *gorm.DB, user *entity.User, roleIDs []int64) error {
	if user == nil || user.ID <= 0 {
		return nil
	}
	if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", user.ID).Error; err != nil {
		return err
	}
	for _, roleID := range normalizeAuditIDs(roleIDs) {
		if err := tx.Exec("INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, roleID).Error; err != nil {
			return err
		}
	}
	return nil
}

func roleIDsFromRoles(roles []entity.Role) []int64 {
	result := make([]int64, 0, len(roles))
	for _, role := range roles {
		if role.ID > 0 {
			result = append(result, role.ID)
		}
	}
	return result
}

func normalizeAuditIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (q *Queries) departmentDescendantIDs(ctx context.Context, departmentID int64) ([]int64, error) {
	departments, err := q.ListAllDepartments(ctx)
	if err != nil {
		return nil, err
	}

	childrenByParent := make(map[int64][]int64)
	for _, department := range departments {
		childrenByParent[department.ParentID] = append(childrenByParent[department.ParentID], department.ID)
	}

	result := []int64{departmentID}
	queue := []int64{departmentID}
	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]
		for _, childID := range childrenByParent[parentID] {
			result = append(result, childID)
			queue = append(queue, childID)
		}
	}
	return result, nil
}
