package authorization

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/jsonx"
	"github.com/xiehqing/infra/pkg/logs"
	"github.com/xiehqing/infra/pkg/ormx"
	"regexp"
	"strconv"
	"strings"
)

const (
	UserStatusDisabled = 0
	UserStatusEnabled  = 1
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{2,31}$`)

type CreateUserRequest struct {
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Nickname     string  `json:"nickname"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Status       *int    `json:"status"`
	RoleIDs      []int64 `json:"roleIds"`
	DepartmentID int64   `json:"departmentId"`
	Operator     string  `json:"operator"`
}

type UpdateUserRequest struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Nickname     string  `json:"nickname"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Status       int     `json:"status"`
	RoleIDs      []int64 `json:"roleIds"`
	DepartmentID int64   `json:"departmentId"`
	Operator     string  `json:"operator"`
}

type ListUsersRequest struct {
	queries.UserListFilter
}

func (as *Service) checkPwdLength(ctx context.Context, password string) (bool, int) {
	pwdMinLenCfg, err := as.queries.GetSystemConfigByKey(ctx, entity.SecurityPasswordMinLength)
	var pwdMinLength int
	if err != nil || pwdMinLenCfg == nil || pwdMinLenCfg.Value != "" {
		if err != nil {
			logs.Errorf("获取系统配置：%s错误，使用默认值8", entity.SecurityPasswordMinLength)
		}
		pwdMinLength = 8
	} else {
		minLen, err := strconv.Atoi(pwdMinLenCfg.Value)
		if err != nil {
			pwdMinLength = 8
		}
		pwdMinLength = minLen
	}
	return minLength(password, pwdMinLength), pwdMinLength
}

func (as *Service) CreateUser(ctx context.Context, req CreateUserRequest) (*entity.User, error) {
	if !required(req.Username) || !required(req.Password) || !required(req.Nickname) {
		return nil, fmt.Errorf("%w: 用户名、密码和昵称不能为空", ErrInvalidArgument)
	}
	if err := validateUsername(req.Username); err != nil {
		return nil, err
	}
	valid, pwdLength := as.checkPwdLength(ctx, req.Password)
	if !valid {
		return nil, fmt.Errorf("密码长度最少不得低于%d位数", pwdLength)
	}

	status := UserStatusEnabled
	if req.Status != nil {
		status = *req.Status
	}
	if !validUserStatus(status) {
		return nil, fmt.Errorf("%w: 用户状态不合法", ErrInvalidArgument)
	}

	user := &entity.User{
		StatusAbleModel: ormx.StatusAbleModel{
			BaseModel: ormx.BaseModel{
				CreatedBy: normalizeString(req.Operator),
				UpdatedBy: normalizeString(req.Operator),
			},
			Status: status,
		},
		Username:     normalizeString(req.Username),
		Password:     md5Password(req.Password),
		Nickname:     normalizeString(req.Nickname),
		Phone:        normalizeString(req.Phone),
		Email:        normalizeString(req.Email),
		DepartmentID: req.DepartmentID,
	}
	logs.Infof(jsonx.ToJsonIgnoreError(user))
	err := as.queries.CreateUser(ctx, user, normalizeIDs(req.RoleIDs))
	if err != nil {
		logs.Errorf("error:%v", err)
		return nil, normalizeDBError(err)
	}
	return as.queries.GetUser(ctx, user.ID)
}

func (as *Service) UpdateUser(ctx context.Context, req UpdateUserRequest) (*entity.User, error) {
	if req.ID <= 0 {
		return nil, ErrInvalidID
	}
	if !required(req.Username) || !required(req.Nickname) {
		return nil, fmt.Errorf("%w: 用户名和昵称不能为空", ErrInvalidArgument)
	}
	if err := validateUsername(req.Username); err != nil {
		return nil, err
	}
	if !validUserStatus(req.Status) {
		return nil, fmt.Errorf("%w: 用户状态不合法", ErrInvalidArgument)
	}

	password := ""
	if req.Password != "" {
		password = md5Password(req.Password)
	}

	user := &entity.User{
		StatusAbleModel: ormx.StatusAbleModel{
			BaseModel: ormx.BaseModel{
				ID:        req.ID,
				UpdatedBy: normalizeString(req.Operator),
			},
			Status: req.Status,
		},
		Username:     normalizeString(req.Username),
		Password:     password,
		Nickname:     normalizeString(req.Nickname),
		Phone:        normalizeString(req.Phone),
		Email:        normalizeString(req.Email),
		DepartmentID: req.DepartmentID,
	}

	if err := as.queries.UpdateUser(ctx, user, normalizeIDs(req.RoleIDs)); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetUser(ctx, req.ID)
}

func (as *Service) GetUser(ctx context.Context, id int64) (*entity.User, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	user, err := as.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (as *Service) GetUsers(ctx context.Context, ids []int64) ([]entity.User, error) {
	if len(ids) <= 0 {
		return nil, ErrInvalidID
	}
	users, err := as.queries.GetUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (as *Service) ListUsers(ctx context.Context, req ListUsersRequest) (ormx.PageResult[entity.User], error) {
	return as.queries.ListUsers(ctx, req.UserListFilter)
}

func (as *Service) DeleteUser(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	user, err := as.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if isAdminUsername(user.Username) {
		return fmt.Errorf("%w: admin 用户不允许删除", ErrInvalidArgument)
	}
	if err := as.queries.DeleteUser(ctx, id); err != nil {
		return normalizeDBError(err)
	}
	return nil
}

func validUserStatus(status int) bool {
	return status == UserStatusDisabled || status == UserStatusEnabled
}

func validateUsername(username string) error {
	username = normalizeString(username)
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("%w: 用户名必须以字母开头，长度 3-32，仅支持字母、数字、下划线、点和短横线", ErrInvalidArgument)
	}
	return nil
}

func md5Password(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

func isAdminUsername(username string) bool {
	return strings.EqualFold(strings.TrimSpace(username), "admin")
}
