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

func TestParseOctober2026SakthiLedgerPatterns(t *testing.T) {
	lines := []string{
		"SAKTHI POULTRY FEEDS - FY 2026-27",
		"Pradeep PF",
		"Ledger Account",
		"1-Oct-26 to 31-Oct-26",
		"1-Oct-26 By Opening Balance 3,31,917.090",
		"By Rate Difference Credit Note 151 32,682.000",
		"By Rate Difference Credit Note 152 7,844.000",
		"To Rate Difference Debit Note 1 33,200.000",
		"To Poultry Feed Sales FEED SALES 1086 4,69,644.210",
		"Sylo -L M 12,630.000 KGS 36.667/KGS 4,63,104.210",
		"Feed",
		"Supplements",
		"6,540.000",
		"10-Oct-26 To Cash Payment 2259 3,00,000.000",
		"To Poultry Feed Sales FEED SALES 1119 4,67,963.200",
		"Sylo -L M 9,440.000 KGS 35.067/KGS 3,31,032.480",
		"Sylo - C M 3,160.000 KGS 41.467/KGS 1,31,035.720",
		"31-Oct-26 By Purchase Eggs EGG PURCHASE 1862 2,24,167.500",
		"Large Egg 40,500 NOS 5.535/NOS 2,24,167.500",
		"By Poultry Feed Sales Credit Note 159 42,000.000",
		"To TDS 194 Q Journal 604 5,531.000",
		"To Closing Balance 7,77,638.360",
	}

	ps := parseStatementLines(lines)

	if ps.OpeningBalance == nil || math.Abs(*ps.OpeningBalance-331917.09) > 0.01 {
		t.Fatalf("expected opening balance 331917.09, got %v", ps.OpeningBalance)
	}
	if ps.ClosingBalance == nil || math.Abs(*ps.ClosingBalance-777638.36) > 0.01 {
		t.Fatalf("expected closing balance 777638.36, got %v", ps.ClosingBalance)
	}
	if len(ps.Discounts) != 3 {
		t.Fatalf("expected 3 credit-note discounts (32682, 7844, 42000), got %d: %+v", len(ps.Discounts), ps.Discounts)
	}
	if len(ps.Payments) != 1 || math.Abs(ps.Payments[0].Amount-300000.0) > 0.01 {
		t.Fatalf("expected 1 payment of 300000, got %+v", ps.Payments)
	}
	if len(ps.TDSEntries) != 1 || math.Abs(ps.TDSEntries[0].Amount-5531.0) > 0.01 {
		t.Fatalf("expected 1 TDS of 5531, got %+v", ps.TDSEntries)
	}

	var foundChickFeed, foundLayerFeed, foundSupp, foundDebitNote bool
	for _, it := range ps.RawItems {
		if it.Category == "feed" {
			stage, _ := classifyFeedStage(it.ItemName)
			if stage == "CHICK" && math.Abs(it.Amount-131035.72) < 0.01 {
				foundChickFeed = true
			}
			if stage == "LAYER" {
				foundLayerFeed = true
			}
		}
		if it.Category == "medicine" && it.ItemName == "FEED SUPPLEMENTS" && math.Abs(it.Amount-6540.0) < 0.01 {
			foundSupp = true
		}
		if it.ItemName == "RATE DIFFERENCE DEBIT NOTE" && math.Abs(it.Amount-33200.0) < 0.01 {
			foundDebitNote = true
		}
	}
	if !foundChickFeed || !foundLayerFeed || !foundSupp || !foundDebitNote {
		t.Fatalf("missing expected parsed items: chick=%v layer=%v supp=%v debit=%v items=%+v",
			foundChickFeed, foundLayerFeed, foundSupp, foundDebitNote, ps.RawItems)
	}
}
