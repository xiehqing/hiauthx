package configx

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/jsonx"
	"strconv"
	"strings"
)

type Reader struct {
	queries *queries.Queries
}

func New(queries *queries.Queries) *Reader {
	return &Reader{queries: queries}
}

func (r *Reader) String(ctx context.Context, key, defaultValue string) string {
	config, err := r.queries.GetSystemConfigByKey(ctx, key)
	if err != nil || config == nil || config.Enabled != entity.ConfigEnabled {
		return defaultValue
	}
	value := strings.TrimSpace(config.Value)
	if value == "" {
		return defaultValue
	}
	return value
}

func (r *Reader) Int(ctx context.Context, key string, defaultValue int) int {
	value := r.String(ctx, key, "")
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func (r *Reader) Int64(ctx context.Context, key string, defaultValue int64) int64 {
	value := r.String(ctx, key, "")
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return result
}

func (r *Reader) Bool(ctx context.Context, key string, defaultValue bool) bool {
	value := r.String(ctx, key, "")
	if value == "" {
		return defaultValue
	}
	return Bool(value)
}

func (r *Reader) JSON(ctx context.Context, key string, target any) error {
	value := r.String(ctx, key, "")
	if value == "" {
		return nil
	}
	if !jsonx.IsJSON(value) {
		return errors.New("配置值不是合法 JSON")
	}
	return json.Unmarshal([]byte(value), target)
}

func Bool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
