package handlers

import "testing"

func TestMergeEggPricesPrefersTransactionsAndPreservesStoredMonths(t *testing.T) {
	txPrices := []PriceHistory{
		{PriceDate: "2026-09-01", PriceType: "EGG", ItemName: "LARGE EGG", Price: 5.52},
		{PriceDate: "2026-09-01", PriceType: "EGG", ItemName: "MEDIUM EGG", Price: 5.42},
		{PriceDate: "2026-08-01", PriceType: "EGG", ItemName: "LARGE EGG", Price: 5.40},
	}
	storedPrices := []PriceHistory{
		{ID: 10, PriceDate: "2026-08-01T00:00:00Z", PriceType: "EGG", ItemName: "LARGE EGG", Price: 5.10},
		{ID: 11, PriceDate: "2025-09-01T00:00:00Z", PriceType: "EGG", ItemName: "LARGE EGG", Price: 4.95},
	}

	merged := mergeEggPrices(txPrices, storedPrices)
	if len(merged) != 4 {
		t.Fatalf("expected 4 merged egg price rows, got %d", len(merged))
	}

	foundSep2026 := false
	foundAug2026Price := 0.0
	foundSep2025 := false
	for _, p := range merged {
		if p.PriceDate == "2026-09-01" && p.ItemName == "LARGE EGG" {
			foundSep2026 = true
		}
		if len(p.PriceDate) >= 7 && p.PriceDate[:7] == "2026-08" && p.ItemName == "LARGE EGG" {
			foundAug2026Price = p.Price
		}
		if len(p.PriceDate) >= 7 && p.PriceDate[:7] == "2025-09" && p.ItemName == "LARGE EGG" {
			foundSep2025 = true
		}
	}

	if !foundSep2026 {
		t.Fatalf("expected September 2026 LARGE EGG price from transactions")
	}
	if foundAug2026Price != 5.40 {
		t.Fatalf("expected transaction-derived 2026-08 LARGE EGG price 5.40 to take precedence over stored 5.10, got %.2f", foundAug2026Price)
	}
	if !foundSep2025 {
		t.Fatalf("expected stored 2025-09 LARGE EGG fallback row to be preserved")
	}
}
