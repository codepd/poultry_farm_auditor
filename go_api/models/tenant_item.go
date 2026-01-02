package models

type TenantItem struct {
	ID           int    `json:"id"`
	TenantID     string `json:"tenant_id"`
	Category     string `json:"category"` // 'EGG' or 'FEED'
	ItemName     string `json:"item_name"`
	DisplayOrder int    `json:"display_order"`
	IsActive     bool   `json:"is_active"`
}

type TenantItemCreateRequest struct {
	Category     string `json:"category" binding:"required"`
	ItemName     string `json:"item_name" binding:"required"`
	DisplayOrder int    `json:"display_order"`
	IsActive     bool   `json:"is_active"`
}

type TenantItemUpdateRequest struct {
	ItemName     *string `json:"item_name,omitempty"`
	DisplayOrder *int    `json:"display_order,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}




