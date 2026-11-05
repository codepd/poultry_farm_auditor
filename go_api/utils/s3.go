package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// AWSCredentials holds temporary or static AWS credentials.
type AWSCredentials struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Token           string    `json:"Token"` // IMDS uses "Token"
	Expiration      time.Time `json:"Expiration"`
}

var (
	cachedCreds     *AWSCredentials
	cachedCredsLock sync.Mutex
)

// ResolveAWSCredentials resolves credentials from env vars, EKS/EC2 IMDSv2, or local AWS CLI SSO profile.
func ResolveAWSCredentials(profile string) (*AWSCredentials, error) {
	// 1. Check static/session environment variables first
	ak := os.Getenv("AWS_ACCESS_KEY_ID")
	sk := os.Getenv("AWS_SECRET_ACCESS_KEY")
	st := os.Getenv("AWS_SESSION_TOKEN")
	if ak != "" && sk != "" {
		return &AWSCredentials{
			AccessKeyID:     ak,
			SecretAccessKey: sk,
			SessionToken:    st,
		}, nil
	}

	cachedCredsLock.Lock()
	defer cachedCredsLock.Unlock()

	if cachedCreds != nil && (cachedCreds.Expiration.IsZero() || time.Now().Add(2*time.Minute).Before(cachedCreds.Expiration)) {
		return cachedCreds, nil
	}

	// 2. Try EC2 / EKS Worker Node Instance Metadata Service (IMDSv2)
	if creds, err := fetchIMDSv2Credentials(); err == nil && creds.AccessKeyID != "" {
		cachedCreds = creds
		return creds, nil
	}

	// 3. Try local AWS CLI (supports AWS SSO profiles like AdminAccess-Pradeep)
	if profile == "" {
		profile = os.Getenv("AWS_PROFILE")
	}
	if creds, err := fetchCLICredentials(profile); err == nil && creds.AccessKeyID != "" {
		cachedCreds = creds
		return creds, nil
	}

	return nil, fmt.Errorf("unable to resolve AWS credentials from environment, IMDSv2, or AWS CLI profile %q", profile)
}

func fetchIMDSv2Credentials() (*AWSCredentials, error) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	tokenReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return nil, err
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDSv2 token status %d", tokenResp.StatusCode)
	}
	tokenBytes, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, err
	}
	imdsToken := strings.TrimSpace(string(tokenBytes))

	roleReq, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/iam/security-credentials/", nil)
	if err != nil {
		return nil, err
	}
	roleReq.Header.Set("X-aws-ec2-metadata-token", imdsToken)
	roleResp, err := client.Do(roleReq)
	if err != nil {
		return nil, err
	}
	defer roleResp.Body.Close()
	roleBytes, err := io.ReadAll(roleResp.Body)
	if err != nil {
		return nil, err
	}
	roleName := strings.TrimSpace(strings.Split(string(roleBytes), "\n")[0])
	if roleName == "" {
		return nil, fmt.Errorf("no IAM role returned from IMDS")
	}

	credReq, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/iam/security-credentials/"+roleName, nil)
	if err != nil {
		return nil, err
	}
	credReq.Header.Set("X-aws-ec2-metadata-token", imdsToken)
	credResp, err := client.Do(credReq)
	if err != nil {
		return nil, err
	}
	defer credResp.Body.Close()

	var creds AWSCredentials
	if err := json.NewDecoder(credResp.Body).Decode(&creds); err != nil {
		return nil, err
	}
	if creds.SessionToken == "" && creds.Token != "" {
		creds.SessionToken = creds.Token
	}
	return &creds, nil
}

