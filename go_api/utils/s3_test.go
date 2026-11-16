package utils

import (
	"strings"
	"testing"
	"time"
)

func TestS3PresignGetURLWithCreds(t *testing.T) {
	creds := &AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "test-session-token",
	}

	urlStr, err := S3PresignGetURLWithCreds(creds, "my-bucket", "ap-south-1", "tenant/2026/ledger.pdf", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(urlStr, "https://my-bucket.s3.ap-south-1.amazonaws.com/tenant/2026/ledger.pdf?") {
		t.Fatalf("unexpected presigned URL prefix: %s", urlStr)
	}
	if !strings.Contains(urlStr, "X-Amz-Signature=") || !strings.Contains(urlStr, "X-Amz-Security-Token=") {
		t.Fatalf("expected SigV4 signature and session token in URL: %s", urlStr)
	}
}
