package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"poultry-farm-api/config"
	"strings"
	"sync"
	"time"
)

var (
	otpVaultMu        sync.RWMutex
	cachedOTPKey      []byte
	cachedKeySource   string
	awsFetchAttempted bool
)

// ResolveOTPEncryptionKey returns a 32-byte AES-256 key and the source identifier.
// Priority:
//  1. Explicit OTP_ENCRYPTION_KEY environment variable (injected via ECS/systemd from Secrets Manager)
//  2. AWS Secrets Manager (secret ID from OTP_SECRET_ID, e.g. "kolipannai/otp-encryption-key")
//  3. AWS SSM Parameter Store SecureString (parameter from OTP_SSM_PARAMETER, KMS-decrypted)
//  4. Key derived via SHA-256 from JWT_SECRET (local fallback)
func ResolveOTPEncryptionKey(cfg *config.Config) ([]byte, string) {
	if cfg == nil {
		cfg = config.Load()
	}
	if cfg != nil && strings.TrimSpace(cfg.OTPEncryptionKey) != "" {
		sum := sha256.Sum256([]byte(strings.TrimSpace(cfg.OTPEncryptionKey)))
		return sum[:], "ENV:OTP_ENCRYPTION_KEY"
	}
	if envKey := strings.TrimSpace(os.Getenv("OTP_ENCRYPTION_KEY")); envKey != "" {
		sum := sha256.Sum256([]byte(envKey))
		return sum[:], "ENV:OTP_ENCRYPTION_KEY"
	}

	otpVaultMu.RLock()
	if len(cachedOTPKey) == 32 && cachedKeySource != "" {
		k := make([]byte, 32)
		copy(k, cachedOTPKey)
		src := cachedKeySource
		otpVaultMu.RUnlock()
		return k, src
	}
	alreadyAttempted := awsFetchAttempted
	otpVaultMu.RUnlock()

	// Only attempt external AWS CLI fetch if not running inside unit tests and not yet attempted
	if !alreadyAttempted && cfg != nil && os.Getenv("SKIP_AWS_VAULT_FETCH") != "true" {
		otpVaultMu.Lock()
		if !awsFetchAttempted {
			awsFetchAttempted = true
			if secretVal, src := fetchKeyFromAWSVault(cfg); secretVal != "" {
				sum := sha256.Sum256([]byte(secretVal))
				cachedOTPKey = sum[:]
				cachedKeySource = src
				k := make([]byte, 32)
				copy(k, cachedOTPKey)
				otpVaultMu.Unlock()
				return k, src
			}
		}
		otpVaultMu.Unlock()
	}

	jwtSecret := "default-kolipannai-secret"
	if cfg != nil && cfg.JWTSecret != "" {
		jwtSecret = cfg.JWTSecret
	}
	sum := sha256.Sum256([]byte("kolipannai-otp-aes256-vault-v1:" + jwtSecret))
	return sum[:], "KMS-Fallback:JWT-Derived"
}

func fetchKeyFromAWSVault(cfg *config.Config) (string, string) {
	region := cfg.AWSRegion
	if region == "" {
		region = "ap-south-1"
	}

	// 1. Try AWS Secrets Manager (HashiCorp Vault equivalent in AWS)
	if cfg.OTPSecretID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		args := []string{
			"secretsmanager", "get-secret-value",
			"--secret-id", cfg.OTPSecretID,
			"--region", region,
			"--query", "SecretString",
			"--output", "text",
		}
		if cfg.AWSProfile != "" {
			args = append(args, "--profile", cfg.AWSProfile)
		}
		out, err := exec.CommandContext(ctx, "aws", args...).Output()
		cancel()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" && val != "None" {
				return val, "AWS-SecretsManager:" + cfg.OTPSecretID
			}
		}
	}

	// 2. Try AWS SSM Parameter Store (SecureString backed by AWS KMS)
	if cfg.OTPSSMParameter != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		args := []string{
			"ssm", "get-parameter",
			"--name", cfg.OTPSSMParameter,
			"--with-decryption",
			"--region", region,
			"--query", "Parameter.Value",
			"--output", "text",
		}
		if cfg.AWSProfile != "" {
			args = append(args, "--profile", cfg.AWSProfile)
		}
		out, err := exec.CommandContext(ctx, "aws", args...).Output()
		cancel()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" && val != "None" {
				return val, "AWS-SSM-KMS:" + cfg.OTPSSMParameter
			}
		}
	}

	return "", ""
}