func fetchCLICredentials(profile string) (*AWSCredentials, error) {
	args := []string{"configure", "export-credentials", "--format", "process"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	cmd := exec.Command("aws", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var creds AWSCredentials
	if err := json.Unmarshal(out, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// S3PutObject uploads an object to S3 using SigV4 and returns the S3 version ID (if versioning is enabled).
func S3PutObject(bucket, region, profile, key string, data []byte, contentType string, tags map[string]string) (string, error) {
	creds, err := ResolveAWSCredentials(profile)
	if err != nil {
		return "", err
	}
	if region == "" {
		region = "ap-south-1"
	}

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, region)
	escapedKey := escapeS3Key(key)
	endpoint := fmt.Sprintf("https://%s/%s", host, escapedKey)

	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	payloadHash := sha256Hex(data)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if creds.SessionToken != "" {
		req.Header.Set("x-amz-security-token", creds.SessionToken)
	}
	if len(tags) > 0 {
		q := url.Values{}
		for k, v := range tags {
			q.Set(k, v)
		}
		req.Header.Set("x-amz-tagging", q.Encode())
	}

	signS3Request(req, creds, region, "/" + escapedKey, payloadHash, amzDate, dateStamp)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("S3 PutObject failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	versionID := resp.Header.Get("x-amz-version-id")
	return versionID, nil
}

// S3GetObject downloads an object from S3 using SigV4.
func S3GetObject(bucket, region, profile, key string) ([]byte, error) {
	creds, err := ResolveAWSCredentials(profile)
	if err != nil {
		return nil, err
	}
	if region == "" {
		region = "ap-south-1"
	}

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, region)
	escapedKey := escapeS3Key(key)
	endpoint := fmt.Sprintf("https://%s/%s", host, escapedKey)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	payloadHash := sha256Hex(nil)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if creds.SessionToken != "" {
		req.Header.Set("x-amz-security-token", creds.SessionToken)
	}

	signS3Request(req, creds, region, "/" + escapedKey, payloadHash, amzDate, dateStamp)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("S3 GetObject failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// S3PresignGetURL generates a SigV4 presigned GET URL for temporary browser viewing/downloading.
func S3PresignGetURL(bucket, region, profile, key string, expires time.Duration) (string, error) {
	creds, err := ResolveAWSCredentials(profile)
	if err != nil {
		return "", err
	}
	if region == "" {
		region = "ap-south-1"
	}

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, region)
	escapedKey := escapeS3Key(key)
	canonicalURI := "/" + escapedKey

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, region)

	expSeconds := int(expires.Seconds())
	if expSeconds <= 0 || expSeconds > 604800 {
		expSeconds = 3600
	}

	queryParams := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    fmt.Sprintf("%s/%s", creds.AccessKeyID, credentialScope),
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       fmt.Sprintf("%d", expSeconds),
		"X-Amz-SignedHeaders": "host",
	}
	if creds.SessionToken != "" {
		queryParams["X-Amz-Security-Token"] = creds.SessionToken
	}

	var keys []string
	for k := range queryParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalQueryParts []string
	for _, k := range keys {
		canonicalQueryParts = append(canonicalQueryParts, awsURIEncode(k, true)+"="+awsURIEncode(queryParams[k], true))
	}
	canonicalQuery := strings.Join(canonicalQueryParts, "&")

	canonicalHeaders := fmt.Sprintf("host:%s\n", host)
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"

	canonicalRequest := strings.Join([]string{
		"GET",
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(creds.SecretAccessKey, dateStamp, region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	return fmt.Sprintf("https://%s%s?%s&X-Amz-Signature=%s", host, canonicalURI, canonicalQuery, signature), nil
}

func signS3Request(req *http.Request, creds *AWSCredentials, region, canonicalURI, payloadHash, amzDate, dateStamp string) {
	headerMap := make(map[string]string)
	var headerNames []string

	for k, vals := range req.Header {
		lowerK := strings.ToLower(k)
		if lowerK == "host" || lowerK == "content-type" || strings.HasPrefix(lowerK, "x-amz-") {
			headerMap[lowerK] = strings.TrimSpace(vals[0])
			headerNames = append(headerNames, lowerK)
		}
	}
	if _, ok := headerMap["host"]; !ok {
		headerMap["host"] = req.Host
		headerNames = append(headerNames, "host")
	}
	sort.Strings(headerNames)

	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(headerMap[name])
		canonicalHeaders.WriteByte('\n')
	}
	signedHeaders := strings.Join(headerNames, ";")

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		"",
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(creds.SecretAccessKey, dateStamp, region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.AccessKeyID, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authHeader)
}

func escapeS3Key(key string) string {
	trimmed := strings.TrimPrefix(key, "/")
	parts := strings.Split(trimmed, "/")
	for i, p := range parts {
		parts[i] = awsURIEncode(p, true)
	}
	return strings.Join(parts, "/")
}

func awsURIEncode(s string, encodeSlash bool) string {
	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			buf.WriteByte(c)
		} else if c == '/' && !encodeSlash {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
