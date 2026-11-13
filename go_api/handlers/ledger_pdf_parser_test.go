package handlers

import (
	"math"
	"testing"
)

func TestParseStatementLinesFullParity(t *testing.T) {
	lines := []string{
		"QA Sample Feed Mill Pvt Ltd",
		"Ledger Account : QA Automation Test Farm",
		"1-Nov-26 to 28-Nov-26",
		"01-Nov-26 Opening Balance 10,000.00",
		"05-Nov-26 Poultry Egg Sales 1,35,000.00",
		"LARGE EGG 25000 NOS 5.40/NOS 1,35,000.00",
		"08-Nov-26 Poultry Feed Sales 1,19,000.00",
		"LAYER MASH BULK 3000 KGS 34.00/KGS 1,02,000.00",
		"GROWER MASH 500 KGS 34.00/KGS 17,000.00",
		"12-Nov-26 Poultry Feed Suppliments 2,640.00",
		"POULTRY FEED SUPPLIMENTS 2 NOS 1320.00/NOS 2,640.00",
		"15-Nov-26 Poultry Feed Sales Credit Note 3,000.00",
		"20-Nov-26 To Canara Bank Payment 50,000.00",
		"25-Nov-26 TDS 1,350.00",
	}

	ps := parseStatementLines(lines)

	if ps.ProviderName != "QA Sample Feed Mill Pvt Ltd" {
		t.Fatalf("unexpected provider name: %q", ps.ProviderName)
	}
	if ps.AccountHolder != "QA Automation Test Farm" {
		t.Fatalf("unexpected account holder: %q", ps.AccountHolder)
	}
	if ps.OpeningBalance == nil || math.Abs(*ps.OpeningBalance-10000.0) > 0.01 {
		t.Fatalf("expected opening balance 10000, got %v", ps.OpeningBalance)
	}
	if len(ps.RawItems) != 4 {
		t.Fatalf("expected 4 parsed items, got %d", len(ps.RawItems))
	}

	// Verify supplement price override (1320 -> D3 FORTE)
	foundD3 := false
	for _, it := range ps.RawItems {
		if it.ItemName == "D3 FORTE" && it.Category == "medicine" {
			foundD3 = true
		}
	}
	if !foundD3 {
		t.Fatalf("expected supplement rate 1320.00 to map to D3 FORTE medicine, items=%+v", ps.RawItems)
	}

	if len(ps.Discounts) != 1 || math.Abs(ps.Discounts[0].Amount-3000.0) > 0.01 {
		t.Fatalf("expected 1 discount of 3000, got %+v", ps.Discounts)
	}
	if len(ps.Payments) != 1 || math.Abs(ps.Payments[0].Amount-50000.0) > 0.01 {
		t.Fatalf("expected 1 payment of 50000, got %+v", ps.Payments)
	}
	if len(ps.TDSEntries) != 1 || math.Abs(ps.TDSEntries[0].Amount-1350.0) > 0.01 {
		t.Fatalf("expected 1 TDS of 1350, got %+v", ps.TDSEntries)
	}

	y, m := deriveLedgerYearMonth("26_11_NOV_QA_SAMPLE.pdf", ps, 0, 0)
	if y != 2026 || m != 11 {
		t.Fatalf("expected 2026-11 from filename, got %d-%d", y, m)
	}
}