// EncryptOTP encrypts the 6-digit OTP using AES-256-GCM so plaintext OTPs are never written to logs.
// Format: "enc:v1:<base64(nonce + ciphertext + gcmTag)>"
func EncryptOTP(plaintextOTP string, cfg *config.Config) (string, string, error) {
	key, keySource := ResolveOTPEncryptionKey(cfg)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", keySource, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", keySource, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", keySource, err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintextOTP), nil)
	return "enc:v1:" + base64.RawURLEncoding.EncodeToString(sealed), keySource, nil
}

// DecryptOTP decrypts an AES-256-GCM encrypted OTP token ("enc:v1:<base64>") using the Vault key.
func DecryptOTP(encryptedToken string, cfg *config.Config) (string, error) {
	trimmed := strings.TrimSpace(encryptedToken)
	if !strings.HasPrefix(trimmed, "enc:v1:") {
		return "", errors.New("invalid encrypted OTP prefix")
	}
	payloadB64 := strings.TrimPrefix(trimmed, "enc:v1:")
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 payload: %w", err)
	}

	key, _ := ResolveOTPEncryptionKey(cfg)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt OTP with vault key: %w", err)
	}
	return string(plaintext), nil
}

// HashOTPForLookup computes a deterministic HMAC-SHA256 digest of (phone + ":" + otp)
// using the Vault key so the database otp_codes.code column stores a keyed hash instead of raw digits.
func HashOTPForLookup(phone, otp string, cfg *config.Config) string {
	key, _ := ResolveOTPEncryptionKey(cfg)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strings.TrimSpace(phone) + ":" + strings.TrimSpace(otp)))
	return "hmac:v1:" + hex.EncodeToString(mac.Sum(nil))
}

// GenerateSignedMagicToken generates a 256-bit random nonce and appends an HMAC-SHA256
// signature bound to the recipient (phone or email) using the Vault key.
// Format: "<64-char-random-hex>.<32-char-hmac-hex>"
func GenerateSignedMagicToken(recipient string, cfg *config.Config) (string, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	nonceHex := hex.EncodeToString(nonceBytes)
	sigHex := computeMagicTokenSignature(nonceHex, recipient, cfg)
	return nonceHex + "." + sigHex, nil
}

// VerifySignedMagicToken verifies that a magic token was signed by our server for the given recipient.
// Legacy unsigned 64-char hex tokens remain supported only if they match the DB invitation row.
func VerifySignedMagicToken(token string, recipient string, cfg *config.Config) bool {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		// Legacy unsigned hex token in DB: allow DB lookup + phone verification to validate
		return len(parts) == 1 && len(parts[0]) >= 16
	}
	expectedSig := computeMagicTokenSignature(parts[0], recipient, cfg)
	return hmac.Equal([]byte(parts[1]), []byte(expectedSig))
}

func computeMagicTokenSignature(nonceHex string, recipient string, cfg *config.Config) string {
	key, _ := ResolveOTPEncryptionKey(cfg)
	mac := hmac.New(sha256.New, key)
	normalizedRecipient := strings.ToLower(strings.TrimSpace(recipient))
	mac.Write([]byte("kolipannai-magic-v1:" + nonceHex + ":" + normalizedRecipient))
	fullSum := hex.EncodeToString(mac.Sum(nil))
	return fullSum[:32]
}
