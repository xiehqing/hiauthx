package authentication

import (
	"errors"
	"github.com/xiehqing/hiauthx/db/queries"
)

var (
	ErrInvalidLogin    = errors.New("用户名或密码错误")
	ErrInvalidRSAKey   = errors.New("RSA 密钥无效")
	ErrUserDisabled    = errors.New("用户已被禁用")
	ErrUserLocked      = errors.New("用户已被临时锁定")
	ErrTokenRequired   = errors.New("请先登录")
	ErrInvalidToken    = errors.New("登录状态已失效，请重新登录")
	ErrInvalidArgument = errors.New("请求参数不合法")
)

const userStatusEnabled = 1

type Service struct {
	queries *queries.Queries
}

func New(queries *queries.Queries) *Service {
	return &Service{
		queries: queries,
	}
}
