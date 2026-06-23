package authorization

import (
	"context"
	"errors"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/xiehqing/infra/pkg/ormx"
)

type CreateMenuRequest struct {
	Type     int    `json:"type"`
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Route    string `json:"route"`
	Sort     int    `json:"sort"`
	Icon     string `json:"icon"`
	Show     int    `json:"show"`
	Operator string `json:"operator"`
}

type UpdateMenuRequest struct {
	ID       int64  `json:"id"`
	Type     int    `json:"type"`
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Route    string `json:"route"`
	Sort     int    `json:"sort"`
	Icon     string `json:"icon"`
	Show     int    `json:"show"`
	Operator string `json:"operator"`
}

type ListMenusRequest struct {
	queries.MenuListFilter
}

type MenuTree struct {
	entity.Menu
	Children []MenuTree `json:"children"`
}

func (as *Service) CreateMenu(ctx context.Context, req CreateMenuRequest) (*entity.Menu, error) {
	if err := validateMenu(req.Type, req.ParentID, req.Name, req.Show); err != nil {
		return nil, err
	}
	if err := as.validateGroupParent(ctx, req.Type, req.ParentID); err != nil {
		return nil, err
	}

	menu := &entity.Menu{
		BaseModel: ormx.BaseModel{
			CreatedBy: normalizeString(req.Operator),
			UpdatedBy: normalizeString(req.Operator),
		},
		Type:     req.Type,
		ParentID: req.ParentID,
		Name:     normalizeString(req.Name),
		Route:    normalizeString(req.Route),
		Sort:     req.Sort,
		Icon:     normalizeString(req.Icon),
		Show:     req.Show,
	}

	if err := as.queries.CreateMenu(ctx, menu); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.queries.GetMenu(ctx, menu.ID)
}

func (as *Service) UpdateMenu(ctx context.Context, req UpdateMenuRequest) (*entity.Menu, error) {
	if req.ID <= 0 {
		return nil, errors.New("菜单id不合法")
	}
	if req.ParentID == req.ID {
		return nil, fmt.Errorf("%w: 上级菜单不能选择自身", ErrInvalidArgument)
	}
	if err := validateMenu(req.Type, req.ParentID, req.Name, req.Show); err != nil {
		return nil, err
	}
	if err := as.validateGroupParent(ctx, req.Type, req.ParentID); err != nil {
		return nil, err
	}

	menu := &entity.Menu{
		BaseModel: ormx.BaseModel{
			ID:        req.ID,
			UpdatedBy: normalizeString(req.Operator),
		},
		Type:     req.Type,
		ParentID: req.ParentID,
		Name:     normalizeString(req.Name),
		Route:    normalizeString(req.Route),
		Sort:     req.Sort,
		Icon:     normalizeString(req.Icon),
		Show:     req.Show,
	}

	if err := as.queries.UpdateMenu(ctx, menu); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetMenu(ctx, req.ID)
}

func (as *Service) GetMenu(ctx context.Context, id int64) (*entity.Menu, error) {
	if id <= 0 {
		return nil, errors.New("菜单id不合法")
	}
	menu, err := as.queries.GetMenu(ctx, id)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, ErrNotFound
	}
	return menu, nil
}

func (as *Service) ListMenus(ctx context.Context, req ListMenusRequest) ([]MenuTree, error) {
	req.Keyword = normalizeString(req.Keyword)
	menus, err := as.queries.ListMenus(ctx, req.MenuListFilter)
	if err != nil {
		return nil, err
	}
	if req.Keyword != "" || req.ParentID != nil {
		allMenus, err := as.queries.ListAllMenus(ctx)
		if err != nil {
			return nil, err
		}
		menus = withMenuAncestors(menus, allMenus)
	}
	return buildMenuTree(menus), nil
}

func (as *Service) DeleteMenu(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("菜单id不合法")
	}
	hasChildren, err := as.queries.HasChildMenus(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return fmt.Errorf("%w: 请先删除子菜单", ErrInvalidArgument)
	}
	return normalizeDBError(as.queries.DeleteMenu(ctx, id))
}

func validateMenu(menuType int, parentID int64, name string, show int) error {
	if menuType != entity.MenuTypeGroup && menuType != entity.MenuTypeMenu && menuType != entity.MenuTypeButton {
		return fmt.Errorf("%w: 菜单类型不合法", ErrInvalidArgument)
	}
	if parentID < 0 {
		return fmt.Errorf("%w: 上级菜单不合法", ErrInvalidArgument)
	}
	if !required(name) {
		return fmt.Errorf("%w: 菜单名称不能为空", ErrInvalidArgument)
	}
	if menuType == entity.MenuTypeGroup {
		return nil
	}
	if show != entity.MenuHidden && show != entity.MenuShown {
		return fmt.Errorf("%w: 显示状态不合法", ErrInvalidArgument)
	}
	return nil
}

func (as *Service) validateGroupParent(ctx context.Context, menuType int, parentID int64) error {
	if menuType != entity.MenuTypeGroup || parentID <= 0 {
		return nil
	}
	parent, err := as.queries.GetMenu(ctx, parentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return ErrNotFound
	}
	if parent.Type == entity.MenuTypeGroup {
		return fmt.Errorf("%w: 分组下不允许再创建子分组", ErrInvalidArgument)
	}
	return nil
}

func buildMenuTree(menus []entity.Menu) []MenuTree {
	childrenByParent := make(map[int64][]entity.Menu)
	for _, menu := range menus {
		childrenByParent[menu.ParentID] = append(childrenByParent[menu.ParentID], menu)
	}

	var build func(parentID int64) []MenuTree
	build = func(parentID int64) []MenuTree {
		children := childrenByParent[parentID]
		result := make([]MenuTree, 0, len(children))
		for _, menu := range children {
			result = append(result, MenuTree{
				Menu:     menu,
				Children: build(menu.ID),
			})
		}
		return result
	}

	return build(0)
}

func withMenuAncestors(matched []entity.Menu, all []entity.Menu) []entity.Menu {
	byID := make(map[int64]entity.Menu, len(all))
	for _, menu := range all {
		byID[menu.ID] = menu
	}

	selected := make(map[int64]entity.Menu, len(matched))
	for _, menu := range matched {
		selected[menu.ID] = menu
		for parentID := menu.ParentID; parentID > 0; {
			parent, ok := byID[parentID]
			if !ok {
				break
			}
			selected[parent.ID] = parent
			parentID = parent.ParentID
		}
	}

	result := make([]entity.Menu, 0, len(selected))
	for _, menu := range all {
		if item, ok := selected[menu.ID]; ok {
			result = append(result, item)
		}
	}
	return result
}
