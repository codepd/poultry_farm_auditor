package middleware

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultRolePermissionsOtherUserCanViewCharts(t *testing.T) {
	perms := defaultRolePermissions(uuid.New(), "OTHER_USER")
	if !perms.CanViewCharts {
		t.Fatalf("expected OTHER_USER to have CanViewCharts=true")
	}
	if perms.CanViewSensitiveData || perms.CanManageUsers {
		t.Fatalf("expected OTHER_USER to remain restricted on sensitive data and user management: %+v", perms)
	}
}
