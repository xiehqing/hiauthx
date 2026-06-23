package authorization

import (
	"errors"
	"fmt"
	"github.com/xiehqing/hiauthx/db/queries"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrInvalidID       = errors.New("ID 不合法")
	ErrInvalidArgument = errors.New("请求参数不合法")
	ErrNotFound        = errors.New("记录不存在")
	ErrBuiltInRole     = errors.New("内置角色不允许删除")
)

type Service struct {
	queries *queries.Queries
}

func New(queries *queries.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

func normalizeString(value string) string {
	return strings.TrimSpace(value)
}

func normalizeIDs(ids []int64) []int64 {
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

func required(value string) bool {
	return strings.TrimSpace(value) != ""
}

// minLength 最小长度
func minLength(value string, minLength int) bool {
	return len(value) >= minLength
}

func normalizeDBError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "Duplicate entry"):
		return fmt.Errorf("%w: 数据已存在，请检查唯一字段", ErrInvalidArgument)
	case strings.Contains(message, "foreign key constraint fails"):
		return fmt.Errorf("%w: 存在关联数据", ErrInvalidArgument)
	}
	return err
}
