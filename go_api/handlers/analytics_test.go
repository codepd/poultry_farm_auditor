package handlers

import (
	"math"
	"poultry-farm-api/config"
	"strings"
	"testing"
)

func TestClassifyFeedStage(t *testing.T) {
	tests := []struct {
		itemName  string
		wantStage string
	}{
		{"LAYER MASH", "LAYER"},
		{"SYLO - LM", "LAYER"},
		{"SYLO L M", "LAYER"},
		{"LAYER CHICKS CRUMBLE 50KG", "CHICK"},
		{"LCC 1", "CHICK"},
		{"SYLO - C M", "CHICK"},
		{"CHICK MASH", "CHICK"},
		{"SYLO- G M", "GROWER"},
		{"GROWER MASH", "GROWER"},
		{"SYLO - PLM", "PRELAYER"},
		{"PRE-LAYER MASH", "PRELAYER"},
		{"SHELL GRIT", "OTHER"},
	}

	for _, tc := range tests {
		gotStage, _ := classifyFeedStage(tc.itemName)
		if gotStage != tc.wantStage {
			t.Errorf("classifyFeedStage(%q) = %q, want %q", tc.itemName, gotStage, tc.wantStage)
		}
	}
}

func TestAnalyticsCacheLifecycleAndInvalidation(t *testing.T) {
	InvalidateAnalyticsCache("")

	tenantA := "8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
	tenantB := "11111111-2222-3333-4444-555555555555"

	keyA1 := tenantA + ":enhanced:2026:9:true"
	keyA2 := tenantA + ":last12m:true"
	keyB1 := tenantB + ":enhanced:2026:9:true"

	setAnalyticsCached(keyA1, map[string]interface{}{"month": 9})
	setAnalyticsCached(keyA2, map[string]interface{}{"months": 12})
	setAnalyticsCached(keyB1, map[string]interface{}{"month": 9})

	if _, ok := getAnalyticsCached(keyA1); !ok {
		t.Fatalf("expected keyA1 to be cached")
	}
	if _, ok := getAnalyticsCached(keyA2); !ok {
		t.Fatalf("expected keyA2 to be cached")
	}
	if _, ok := getAnalyticsCached(keyB1); !ok {
		t.Fatalf("expected keyB1 to be cached")
	}

	// Invalidate tenantA only
	InvalidateAnalyticsCache(tenantA)

	if _, ok := getAnalyticsCached(keyA1); ok {
		t.Errorf("expected keyA1 to be evicted after tenantA invalidation")
	}
	if _, ok := getAnalyticsCached(keyA2); ok {
		t.Errorf("expected keyA2 to be evicted after tenantA invalidation")
	}
	if _, ok := getAnalyticsCached(keyB1); !ok {
		t.Errorf("expected keyB1 to remain cached after tenantA invalidation")
	}
}

func TestSanitizeFloat(t *testing.T) {
	if got := sanitizeFloat(math.NaN()); got != 0 {
		t.Errorf("sanitizeFloat(NaN) = %v, want 0", got)
	}
	if got := sanitizeFloat(math.Inf(1)); got != 0 {
		t.Errorf("sanitizeFloat(+Inf) = %v, want 0", got)
	}
	if got := sanitizeFloat(math.Inf(-1)); got != 0 {
		t.Errorf("sanitizeFloat(-Inf) = %v, want 0", got)
	}
	if got := sanitizeFloat(123.45); got != 123.45 {
		t.Errorf("sanitizeFloat(123.45) = %v, want 123.45", got)
	}
}

