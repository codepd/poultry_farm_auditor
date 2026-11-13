package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"poultry-farm-api/config"
	"poultry-farm-api/database"
	"poultry-farm-api/middleware"
	"poultry-farm-api/utils"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// DocumentUploadRecord represents an entry in the document_uploads audit table.
type DocumentUploadRecord struct {
	ID               int                    `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	S3Bucket         string                 `json:"s3_bucket"`
	S3Key            string                 `json:"s3_key"`
	S3VersionID      string                 `json:"s3_version_id,omitempty"`
	FileName         string                 `json:"file_name"`
	FileType         string                 `json:"file_type"`
	ContentType      string                 `json:"content_type"`
	FileSize         int64                  `json:"file_size"`
	SHA256Hash       string                 `json:"sha256_hash"`
	DocumentYear     *int                   `json:"document_year,omitempty"`
	DocumentMonth    *int                   `json:"document_month,omitempty"`
	UploadSource     string                 `json:"upload_source"`
	UploadStatus     string                 `json:"upload_status"`
	ParseResult      map[string]interface{} `json:"parse_result,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	UploadedByUserID *int                   `json:"uploaded_by_user_id,omitempty"`
	LedgerParseID    *int                   `json:"ledger_parse_id,omitempty"`
	DownloadURL      string                 `json:"download_url,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

func detectDocumentTypeAndMIME(filename, requestedType string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "LEDGER_PDF", "application/pdf", nil
	case ".xlsx":
		return "PARSED_EXCEL", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	case ".xls":
		return "PARSED_EXCEL", "application/vnd.ms-excel", nil
	case ".jpg", ".jpeg":
		return "FEED_PRICE_IMAGE", "image/jpeg", nil
	case ".png":
		return "FEED_PRICE_IMAGE", "image/png", nil
	case ".webp":
		return "FEED_PRICE_IMAGE", "image/webp", nil
	default:
		if requestedType != "" {
			return requestedType, "application/octet-stream", nil
		}
		return "", "", fmt.Errorf("unsupported file format (%s). Supported: .pdf, .xlsx, .xls, .jpg, .jpeg, .png", ext)
	}
}

func inferDocumentYearMonth(filename string) (int, int) {
	lower := strings.ToLower(filepath.Base(filename))
	year := time.Now().Year()
	month := 0

	yearRe := regexp.MustCompile(`(202[0-9])`)
	if m := yearRe.FindStringSubmatch(lower); len(m) > 1 {
		if y, err := strconv.Atoi(m[1]); err == nil {
			year = y
		}
	} else {
		shortYearRe := regexp.MustCompile(`(?:^|parsed_)(2[4-9])_(0[1-9]|1[0-2])`)
		if m := shortYearRe.FindStringSubmatch(lower); len(m) > 2 {
			if y, err := strconv.Atoi(m[1]); err == nil {
				year = 2000 + y
			}
			if mo, err := strconv.Atoi(m[2]); err == nil {
				month = mo
			}
		}
	}

	monthMap := []struct {
		key string
		val int
	}{
		{"jan", 1}, {"feb", 2}, {"fed", 2}, {"mar", 3}, {"apr", 4},
		{"may", 5}, {"jun", 6}, {"jul", 7}, {"aug", 8},
		{"sep", 9}, {"oct", 10}, {"nov", 11}, {"dec", 12},
	}
	if month == 0 && !strings.Contains(lower, "yearly") && !strings.Contains(lower, "summary") {
		for _, mm := range monthMap {
			if strings.Contains(lower, mm.key) {
				month = mm.val
				break
			}
		}
	}

	if month == 0 {
		dateRe := regexp.MustCompile(`^(0[1-9]|1[0-2])_[0-9]{2}`)
		if m := dateRe.FindStringSubmatch(lower); len(m) > 1 {
			if parsedM, err := strconv.Atoi(m[1]); err == nil {
				month = parsedM
			}
		}
	}
	return year, month
}

func buildS3DocumentKey(tenantID, fileType, filename string, year int) string {
	if year <= 0 {
		year = time.Now().Year()
	}
	cleanName := filepath.Base(filename)
	switch fileType {
	case "FEED_PRICE_IMAGE":
		return fmt.Sprintf("%s/feed_price_history/%d/%s", tenantID, year, cleanName)
	case "PARSED_EXCEL":
		return fmt.Sprintf("%s/parsed_excels/%d/%s", tenantID, year, cleanName)
	default:
		return fmt.Sprintf("%s/ledgers/%d/%s", tenantID, year, cleanName)
	}
}

// UploadLedgerHandler handles document upload (PDF ledger, Excel, or Feed Price sheet),
// stores the artifact in S3 for auditability, and supports both async and sync parsing.
func UploadLedgerHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to verify permissions")
		return
	}
	if !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "You do not have administrative permissions to upload documents")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithError(w, http.StatusBadRequest, "File size too large (max 32MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("pdf")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "No file provided in form field 'file' or 'pdf'")
			return
		}
	}
	defer file.Close()

	requestedType := strings.ToUpper(strings.TrimSpace(r.FormValue("file_type")))
	fileType, contentType, err := detectDocumentTypeAndMIME(header.Filename, requestedType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	hashSum := sha256.Sum256(fileBytes)
	sha256HexStr := hex.EncodeToString(hashSum[:])
	fileSize := int64(len(fileBytes))

	inferredYear, inferredMonth := inferDocumentYearMonth(header.Filename)
	if yStr := r.FormValue("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 2000 {
			inferredYear = y
		}
	}
	if mStr := r.FormValue("month"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m >= 1 && m <= 12 {
			inferredMonth = m
		}
	}

	cfg := config.Load()
	s3Key := buildS3DocumentKey(tenantID.String(), fileType, header.Filename, inferredYear)

	// Save local staged copy for parser / fallback
	uploadDir := filepath.Join(cfg.UploadPath, "staged", tenantID.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to initialize staging directory")
		return
	}
	stagedPath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", sha256HexStr[:10], filepath.Base(header.Filename)))
	if err := os.WriteFile(stagedPath, fileBytes, 0644); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to stage uploaded file")
		return
	}

	// Upload to S3 for immutable audit storage
	tags := map[string]string{
		"tenant_id":     tenantID.String(),
		"file_type":     fileType,
		"upload_source": "WEB_PORTAL",
		"sha256":        sha256HexStr[:32],
	}
	s3VersionID, s3Err := utils.S3PutObject(
		cfg.S3DocumentsBucket,
		cfg.AWSRegion,
		cfg.AWSProfile,
		s3Key,
		fileBytes,
		contentType,
		tags,
	)
	if s3Err != nil {
		log.Printf("[WARN] S3 upload to s3://%s/%s failed (staged locally at %s): %v", cfg.S3DocumentsBucket, s3Key, stagedPath, s3Err)
	}

	initialStatus := "UPLOADED"
	if fileType == "FEED_PRICE_IMAGE" {
		initialStatus = "ARCHIVED"
	} else {
		initialStatus = "PARSING"
	}

	var docMonthPtr *int
	if inferredMonth >= 1 && inferredMonth <= 12 {
		docMonthPtr = &inferredMonth
	}

	var uploadID int
	err = database.DB.QueryRow(`
		INSERT INTO document_uploads (
			tenant_id, s3_bucket, s3_key, s3_version_id, file_name, file_type,
			content_type, file_size, sha256_hash, document_year, document_month,
			upload_source, upload_status, uploaded_by_user_id, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'WEB_PORTAL', $12, $13, CURRENT_TIMESTAMP)
		ON CONFLICT (tenant_id, s3_key) DO UPDATE SET
			s3_version_id = COALESCE(EXCLUDED.s3_version_id, document_uploads.s3_version_id),
			file_size = EXCLUDED.file_size,
			sha256_hash = EXCLUDED.sha256_hash,
			document_year = COALESCE(EXCLUDED.document_year, document_uploads.document_year),
			document_month = COALESCE(EXCLUDED.document_month, document_uploads.document_month),
			upload_status = EXCLUDED.upload_status,
			error_message = NULL,
			uploaded_by_user_id = EXCLUDED.uploaded_by_user_id,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`,
		tenantID, cfg.S3DocumentsBucket, s3Key, s3VersionID, header.Filename, fileType,
		contentType, fileSize, sha256HexStr, inferredYear, docMonthPtr,
		initialStatus, user.UserID,
	).Scan(&uploadID)
	if err != nil {
		log.Printf("[ERROR] Failed to record document_upload in DB: %v", err)
	}

	// If this is a feed price sheet image, return immediately with audit info
	if fileType == "FEED_PRICE_IMAGE" {
		downloadURL, _ := utils.S3PresignGetURL(cfg.S3DocumentsBucket, cfg.AWSRegion, cfg.AWSProfile, s3Key, 1*time.Hour)
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"upload_id":     uploadID,
				"file_name":     header.Filename,
				"file_type":     fileType,
				"s3_bucket":     cfg.S3DocumentsBucket,
				"s3_key":        s3Key,
				"s3_version_id": s3VersionID,
				"sha256_hash":   sha256HexStr,
				"upload_status": "ARCHIVED",
				"download_url":  downloadURL,
				"s3_synced":     s3Err == nil,
			},
		})
		return
	}

	commit := r.URL.Query().Get("commit") == "true" || r.FormValue("commit") == "true"
	asyncMode := r.URL.Query().Get("async") == "true" || r.FormValue("async") == "true"

	if asyncMode {
		go func(uID int, fPath, fType string, tID uuid.UUID, doCommit bool, fy, fm int) {
			result, parseErr := runLedgerParser(fPath, fType, tID, doCommit, fy, fm)
			updateDocumentUploadAfterParse(uID, tID, result, parseErr, doCommit)
		}(uploadID, stagedPath, fileType, tenantID, commit, inferredYear, inferredMonth)

		respondWithJSON(w, http.StatusAccepted, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"upload_id":     uploadID,
				"file_name":     header.Filename,
				"file_type":     fileType,
				"s3_bucket":     cfg.S3DocumentsBucket,
				"s3_key":        s3Key,
				"s3_version_id": s3VersionID,
				"sha256_hash":   sha256HexStr,
				"upload_status": "PARSING",
				"async":         true,
				"s3_synced":     s3Err == nil,
			},
		})
		return
	}

	// Synchronous execution (used for immediate preview or commit)
	result, parseErr := runLedgerParser(stagedPath, fileType, tenantID, commit, inferredYear, inferredMonth)
	updateDocumentUploadAfterParse(uploadID, tenantID, result, parseErr, commit)

	if parseErr != nil && result == nil {
		respondWithError(w, http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	downloadURL, _ := utils.S3PresignGetURL(cfg.S3DocumentsBucket, cfg.AWSRegion, cfg.AWSProfile, s3Key, 1*time.Hour)
	result["upload_id"] = uploadID
	result["s3_bucket"] = cfg.S3DocumentsBucket
	result["s3_key"] = s3Key
	result["s3_version_id"] = s3VersionID
	result["sha256_hash"] = sha256HexStr
	result["download_url"] = downloadURL
	result["s3_synced"] = s3Err == nil

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

func runLedgerParser(filePath, fileType string, tenantID uuid.UUID, commit bool, fallbackYear, fallbackMonth int) (map[string]interface{}, error) {
	if fileType == "LEDGER_PDF" || strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		return ParseAndCommitLedgerPDF(filePath, tenantID, commit, fallbackYear, fallbackMonth)
	}
	return executePythonLedgerParser(filePath, tenantID, commit)
}

func executePythonLedgerParser(filePath string, tenantID uuid.UUID, commit bool) (map[string]interface{}, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		absFilePath = filePath
	}
	projectRoot, _ := filepath.Abs("..")
	pythonBin := filepath.Join(projectRoot, "python_backend", "venv", "bin", "python")
	if _, err := os.Stat(pythonBin); os.IsNotExist(err) {
		pythonBin = "python3"
	}

	scriptPath := filepath.Join(projectRoot, "python_backend", "cli", "process_ledger_pdf.py")
	args := []string{
		scriptPath,
		absFilePath,
		"--tenant-id", tenantID.String(),
		"--json",
	}
	if !commit {
		args = append(args, "--dry-run")
	}

	cmd := exec.Command(pythonBin, args...)
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	outputStr := stdout.String()

	jsonStart := strings.Index(outputStr, "{")
	if jsonStart == -1 {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(outputStr)
		}
		if errMsg == "" && runErr != nil {
			errMsg = runErr.Error()
		}
		return nil, fmt.Errorf("failed to parse statement: %s", errMsg)
	}

	jsonPayload := outputStr[jsonStart:]
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPayload), &result); err != nil {
		return nil, fmt.Errorf("failed to decode parser output: %v", err)
	}

	if runErr != nil && !commit {
		result["warning"] = stderr.String()
	} else if runErr != nil && commit {
		return result, fmt.Errorf("commit warning/error: %s", stderr.String())
	}

	return result, nil
}

func updateDocumentUploadAfterParse(uploadID int, tenantID uuid.UUID, result map[string]interface{}, parseErr error, committed bool) {
	if uploadID <= 0 {
		return
	}

	if parseErr != nil && result == nil {
		_, _ = database.DB.Exec(`
			UPDATE document_uploads
			SET upload_status = 'ERROR',
			    error_message = $1,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND tenant_id = $3
		`, parseErr.Error(), uploadID, tenantID)
		return
	}

	status := "PARSED"
	if committed {
		status = "COMMITTED"
		InvalidateAnalyticsCache(tenantID.String())
	}

	var parsedYear, parsedMonth *int
	if yVal, ok := result["year"].(float64); ok && yVal > 2000 {
		yInt := int(yVal)
		parsedYear = &yInt
	}
	if mVal, ok := result["month"].(float64); ok && mVal >= 1 && mVal <= 12 {
		mInt := int(mVal)
		parsedMonth = &mInt
	}

	resultJSON, _ := json.Marshal(result)

	_, _ = database.DB.Exec(`
		UPDATE document_uploads
		SET upload_status = $1,
		    parse_result = $2,
		    document_year = COALESCE($3, document_year),
		    document_month = COALESCE($4, document_month),
		    error_message = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND tenant_id = $6
	`, status, string(resultJSON), parsedYear, parsedMonth, uploadID, tenantID)
}

// CommitDocumentUploadHandler commits a previously uploaded and parsed document by ID.
func CommitDocumentUploadHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	perms, err := middleware.GetUserPermissions(user.UserID, user.TenantID)
	if err != nil || !perms.CanEditTransactions {
		respondWithError(w, http.StatusForbidden, "You do not have permission to commit statements")
		return
	}

	vars := mux.Vars(r)
	uploadID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid upload ID")
		return
	}

	var s3Bucket, s3Key, fileName, sha256Hash string
	err = database.DB.QueryRow(`
		SELECT s3_bucket, s3_key, file_name, COALESCE(sha256_hash, '')
		FROM document_uploads
		WHERE id = $1 AND tenant_id = $2
	`, uploadID, tenantID).Scan(&s3Bucket, &s3Key, &fileName, &sha256Hash)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Upload record not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to query upload record")
		return
	}

	cfg := config.Load()
	uploadDir := filepath.Join(cfg.UploadPath, "staged", tenantID.String())
	_ = os.MkdirAll(uploadDir, 0755)

	prefix := ""
	if len(sha256Hash) >= 10 {
		prefix = sha256Hash[:10] + "_"
	}
	stagedPath := filepath.Join(uploadDir, prefix+filepath.Base(fileName))

	// If local staged file doesn't exist, fetch from S3
	if _, statErr := os.Stat(stagedPath); os.IsNotExist(statErr) {
		objBytes, getErr := utils.S3GetObject(s3Bucket, cfg.AWSRegion, cfg.AWSProfile, s3Key)
		if getErr != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Could not retrieve document from S3 (%s): %v", s3Key, getErr))
			return
		}
		if writeErr := os.WriteFile(stagedPath, objBytes, 0644); writeErr != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to write temporary file for commit")
			return
		}
	}

	result, parseErr := runLedgerParser(stagedPath, "", tenantID, true, 0, 0)
	updateDocumentUploadAfterParse(uploadID, tenantID, result, parseErr, true)
	if parseErr != nil && result == nil {
		respondWithError(w, http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// GetDocumentUploadStatusHandler returns a single upload record with parsed result and presigned S3 URL.
func GetDocumentUploadStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	vars := mux.Vars(r)
	uploadID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid upload ID")
		return
	}

	var rec DocumentUploadRecord
	var versionID, contentType, sha256Hash, errMsg sql.NullString
	var fileSize sql.NullInt64
	var docYear, docMonth, uploadedBy, ledgerParseID sql.NullInt32
	var parseResultRaw []byte

	err = database.DB.QueryRow(`
		SELECT id, tenant_id::text, s3_bucket, s3_key, s3_version_id, file_name, file_type,
		       content_type, file_size, sha256_hash, document_year, document_month,
		       upload_source, upload_status, parse_result, error_message,
		       uploaded_by_user_id, ledger_parse_id, created_at, updated_at
		FROM document_uploads
		WHERE id = $1 AND tenant_id = $2
	`, uploadID, tenantID).Scan(
		&rec.ID, &rec.TenantID, &rec.S3Bucket, &rec.S3Key, &versionID, &rec.FileName, &rec.FileType,
		&contentType, &fileSize, &sha256Hash, &docYear, &docMonth,
		&rec.UploadSource, &rec.UploadStatus, &parseResultRaw, &errMsg,
		&uploadedBy, &ledgerParseID, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Upload record not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	populateNullableUploadFields(&rec, versionID, contentType, sha256Hash, errMsg, fileSize, docYear, docMonth, uploadedBy, ledgerParseID, parseResultRaw)

	cfg := config.Load()
	if urlStr, urlErr := utils.S3PresignGetURL(rec.S3Bucket, cfg.AWSRegion, cfg.AWSProfile, rec.S3Key, 1*time.Hour); urlErr == nil {
		rec.DownloadURL = urlStr
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    rec,
	})
}

// ListDocumentUploadsHandler returns the audit history of uploaded/migrated documents in S3.
func ListDocumentUploadsHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	tenantID, err := uuid.Parse(user.TenantID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	fileTypeFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("file_type")))
	yearFilter := strings.TrimSpace(r.URL.Query().Get("year"))

	query := `
		SELECT id, tenant_id::text, s3_bucket, s3_key, s3_version_id, file_name, file_type,
		       content_type, file_size, sha256_hash, document_year, document_month,
		       upload_source, upload_status, parse_result, error_message,
		       uploaded_by_user_id, ledger_parse_id, created_at, updated_at
		FROM document_uploads
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if fileTypeFilter != "" && fileTypeFilter != "ALL" {
		query += fmt.Sprintf(" AND file_type = $%d", argIdx)
		args = append(args, fileTypeFilter)
		argIdx++
	}
	if yearFilter != "" && yearFilter != "ALL" {
		if y, yErr := strconv.Atoi(yearFilter); yErr == nil {
			query += fmt.Sprintf(" AND document_year = $%d", argIdx)
			args = append(args, y)
			argIdx++
		}
	}

	query += " ORDER BY COALESCE(document_year, 0) DESC, COALESCE(document_month, 0) DESC, created_at DESC LIMIT 200"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query document uploads: %v", err))
		return
	}
	defer rows.Close()

	cfg := config.Load()
	var records []DocumentUploadRecord
	for rows.Next() {
		var rec DocumentUploadRecord
		var versionID, contentType, sha256Hash, errMsg sql.NullString
		var fileSize sql.NullInt64
		var docYear, docMonth, uploadedBy, ledgerParseID sql.NullInt32
		var parseResultRaw []byte

		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.S3Bucket, &rec.S3Key, &versionID, &rec.FileName, &rec.FileType,
			&contentType, &fileSize, &sha256Hash, &docYear, &docMonth,
			&rec.UploadSource, &rec.UploadStatus, &parseResultRaw, &errMsg,
			&uploadedBy, &ledgerParseID, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			continue
		}

		populateNullableUploadFields(&rec, versionID, contentType, sha256Hash, errMsg, fileSize, docYear, docMonth, uploadedBy, ledgerParseID, parseResultRaw)

		if urlStr, urlErr := utils.S3PresignGetURL(rec.S3Bucket, cfg.AWSRegion, cfg.AWSProfile, rec.S3Key, 1*time.Hour); urlErr == nil {
			rec.DownloadURL = urlStr
		}
		records = append(records, rec)
	}

	if records == nil {
		records = []DocumentUploadRecord{}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    records,
	})
}

