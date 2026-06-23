package authentication

import (
	"context"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"strconv"
	"strings"

	"github.com/xiehqing/hitoken/htputil"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Device   string `json:"device"`
}

type LoginResponse struct {
	AccessToken string            `json:"accessToken"`
	TokenType   string            `json:"tokenType"`
	User        *entity.User      `json:"user"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Menus       []MenuTree        `json:"menus"`
	Department  entity.Department `json:"department"`
}

type MenuTree struct {
	entity.Menu
	Children []MenuTree `json:"children"`
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		return nil, fmt.Errorf("%w: 用户名和密码不能为空", ErrInvalidArgument)
	}

	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidLogin
	}
	if user.Status != userStatusEnabled {
		return nil, ErrUserDisabled
	}
	loginID := strconv.FormatInt(user.ID, 10)
	shouldLockOnFailure := !isAdminUsername(user.Username)
	if shouldLockOnFailure {
		if err := s.checkLoginLocked(loginID); err != nil {
			return nil, err
		}
	}

	authPwd := req.Password
	encryptConfig, err := s.GetEncryptConfig(ctx)
	if err != nil {
		return nil, err
	}
	if encryptConfig.Enabled {
		authPwd, err = s.decryptLoginPassword(ctx, req.Password)
		if err != nil {
			if shouldLockOnFailure {
				_ = s.recordLoginFailure(ctx, loginID)
			}
			return nil, ErrInvalidLogin
		}
	}
	if user.Password != md5Password(authPwd) {
		if shouldLockOnFailure {
			_ = s.recordLoginFailure(ctx, loginID)
		}
		return nil, ErrInvalidLogin
	}
	if shouldLockOnFailure {
		s.clearLoginFailure(loginID)
	}

	device := strings.TrimSpace(req.Device)
	if device == "" {
		device = "web"
	}

	token, err := htputil.Login(loginID, device)
	if err != nil {
		return nil, err
	}

	roles := roleNames(user.Roles)
	permissions, menus, err := s.authMenus(ctx, user)
	if err != nil {
		return nil, err
	}
	_ = htputil.SetRoles(loginID, roles)
	_ = htputil.SetPermissions(loginID, permissions)

	return &LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        user,
		Roles:       roles,
		Permissions: permissions,
		Menus:       menus,
		Department:  user.Department,
	}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	token = normalizeToken(token)
	if token == "" {
		return ErrTokenRequired
	}
	return htputil.LogoutByToken(token)
}

func (s *Service) CurrentUser(ctx context.Context, token string) (*LoginResponse, error) {
	token = normalizeToken(token)
	if token == "" {
		return nil, ErrTokenRequired
	}

	loginID, err := htputil.GetLoginID(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(loginID, 10, 64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.queries.GetUserForAuth(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidToken
	}
	if user.Status != userStatusEnabled {
		return nil, ErrUserDisabled
	}

	roles, _ := htputil.GetRoles(loginID)
	permissions, menus, err := s.authMenus(ctx, user)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		roles = roleNames(user.Roles)
	}
	return &LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        user,
		Roles:       roles,
		Permissions: permissions,
		Menus:       menus,
		Department:  user.Department,
	}, nil
}

func (s *Service) authMenus(ctx context.Context, user *entity.User) ([]string, []MenuTree, error) {
	var menus []entity.Menu
	var err error
	if user != nil && isAdminUsername(user.Username) {
		menus, err = s.queries.ListAllMenus(ctx)
		if err != nil {
			return nil, nil, err
		}
	} else {
		menus = userMenus(user.Roles)
		if len(menus) == 0 {
			return []string{}, []MenuTree{}, nil
		}
		allMenus, err := s.queries.ListAllMenus(ctx)
		if err != nil {
			return nil, nil, err
		}
		menus = withMenuAncestors(menus, allMenus)
	}

	tree := buildMenuTree(menus)
	permissions := menuPermissions(menus)
	return permissions, tree, nil
}
