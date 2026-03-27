package models

import (
	"github.com/google/uuid"
	"time"
)

type Tenant struct {
	ID                             uuid.UUID  `json:"id"`
	ParentID                       *uuid.UUID `json:"parent_id,omitempty"`
	Name                           string     `json:"name"`
	Location                       string     `json:"location,omitempty"`
	CountryCode                    string     `json:"country_code"`
	Currency                       string     `json:"currency"`
	NumberFormat                   string     `json:"number_format"`
	DateFormat                     string     `json:"date_format"`
	Timezone                       string     `json:"timezone"`
	Capacity                       *int       `json:"capacity,omitempty"`
	AgeCategoryChickMaxWeeks       *int       `json:"age_category_chick_max_weeks,omitempty"`
	AgeCategoryGrowerMaxWeeks      *int       `json:"age_category_grower_max_weeks,omitempty"`
	AgeCategoryPreLayerMaxWeeks    *int       `json:"age_category_prelayer_max_weeks,omitempty"`
	RefreshTTLWithoutRememberHours *int       `json:"refresh_ttl_without_remember_hours,omitempty"`
	RefreshTTLWithRememberDays     *int       `json:"refresh_ttl_with_remember_days,omitempty"`
	CreatedAt                      time.Time  `json:"created_at"`
	UpdatedAt                      time.Time  `json:"updated_at"`
}

// IsTopLevel returns true if this is a top-level tenant (no parent)
func (t *Tenant) IsTopLevel() bool {
	return t.ParentID == nil
}

type TenantCreateRequest struct {
	ParentID                       *uuid.UUID `json:"parent_id,omitempty"`
	Name                           string     `json:"name" binding:"required"`
	Location                       string     `json:"location,omitempty"`
	CountryCode                    string     `json:"country_code"`
	Currency                       string     `json:"currency"`
	NumberFormat                   string     `json:"number_format"`
	DateFormat                     string     `json:"date_format"`
	Timezone                       string     `json:"timezone"`
	Capacity                       *int       `json:"capacity,omitempty"`
	RefreshTTLWithoutRememberHours *int       `json:"refresh_ttl_without_remember_hours,omitempty"`
	RefreshTTLWithRememberDays     *int       `json:"refresh_ttl_with_remember_days,omitempty"`
}

type TenantUpdateRequest struct {
	Name                           *string `json:"name,omitempty"`
	Location                       *string `json:"location,omitempty"`
	CountryCode                    *string `json:"country_code,omitempty"`
	Currency                       *string `json:"currency,omitempty"`
	NumberFormat                   *string `json:"number_format,omitempty"`
	DateFormat                     *string `json:"date_format,omitempty"`
	Timezone                       *string `json:"timezone,omitempty"`
	Capacity                       *int    `json:"capacity,omitempty"`
	AgeCategoryChickMaxWeeks       *int    `json:"age_category_chick_max_weeks,omitempty"`
	AgeCategoryGrowerMaxWeeks      *int    `json:"age_category_grower_max_weeks,omitempty"`
	AgeCategoryPreLayerMaxWeeks    *int    `json:"age_category_prelayer_max_weeks,omitempty"`
	RefreshTTLWithoutRememberHours *int    `json:"refresh_ttl_without_remember_hours,omitempty"`
	RefreshTTLWithRememberDays     *int    `json:"refresh_ttl_with_remember_days,omitempty"`
}
