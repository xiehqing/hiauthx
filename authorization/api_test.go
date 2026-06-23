package authorization

import (
	"errors"
	"testing"

	"github.com/xiehqing/hiauthx/db/entity"
)

func TestNormalizeAPI(t *testing.T) {
	method, path, resource := normalizeAPI(" get ", " /api/v1/users/:id ", "")
	if method != "GET" || path != "/api/v1/users/:id" || resource != "request" {
		t.Fatalf("unexpected normalized API: %q %q %q", method, path, resource)
	}
}

func TestValidateAPI(t *testing.T) {
	if err := validateAPI("查询用户", "GET", "/api/v1/users/:id", "用户管理", "query", entity.APIStatusEnabled); err != nil {
		t.Fatalf("valid API was rejected: %v", err)
	}
	for _, test := range []struct {
		name, method, path, module, action string
		status                             int
	}{
		{"", "GET", "/users", "用户管理", "query", 1},
		{"用户", "TRACE", "/users", "用户管理", "query", 1},
		{"用户", "GET", "users", "用户管理", "query", 1},
		{"用户", "GET", "/users", "", "query", 1},
		{"用户", "GET", "/users", "用户管理", "query", 2},
	} {
		if err := validateAPI(test.name, test.method, test.path, test.module, test.action, test.status); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalid API was accepted: %+v", test)
		}
	}
}
