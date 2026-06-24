package authentication

import (
	"testing"

	"github.com/xiehqing/hiauthx/db/entity"
)

func TestIsSystemManager(t *testing.T) {
	if !isSystemManager(&entity.User{Username: "admin"}) {
		t.Fatal("admin username should be treated as system manager")
	}
	if !isSystemManager(&entity.User{Roles: []entity.Role{{Name: "role_admin"}}}) {
		t.Fatal("role_admin should be treated as system manager")
	}
	if isSystemManager(&entity.User{Username: "user", Roles: []entity.Role{{Name: "role_user"}}}) {
		t.Fatal("normal user should not be treated as system manager")
	}
	if isSystemManager(nil) {
		t.Fatal("nil user should not be treated as system manager")
	}
}
