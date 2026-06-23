package routes

import (
	"testing"

	"github.com/xiehqing/hiauthx/db/entity"
)

func TestAPIMetadataCache(t *testing.T) {
	cache := newAPIMetadataCache()
	cache.replace([]entity.API{
		{Method: "get", Path: "/api/v1/users/:id", Module: "用户管理", Status: entity.APIStatusEnabled},
		{Method: "POST", Path: "/api/v1/users", Module: "用户管理", Status: entity.APIStatusDisabled},
	})

	item, ok := cache.get("GET", "/api/v1/users/:id")
	if !ok || item.Module != "用户管理" {
		t.Fatalf("enabled API was not found: %+v, %v", item, ok)
	}
	if _, ok := cache.get("POST", "/api/v1/users"); ok {
		t.Fatal("disabled API must not be cached")
	}

	cache.replace(nil)
	if _, ok := cache.get("GET", "/api/v1/users/:id"); ok {
		t.Fatal("old snapshot remained visible after replacement")
	}
}
