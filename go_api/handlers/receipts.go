package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"

	"github.com/gorilla/mux"
)

// UploadReceipt handles receipt file upload
func UploadReceipt(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUserFromContext(r)
		if user == nil {
			respondWithError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		// Parse multipart form (max 10MB)
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Failed to parse form")
			return
		}

		transactionIDStr := r.FormValue("transaction_id")
		if transactionIDStr == "" {
			respondWithError(w, http.StatusBadRequest, "transaction_id is required")
			return
		}

		transactionID, err := strconv.Atoi(transactionIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid transaction_id")
			return
		}

		// Verify transaction exists and user has access
		var tenantID string
		err = database.DB.QueryRow(`
			SELECT tenant_id::text FROM transactions WHERE id = $1
		`, transactionID).Scan(&tenantID)

		if err != nil {
			respondWithError(w, http.StatusNotFound, "Transaction not found")
			return
		}

		// Verify user has access to this tenant
		var hasAccess bool
		err = database.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM tenant_users 
				WHERE user_id = $1 AND tenant_id::text = $2
			)
		`, user.UserID, tenantID).Scan(&hasAccess)

		if err != nil || !hasAccess {
			respondWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Get file from form
		file, handler, err := r.FormFile("file")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "No file provided")
			return
		}
		defer file.Close()

		// Create uploads directory if it doesn't exist
		uploadDir := cfg.UploadPath
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create upload directory")
			return
		}

		// Generate unique filename
		ext := filepath.Ext(handler.Filename)
		filename := fmt.Sprintf("receipt_%d_%d%s", transactionID, time.Now().Unix(), ext)
		filePath := filepath.Join(uploadDir, filename)

		// Create file on disk
		dst, err := os.Create(filePath)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}
		defer dst.Close()

		// Copy file content
		fileSize, err := io.Copy(dst, file)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}

		// Get MIME type
		mimeType := handler.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Save receipt record to database
		var receiptID int
		err = database.DB.QueryRow(`
			INSERT INTO receipts (
				transaction_id, file_name, file_path, file_size, mime_type, uploaded_by_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, transactionID, handler.Filename, filePath, fileSize, mimeType, user.UserID).Scan(&receiptID)

		if err != nil {
			// Clean up file if DB insert fails
			os.Remove(filePath)
			respondWithError(w, http.StatusInternalServerError, "Failed to save receipt record")
			return
		}

		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"success": true,
			"message": "Receipt uploaded successfully",
			"data": map[string]interface{}{
				"id":             receiptID,
				"transaction_id": transactionID,
				"file_name":     handler.Filename,
				"file_size":      fileSize,
				"mime_type":      mimeType,
			},
		})
	}
}

// GetReceipts returns all receipts for a transaction
func GetReceipts(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["transaction_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	// Verify transaction exists and user has access
	var tenantID string
	err = database.DB.QueryRow(`
		SELECT tenant_id::text FROM transactions WHERE id = $1
	`, transactionID).Scan(&tenantID)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	// Verify access
	var hasAccess bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tenant_users 
			WHERE user_id = $1 AND tenant_id::text = $2
		)
	`, user.UserID, tenantID).Scan(&hasAccess)

	if err != nil || !hasAccess {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, transaction_id, file_name, file_path, file_size, mime_type,
		       uploaded_by_user_id, created_at
		FROM receipts
		WHERE transaction_id = $1
		ORDER BY created_at DESC
	`, transactionID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch receipts")
		return
	}
	defer rows.Close()

	type ReceiptResponse struct {
		ID              int    `json:"id"`
		TransactionID   int    `json:"transaction_id"`
		FileName        string `json:"file_name"`
		FilePath        string `json:"file_path"`
		FileSize        *int64 `json:"file_size,omitempty"`
		MimeType        string `json:"mime_type"`
		UploadedByUserID *int  `json:"uploaded_by_user_id,omitempty"`
		CreatedAt       string `json:"created_at"`
	}

	var receipts []ReceiptResponse
	for rows.Next() {
		var receipt ReceiptResponse
		var fileSize sql.NullInt64
		var uploadedBy sql.NullInt64
		var createdAt time.Time

		err := rows.Scan(
			&receipt.ID, &receipt.TransactionID, &receipt.FileName,
			&receipt.FilePath, &fileSize, &receipt.MimeType,
			&uploadedBy, &createdAt,
		)
		if err != nil {
			continue
		}

		if fileSize.Valid {
			receipt.FileSize = &fileSize.Int64
		}
		if uploadedBy.Valid {
			uid := int(uploadedBy.Int64)
			receipt.UploadedByUserID = &uid
		}
		receipt.CreatedAt = createdAt.Format(time.RFC3339)

		receipts = append(receipts, receipt)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    receipts,
	})
}

