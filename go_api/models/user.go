package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantUser struct {
	ID        int       `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    int       `json:"user_id"`
	Role      string    `json:"role"` // ADMIN, OWNER, CO_OWNER, OTHER_USER, AUDITOR
	IsOwner   bool      `json:"is_owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Invitation struct {
	ID              int        `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	InvitedByUserID int        `json:"invited_by_user_id"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	Token           string     `json:"token"`
	ExpiresAt       time.Time  `json:"expires_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type RolePermission struct {
	ID                     int       `json:"id"`
	TenantID               uuid.UUID `json:"tenant_id"`
	Role                   string    `json:"role"`
	CanViewSensitiveData   bool      `json:"can_view_sensitive_data"`
	CanEditTransactions    bool      `json:"can_edit_transactions"`
	CanApproveTransactions bool      `json:"can_approve_transactions"`
	CanManageUsers         bool      `json:"can_manage_users"`
	CanViewCharts          bool      `json:"can_view_charts"`
}
