package authentication

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/cryptox"
	"github.com/xiehqing/infra/pkg/logs"
	"sort"
	"strings"
)

func decryptPassword(password, aesKey string) string {
	authPwd := password
	if aesKey != "" {
		decryptPwd, err := cryptox.NewAes(cryptox.WithKey(aesKey)).Decrypt(password)
		if err != nil {
			logs.Errorf("解密密码错误: %v，即将尝试使用明文密码登录", err)
		} else {
			authPwd = string(decryptPwd)
		}
	}
	return authPwd
}

func md5Password(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

func roleNames(roles []entity.Role) []string {
	result := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		name := strings.TrimSpace(role.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func userPermissions(roles []entity.Role) ([]string, []entity.Menu) {
	menus := userMenus(roles)
	return menuPermissions(menus), menus
}

func userMenus(roles []entity.Role) []entity.Menu {
	menus := make([]entity.Menu, 0)
	seenMenus := make(map[int64]struct{})

	for _, role := range roles {
		for _, menu := range role.Menus {
			if _, ok := seenMenus[menu.ID]; !ok {
				seenMenus[menu.ID] = struct{}{}
				menus = append(menus, menu)
			}
		}
	}

	return menus
}

func menuPermissions(menus []entity.Menu) []string {
	permissions := make([]string, 0, len(menus))
	seenPerms := make(map[string]struct{}, len(menus))
	for _, menu := range menus {
		permission := strings.TrimSpace(menu.Route)
		if permission == "" {
			continue
		}
		if _, ok := seenPerms[permission]; ok {
			continue
		}
		seenPerms[permission] = struct{}{}
		permissions = append(permissions, permission)
	}
	return permissions
}

func buildMenuTree(menus []entity.Menu) []MenuTree {
	sortMenus(menus)
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

func sortMenus(menus []entity.Menu) {
	sort.SliceStable(menus, func(i, j int) bool {
		if menus[i].Sort == menus[j].Sort {
			return menus[i].ID < menus[j].ID
		}
		return menus[i].Sort < menus[j].Sort
	})
}

func normalizeToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token)
}