func TestSeptember2026ProfitAndPerEggConsistency(t *testing.T) {
	// Real September 2026 numbers from Pradeep Farm snapshot:
	// Eggs Sold: 7,98,060 Nos
	// Egg Revenue: ₹43,31,665.32
	// Discounts: ₹38,557.00
	// Layer Feed: ₹32,99,736.22
	// Rearing Feed: ₹6,45,796.36
	// Total Feed: ₹39,45,532.58
	// Medicines: ₹96,115.00
	// Overheads: ₹3,82,000.00
	eggsSold := 798060.0
	eggRev := 4331665.32
	discounts := 38557.0
	layerFeed := 3299736.22
	rearingFeed := 645796.36
	totalFeed := layerFeed + rearingFeed
	meds := 96115.0
	overheads := 382000.0

	// 1. Flock Operating Profit (Eggs + Discounts - Layer Feed - Meds)
	flockProfit := (eggRev + discounts) - (layerFeed + meds)
	flockProfitPerEgg := flockProfit / eggsSold
	if math.Abs(flockProfit-974371.10) > 0.01 {
		t.Errorf("flockOperatingProfit = %.2f, want 974371.10", flockProfit)
	}
	if math.Abs(flockProfitPerEgg-1.2209) > 0.01 {
		t.Errorf("flockProfitPerEgg = %.4f, want ~1.22", flockProfitPerEgg)
	}

	// 2. Statement Ledger Profit (Eggs + Discounts - Total Feed - Meds)
	ledgerProfit := (eggRev + discounts) - (totalFeed + meds)
	ledgerProfitPerEgg := ledgerProfit / eggsSold
	if math.Abs(ledgerProfit-328574.74) > 0.01 {
		t.Errorf("ledgerNetProfit = %.2f, want 328574.74", ledgerProfit)
	}
	if math.Abs(ledgerProfitPerEgg-0.4117) > 0.01 {
		t.Errorf("ledgerProfitPerEgg = %.4f, want ~0.41", ledgerProfitPerEgg)
	}

	// 3. Comprehensive Farm Net Profit (Eggs + Discounts - Total Feed - Meds - Overheads)
	farmProfit := (eggRev + discounts) - (totalFeed + meds + overheads)
	farmProfitPerEgg := farmProfit / eggsSold
	if math.Abs(farmProfit-(-53425.26)) > 0.01 {
		t.Errorf("farmNetProfit = %.2f, want -53425.26", farmProfit)
	}
	if math.Abs(farmProfitPerEgg-(-0.0669)) > 0.01 {
		t.Errorf("farmProfitPerEgg = %.4f, want ~-0.07", farmProfitPerEgg)
	}
}

func TestOTPVaultEncryptionAndHashing(t *testing.T) {
	cfg := &config.Config{
		OTPEncryptionKey: "unit-test-kms-vault-master-key-2026",
	}

	rawOTP := "482910"
	phone := "+919876543210"

	encrypted, keySource, err := EncryptOTP(rawOTP, cfg)
	if err != nil {
		t.Fatalf("EncryptOTP failed: %v", err)
	}
	if !strings.HasPrefix(encrypted, "enc:v1:") {
		t.Errorf("expected encrypted OTP to start with enc:v1:, got %s", encrypted)
	}
	if strings.Contains(encrypted, rawOTP) {
		t.Errorf("encrypted OTP %q must never contain raw OTP %q", encrypted, rawOTP)
	}
	if keySource != "ENV:OTP_ENCRYPTION_KEY" {
		t.Errorf("expected keySource ENV:OTP_ENCRYPTION_KEY, got %s", keySource)
	}

	// Verify round-trip decryption with Vault key
	decrypted, err := DecryptOTP(encrypted, cfg)
	if err != nil {
		t.Fatalf("DecryptOTP failed: %v", err)
	}
	if decrypted != rawOTP {
		t.Errorf("DecryptOTP = %q, want %q", decrypted, rawOTP)
	}

	// Verify decryption fails with wrong key
	wrongCfg := &config.Config{OTPEncryptionKey: "wrong-key"}
	if _, err := DecryptOTP(encrypted, wrongCfg); err == nil {
		t.Errorf("expected DecryptOTP with wrong key to fail, but succeeded")
	}

	// Verify deterministic HMAC-SHA256 lookup hash
	hash1 := HashOTPForLookup(phone, rawOTP, cfg)
	hash2 := HashOTPForLookup(phone, rawOTP, cfg)
	if hash1 != hash2 {
		t.Errorf("expected deterministic HMAC hash, got %s vs %s", hash1, hash2)
	}
	if !strings.HasPrefix(hash1, "hmac:v1:") || strings.Contains(hash1, rawOTP) {
		t.Errorf("invalid OTP HMAC hash: %s", hash1)
	}
}
