package models

import (
	"database/sql"
	"time"
	"github.com/google/uuid"
)

type Transaction struct {
	ID                int            `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	TransactionDate   time.Time      `json:"transaction_date"`
	TransactionType   string         `json:"transaction_type"` // SALE, PURCHASE, PAYMENT, TDS, DISCOUNT, EXPENSE, INCOME
	Category          string         `json:"category"`        // EGG, FEED, MEDICINE, OTHER, CHICK, GROWER, MANURE, EMPLOYEE
	ItemName          sql.NullString `json:"item_name,omitempty"`
	Quantity          sql.NullFloat64 `json:"quantity,omitempty"`
	Unit              sql.NullString `json:"unit,omitempty"`
	Rate              sql.NullFloat64 `json:"rate,omitempty"`
	Amount            float64         `json:"amount"`
	Notes             sql.NullString `json:"notes,omitempty"`
	Status            string         `json:"status"` // DRAFT, SUBMITTED, APPROVED, REJECTED
	SubmittedByUserID *int            `json:"submitted_by_user_id,omitempty"`
	ApprovedByUserID  *int            `json:"approved_by_user_id,omitempty"`
	ApprovedAt        *time.Time      `json:"approved_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Receipt struct {
	ID               int       `json:"id"`
	TransactionID    int       `json:"transaction_id"`
	FileName         string    `json:"file_name"`
	FilePath         string    `json:"file_path"`
	FileSize         *int64    `json:"file_size,omitempty"`
	MimeType         string    `json:"mime_type"`
	UploadedByUserID *int      `json:"uploaded_by_user_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
