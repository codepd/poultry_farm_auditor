package middleware

import (
	"testing"

	"github.com/google/uuid"
)

func TestDefaultRolePermissionsOtherUserCannotViewCharts(t *testing.T) {
	perms := defaultRolePermissions(uuid.New(), "OTHER_USER")
	if perms.CanViewCharts {
		t.Fatalf("expected OTHER_USER to have CanViewCharts=false")
	}
	if perms.CanViewSensitiveData || perms.CanManageUsers {
		t.Fatalf("expected OTHER_USER to remain restricted on sensitive data and user management: %+v", perms)
	}
}

func TestHasPermissionManageImpliesRead(t *testing.T) {
	perms := []string{"manage:expense", "manage:ledger", "read:hen_batch"}
	if !HasPermission(perms, "read:expense") {
		t.Fatalf("expected manage:expense to grant read:expense")
	}
	if !HasPermission(perms, "manage:expense") {
		t.Fatalf("expected manage:expense to grant manage:expense")
	}
	if !HasPermission(perms, "read:ledger") {
		t.Fatalf("expected manage:ledger to grant read:ledger")
	}
	if HasPermission(perms, "manage:hen_batch") {
		t.Fatalf("expected read:hen_batch NOT to grant manage:hen_batch")
	}
}

func TestDefaultPermissionsForRoleDailyLogs(t *testing.T) {
	for _, role := range []string{"OWNER", "CO_OWNER", "MANAGER", "OTHER_USER"} {
		perms := DefaultPermissionsForRole(role, false)
		if !HasPermission(perms, "submit:daily_log") {
			t.Fatalf("expected role %s to have submit:daily_log", role)
		}
	}
	mgrNoProfit := DefaultPermissionsForRole("MANAGER", false)
	if HasPermission(mgrNoProfit, "read:profit") {
		t.Fatalf("expected MANAGER without override NOT to have read:profit")
	}
	mgrWithProfit := DefaultPermissionsForRole("MANAGER", true)
	if !HasPermission(mgrWithProfit, "read:profit") {
		t.Fatalf("expected MANAGER with override to have read:profit")
	}
}

func TestParseOrCreateTraceContext(t *testing.T) {
	incoming := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	traceID, spanID, outHeader, corrID := ParseOrCreateTraceContext(incoming, "")
	if traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("expected preserved traceID, got %q", traceID)
	}
	if len(spanID) != 16 {
		t.Fatalf("expected 16-char spanID, got %q", spanID)
	}
	if corrID != traceID {
		t.Fatalf("expected corrID to default to traceID, got %q", corrID)
	}
	if outHeader != "00-"+traceID+"-"+spanID+"-01" {
		t.Fatalf("unexpected traceparent output %q", outHeader)
	}

	// Fallback generation when no header is supplied
	t2, s2, _, c2 := ParseOrCreateTraceContext("", "req-custom-123")
	if len(t2) != 32 || len(s2) != 16 || c2 != "req-custom-123" {
		t.Fatalf("unexpected generated trace context: %q %q %q", t2, s2, c2)
	}
}