func populateNullableUploadFields(
	rec *DocumentUploadRecord,
	versionID, contentType, sha256Hash, errMsg sql.NullString,
	fileSize sql.NullInt64,
	docYear, docMonth, uploadedBy, ledgerParseID sql.NullInt32,
	parseResultRaw []byte,
) {
	if versionID.Valid {
		rec.S3VersionID = versionID.String
	}
	if contentType.Valid {
		rec.ContentType = contentType.String
	}
	if fileSize.Valid {
		rec.FileSize = fileSize.Int64
	}
	if sha256Hash.Valid {
		rec.SHA256Hash = sha256Hash.String
	}
	if errMsg.Valid {
		rec.ErrorMessage = errMsg.String
	}
	if docYear.Valid {
		y := int(docYear.Int32)
		rec.DocumentYear = &y
	}
	if docMonth.Valid {
		m := int(docMonth.Int32)
		rec.DocumentMonth = &m
	}
	if uploadedBy.Valid {
		u := int(uploadedBy.Int32)
		rec.UploadedByUserID = &u
	}
	if ledgerParseID.Valid {
		lp := int(ledgerParseID.Int32)
		rec.LedgerParseID = &lp
	}
	if len(parseResultRaw) > 0 {
		var pr map[string]interface{}
		if err := json.Unmarshal(parseResultRaw, &pr); err == nil {
			rec.ParseResult = pr
		}
	}
}
