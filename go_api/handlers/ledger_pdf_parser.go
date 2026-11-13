package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"poultry-farm-api/database"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
)

var (
	eggKeywords = []string{"LARGE EGG", "MEDIUM EGG", "SMALL EGG", "CORRECT EGG", "EXPORT EGG", "CORRECT SIZE"}
	feedKeywords = []string{
		"LAYER MASH", "GROWER MASH", "PRE LAYER MASH",
		"LAYER MASH BULK", "GROWER MASH BULK", "PRE LAYER MASH BULK",
		"SYLO", "LCC", "LAYER CHICKS CRUMBLE",
	}
	medicinesKnown = []string{
		"D3 FORTE", "VETMULIN", "OXYCYCLINE", "TIAZIN", "BPPS FORTE", "CTC", "SHELL GRIT",
		"ROVIMIX", "CHOLIMARIN", "ZAGROMIN", "G PRO NATURO", "NECROVET", "TOXOL",
		"FRA C12 DRY", "CALCI ROYAL FS", "CALDLIV FS", "RESPAFEED", "VENTRIM (VITAL)",
		"SOYA", "FARM GRIT",
	}
	medicinePriceMap = map[float64]string{
		1320.0: "D3 FORTE",
		1250.0: "VETMULIN",
		1375.0: "OXYCYCLINE",
		1150.0: "TIAZIN",
		850.0:  "NECROVET",
		85.0:   "TOXOL",
		440.0:  "G PRO NATURO",
		10.50:  "SHELL GRIT",
		650.0:  "ROVIMIX",
		100.0:  "CHOLIMARIN",
		260.0:  "ZAGROMIN",
		40.0:   "SOYA",
		265.0:  "CTC",
	}
	numTokenRe     = regexp.MustCompile(`[\d,]+\.\d+|[\d,]+`)
	dateHeaderRe   = regexp.MustCompile(`^(\d{1,2}-[A-Za-z]{3}-\d{2,4})\b`)
	discountMultRe = regexp.MustCompile(`([\d,]+(?:\.\d+)?)\s*[xX\*]\s*([\d,]+(?:\.\d+)?)`)
	billNoRe       = regexp.MustCompile(`BILL\s*NO\.?\s*(\d+)|BILLNO\s*(\d+)`)
	toTdsRe        = regexp.MustCompile(`\bTO\s+TDS\b`)
	tdsWordRe      = regexp.MustCompile(`\bTDS\b`)
	splitFeedRe    = regexp.MustCompile(`(?i)^FEED\s+([\d,]+(?:\.\d+)?)$`)
	filenameYMRe   = regexp.MustCompile(`(?i)(\d{2})_(\d{2})_([A-Z]{3})`)
	autoEggRe      = regexp.MustCompile(`^(LARGE|SMALL|MEDIUM)(\s|$)`)
)

type parsedLedgerItem struct {
	Date         time.Time `json:"date"`
	TxnType      string    `json:"txn_type"`
	Category     string    `json:"category"` // egg, feed, medicine, other
	ItemName     string    `json:"item_name"`
	Qty          *float64  `json:"qty,omitempty"`
	Unit         string    `json:"unit,omitempty"`
	Rate         *float64  `json:"rate,omitempty"`
	Amount       float64   `json:"amount"`
	BillNo       string    `json:"bill_no,omitempty"`
	ParsingNotes string    `json:"parsing_notes,omitempty"`
}

type parsedLedgerAmountEntry struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
	BillNo string    `json:"bill_no,omitempty"`
	Raw    string    `json:"raw"`
}

type parsedAggRow struct {
	ItemName    string  `json:"item_name"`
	TotalQty    float64 `json:"total_qty"`
	TotalAmount float64 `json:"total_amount"`
}

type parsedStatement struct {
	RawItems       []parsedLedgerItem
	Payments       []parsedLedgerAmountEntry
	Discounts      []parsedLedgerAmountEntry
	TDSEntries     []parsedLedgerAmountEntry
	OpeningBalance *float64
	ClosingBalance *float64
	ProviderName   string
	AccountHolder  string
	LedgerPeriod   string
	EggsAgg        []parsedAggRow
	FeedsAgg       []parsedAggRow
	MedsAgg        []parsedAggRow
	OtherAgg       []parsedAggRow
}

// ParseAndCommitLedgerPDF extracts text from a PDF ledger natively in Go,
// classifies items/payments/discounts/TDS, and optionally commits to PostgreSQL.
func ParseAndCommitLedgerPDF(filePath string, tenantID uuid.UUID, commit bool, fallbackYear int, fallbackMonth int) (map[string]interface{}, error) {
	lines, err := extractPDFLines(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %v", err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("no text lines found in PDF")
	}

	parsed := parseStatementLines(lines)
	year, month := deriveLedgerYearMonth(filepath.Base(filePath), parsed, fallbackYear, fallbackMonth)

	var totalEggs, totalFeeds, totalMeds, totalOther float64
	var eggsLargeQty, eggsMediumQty, eggsSmallQty, feedsTotalKg float64

	for _, it := range parsed.RawItems {
		switch it.Category {
		case "egg":
			totalEggs += it.Amount
			if it.Qty != nil {
				upper := strings.ToUpper(it.ItemName)
				if strings.Contains(upper, "LARGE") {
					eggsLargeQty += *it.Qty
				} else if strings.Contains(upper, "MEDIUM") {
					eggsMediumQty += *it.Qty
				} else if strings.Contains(upper, "SMALL") {
					eggsSmallQty += *it.Qty
				}
			}
		case "feed":
			totalFeeds += it.Amount
			if it.Qty != nil {
				u := strings.ToUpper(it.Unit)
				if u == "BAG" || u == "BAGS" {
					feedsTotalKg += (*it.Qty) * 50.0
				} else {
					feedsTotalKg += *it.Qty
				}
			}
		case "medicine":
			totalMeds += it.Amount
		default:
			totalOther += it.Amount
		}
	}

	var totalPayments, totalDiscounts, totalTDS float64
	for _, p := range parsed.Payments {
		totalPayments += p.Amount
	}
	for _, d := range parsed.Discounts {
		totalDiscounts += d.Amount
	}
	for _, t := range parsed.TDSEntries {
		totalTDS += t.Amount
	}

	netProfit := round2(totalEggs + totalDiscounts - totalFeeds - totalMeds - totalOther - totalTDS)
	summary := map[string]interface{}{
		"total_eggs":      round2(totalEggs),
		"total_feeds":     round2(totalFeeds),
		"total_medicines": round2(totalMeds),
		"total_other":     round2(totalOther),
		"total_payments":  round2(totalPayments),
		"total_discounts": round2(totalDiscounts),
		"total_tds":       round2(totalTDS),
		"net_profit":      netProfit,
		"opening_balance": parsed.OpeningBalance,
		"closing_balance": parsed.ClosingBalance,
		"provider_name":   parsed.ProviderName,
		"account_holder":  parsed.AccountHolder,
		"ledger_period":   parsed.LedgerPeriod,
	}

	pdfName := filepath.Base(filePath)
	result := map[string]interface{}{
		"success":      true,
		"parser":       "go_native",
		"pdf_filename": pdfName,
		"year":         year,
		"month":        month,
		"summary":      summary,
		"eggs":         parsed.EggsAgg,
		"feeds":        parsed.FeedsAgg,
		"medicines":    parsed.MedsAgg,
		"payments":     parsed.Payments,
		"discounts":    parsed.Discounts,
		"dry_run":      !commit,
		"committed":    false,
	}

	if !commit {
		return result, nil
	}

	ledgerParseID, err := commitParsedLedgerToDB(
		tenantID, pdfName, year, month, summary, parsed,
		eggsLargeQty, eggsMediumQty, eggsSmallQty, feedsTotalKg,
	)
	if err != nil {
		return result, fmt.Errorf("failed to commit parsed ledger: %v", err)
	}

	InvalidateAnalyticsCache(tenantID.String())
	result["committed"] = true
	result["ledger_parse_id"] = ledgerParseID
	return result, nil
}

func extractPDFLines(filePath string) ([]string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	numPages := r.NumPage()
	for pageIdx := 1; pageIdx <= numPages; pageIdx++ {
		p := r.Page(pageIdx)
		if p.V.IsNull() {
			continue
		}
		rows, err := p.GetTextByRow()
		if err != nil {
			continue
		}
		for _, row := range rows {
			words := make([]pdf.Text, len(row.Content))
			copy(words, row.Content)
			sort.Slice(words, func(i, j int) bool {
				return words[i].X < words[j].X
			})
			var parts []string
			for _, w := range words {
				s := strings.TrimSpace(w.S)
				if s != "" {
					parts = append(parts, s)
				}
			}
			joined := strings.TrimSpace(strings.Join(parts, " "))
			if joined != "" {
				lines = append(lines, joined)
			}
		}
	}
	return lines, nil
}

func parseStatementLines(allLines []string) *parsedStatement {
	ps := &parsedStatement{}
	extractProviderMetadata(allLines, ps)

	var lastBillNo string
	i := 0
	for i < len(allLines) {
		line := strings.TrimSpace(allLines[i])
		m := dateHeaderRe.FindStringSubmatchIndex(line)
		if m == nil {
			i++
			continue
		}
		dateStr := line[m[2]:m[3]]
		parsedDate, ok := parseLedgerDate(dateStr)
		if !ok {
			i++
			continue
		}
		rest := strings.TrimSpace(line[m[1]:])
		headerText := strings.ToUpper(rest)
		headerAmt := extractHeaderAmount(headerText)

		txnType := "other"
		switch {
		case strings.Contains(headerText, "CREDIT NOTE") || strings.Contains(headerText, "DISCOUNT"):
			txnType = "discount"
		case strings.Contains(headerText, "POULTRY EGG") || strings.Contains(headerText, "EGG PURCHASE") || strings.Contains(headerText, "PURCHASE EGGS"):
			txnType = "egg_purchase"
		case strings.Contains(headerText, "POULTRY FEED") || strings.Contains(headerText, "FEED SALES"):
			txnType = "feed_sale"
		case strings.Contains(headerText, "CANARA BANK") || strings.Contains(headerText, "PAYMENT"):
			txnType = "payment"
		case tdsWordRe.MatchString(headerText) && !strings.Contains(headerText, "DEDUCTED"):
			txnType = "tds"
		case strings.Contains(headerText, "OPENING BALANCE"):
			txnType = "opening_balance"
		case strings.Contains(headerText, "CLOSING BALANCE"):
			txnType = "closing_balance"
		}

		if bm := billNoRe.FindStringSubmatch(headerText); len(bm) >= 3 {
			if bm[1] != "" {
				lastBillNo = bm[1]
			} else if bm[2] != "" {
				lastBillNo = bm[2]
			}
		}

		if txnType == "payment" && headerAmt != nil {
			ps.Payments = append(ps.Payments, parsedLedgerAmountEntry{Date: parsedDate, Amount: *headerAmt, Raw: line})
		}
		if txnType == "tds" && headerAmt != nil {
			ps.TDSEntries = append(ps.TDSEntries, parsedLedgerAmountEntry{Date: parsedDate, Amount: *headerAmt, BillNo: lastBillNo, Raw: line})
		}
		if txnType == "discount" && headerAmt != nil {
			ps.Discounts = append(ps.Discounts, parsedLedgerAmountEntry{Date: parsedDate, Amount: *headerAmt, Raw: line})
		}
		if txnType == "opening_balance" && headerAmt != nil {
			val := *headerAmt
			ps.OpeningBalance = &val
		}
		if txnType == "closing_balance" && headerAmt != nil {
			val := *headerAmt
			ps.ClosingBalance = &val
		}

		// Scan sub-lines until next date header
		j := i + 1
		for j < len(allLines) {
			nxt := strings.TrimSpace(allLines[j])
			if nxt == "" {
				j++
				continue
			}
			if dateHeaderRe.MatchString(nxt) {
				break
			}
			nxtUpper := strings.ToUpper(nxt)
			if bm := billNoRe.FindStringSubmatch(nxtUpper); len(bm) >= 3 {
				if bm[1] != "" {
					lastBillNo = bm[1]
				} else if bm[2] != "" {
					lastBillNo = bm[2]
				}
			}

			// Handle dateless voucher headers on the same date (e.g. "By Rate Difference Credit Note 151 32,682.000",
			// "To Rate Difference Debit Note 1 33,200.000", "By Poultry Feed Sales Credit Note 159 42,000.000", "To Closing Balance 7,77,638.360")
			if strings.HasPrefix(nxtUpper, "BY ") || strings.HasPrefix(nxtUpper, "TO ") {
				subAmt := extractHeaderAmount(nxtUpper)
				handledDateless := false
				switch {
				case strings.Contains(nxtUpper, "CLOSING BALANCE") && subAmt != nil:
					val := *subAmt
					ps.ClosingBalance = &val
					handledDateless = true
				case strings.Contains(nxtUpper, "OPENING BALANCE") && subAmt != nil:
					val := *subAmt
					ps.OpeningBalance = &val
					handledDateless = true
				case (strings.Contains(nxtUpper, "CREDIT NOTE") || strings.Contains(nxtUpper, "DISCOUNT")) && subAmt != nil:
					ps.Discounts = append(ps.Discounts, parsedLedgerAmountEntry{
						Date: parsedDate, Amount: *subAmt, Raw: nxt,
					})
					handledDateless = true
				case strings.Contains(nxtUpper, "DEBIT NOTE") && subAmt != nil:
					ps.RawItems = append(ps.RawItems, parsedLedgerItem{
						Date: parsedDate, TxnType: "debit_note", Category: "other",
						ItemName: "RATE DIFFERENCE DEBIT NOTE", Amount: *subAmt, BillNo: lastBillNo,
						ParsingNotes: "debit_note_header",
					})
					handledDateless = true
				case (strings.Contains(nxtUpper, "CANARA BANK") || strings.Contains(nxtUpper, "PAYMENT") || strings.HasPrefix(nxtUpper, "TO CASH")) && subAmt != nil:
					ps.Payments = append(ps.Payments, parsedLedgerAmountEntry{
						Date: parsedDate, Amount: *subAmt, Raw: nxt,
					})
					handledDateless = true
				}
				if handledDateless {
					j++
					continue
				}
			}

			if toTdsRe.MatchString(nxtUpper) || strings.HasPrefix(nxtUpper, "TO TDS") {
				tokens := strings.Fields(nxt)
				if _, amtTok := findLastAmountToken(tokens); amtTok != "" {
					if amt, ok := cleanLedgerNumber(amtTok); ok {
						ps.TDSEntries = append(ps.TDSEntries, parsedLedgerAmountEntry{
							Date: parsedDate, Amount: amt, BillNo: lastBillNo, Raw: nxt,
						})
					}
				}
			}

			// Format A: "Feed 6,540.000" followed by "Supplements"
			if sm := splitFeedRe.FindStringSubmatch(nxt); len(sm) == 2 && j+1 < len(allLines) {
				nxt2 := strings.ToUpper(strings.TrimSpace(allLines[j+1]))
				if strings.Contains(nxt2, "SUPPLEMENT") {
					if suppAmt, ok := cleanLedgerNumber(sm[1]); ok {
						ps.RawItems = append(ps.RawItems, parsedLedgerItem{
							Date: parsedDate, TxnType: txnType, Category: "medicine",
							ItemName: "FEED SUPPLEMENTS", Amount: suppAmt, BillNo: lastBillNo,
							ParsingNotes: "split_feed_supplement",
						})
					}
					j += 2
					continue
				}
			}

			// Format B: "Feed" on line j, "Supplements" on line j+1, "6,540.000" on line j+2
			if strings.EqualFold(nxt, "FEED") && j+2 < len(allLines) {
				nxt2 := strings.ToUpper(strings.TrimSpace(allLines[j+1]))
				nxt3 := strings.TrimSpace(allLines[j+2])
				if strings.Contains(nxt2, "SUPPLEMENT") {
					if suppAmt, ok := cleanLedgerNumber(nxt3); ok {
						ps.RawItems = append(ps.RawItems, parsedLedgerItem{
							Date: parsedDate, TxnType: txnType, Category: "medicine",
							ItemName: "FEED SUPPLEMENTS", Amount: suppAmt, BillNo: lastBillNo,
							ParsingNotes: "split_feed_supplement_3line",
						})
						j += 3
						continue
					}
				}
			}

			if strings.Contains(nxtUpper, "/KGS") || strings.Contains(nxtUpper, "/NOS") || strings.Contains(nxtUpper, "/BAG") {
				if item, ok := parseLedgerItemLine(nxt); ok && item.ItemName != "" {
					mappedMed := applySupplementPriceOverride(item)
					cat, autoNote := determineLedgerCategory(item.ItemName, item.Unit)
					var notes []string
					if item.ParsingNotes != "" {
						notes = append(notes, item.ParsingNotes)
					}
					if mappedMed != "" {
						notes = append(notes, "supplement_rate_mapped="+mappedMed)
					}
					if autoNote != "" {
						notes = append(notes, autoNote)
					}
					item.Date = parsedDate
					item.TxnType = txnType
					item.Category = cat
					item.BillNo = lastBillNo
					item.ParsingNotes = strings.Join(notes, ";")
					ps.RawItems = append(ps.RawItems, *item)
				}
			} else if strings.Contains(nxtUpper, "DISCOUNT") || strings.Contains(nxtUpper, "CREDIT NOTE") {
				if discAmt := extractDiscountAmount(nxt); discAmt != nil {
					ps.Discounts = append(ps.Discounts, parsedLedgerAmountEntry{
						Date: parsedDate, Amount: *discAmt, Raw: nxt,
					})
				}
			}
			j++
		}
		i = j
	}

	ps.EggsAgg = aggregateCategory(ps.RawItems, "egg")
	ps.FeedsAgg = aggregateCategory(ps.RawItems, "feed")
	ps.MedsAgg = aggregateCategory(ps.RawItems, "medicine")
	ps.OtherAgg = aggregateCategory(ps.RawItems, "other")
	return ps
}

func extractProviderMetadata(lines []string, ps *parsedStatement) {
	for _, raw := range lines {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if ps.ProviderName == "" && !dateHeaderRe.MatchString(s) {
			ps.ProviderName = s
		}
		upper := strings.ToUpper(s)
		if strings.Contains(upper, "LEDGER ACCOUNT") {
			if idx := strings.Index(s, ":"); idx != -1 {
				ps.AccountHolder = strings.TrimSpace(s[idx+1:])
			}
		}
		if strings.Contains(upper, " TO ") && dateHeaderRe.MatchString(s) {
			ps.LedgerPeriod = s
		}
	}
}

func parseLedgerDate(s string) (time.Time, bool) {
	for _, layout := range []string{"2-Jan-06", "02-Jan-06", "2-Jan-2006", "02-Jan-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func cleanLedgerNumber(tok string) (float64, bool) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(tok), ",", "")
	if cleaned == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func extractHeaderAmount(headerText string) *float64 {
	matches := numTokenRe.FindAllString(headerText, -1)
	if len(matches) == 0 {
		return nil
	}
	for _, m := range matches {
		if strings.Contains(m, ".") {
			if v, ok := cleanLedgerNumber(m); ok {
				return &v
			}
		}
	}
	if v, ok := cleanLedgerNumber(matches[len(matches)-1]); ok {
		return &v
	}
	return nil
}

func findLastAmountToken(tokens []string) (int, string) {
	skipPrev := map[string]bool{
		"BILL": true, "BILLNO": true, "BILLNO:": true,
		"NO": true, "INV": true, "INVOICE": true,
		"SL": true, "SR": true, "SL.": true, "SR.": true,
	}
	for i := len(tokens) - 1; i >= 0; i-- {
		if numTokenRe.MatchString(tokens[i]) {
			if i > 0 && skipPrev[strings.ToUpper(tokens[i-1])] {
				continue
			}
			return i, tokens[i]
		}
	}
	return -1, ""
}

func parseLedgerItemLine(line string) (*parsedLedgerItem, bool) {
	tokens := strings.Fields(strings.TrimSpace(line))
	idxAmt, amtTok := findLastAmountToken(tokens)
	if idxAmt < 0 {
		return nil, false
	}
	amount, ok := cleanLedgerNumber(amtTok)
	if !ok {
		return nil, false
	}
	amountSource := "last_token"

	rateIdx := -1
	for i, t := range tokens {
		if strings.Contains(t, "/") {
			rateIdx = i
			break
		}
	}

	var qty *float64
	var rate *float64
	var unit, itemName string

	if rateIdx >= 0 {
		rateParts := strings.SplitN(tokens[rateIdx], "/", 2)
		if rVal, ok := cleanLedgerNumber(rateParts[0]); ok {
			rate = &rVal
		}
		j := rateIdx - 1
		if j >= 0 && isUnitToken(tokens[j]) {
			unit = strings.ToUpper(tokens[j])
			if j-1 >= 0 && numTokenRe.MatchString(tokens[j-1]) {
				if qVal, ok := cleanLedgerNumber(tokens[j-1]); ok {
					qty = &qVal
				}
				itemName = strings.Join(tokens[:j-1], " ")
			} else {
				itemName = strings.Join(tokens[:j], " ")
			}
		} else if j >= 0 && numTokenRe.MatchString(tokens[j]) {
			if qVal, ok := cleanLedgerNumber(tokens[j]); ok {
				qty = &qVal
			}
			if len(rateParts) == 2 {
				unit = strings.ToUpper(rateParts[1])
			}
			itemName = strings.Join(tokens[:j], " ")
		} else {
			itemName = strings.Join(tokens[:idxAmt], " ")
		}
	} else {
		itemName = strings.Join(tokens[:idxAmt], " ")
	}

	if rate != nil && qty != nil {
		expected := round2((*qty) * (*rate))
		for i := len(tokens) - 1; i >= 0; i-- {
			if numTokenRe.MatchString(tokens[i]) {
				if tv, ok := cleanLedgerNumber(tokens[i]); ok && math.Abs(tv-expected) < 0.01 {
					amount = tv
					amountSource = "qty_rate"
					break
				}
			}
		}
	}

	return &parsedLedgerItem{
		ItemName:     strings.ToUpper(strings.TrimSpace(itemName)),
		Qty:          qty,
		Unit:         unit,
		Rate:         rate,
		Amount:       amount,
		ParsingNotes: "amount_source=" + amountSource,
	}, true
}

func isUnitToken(tok string) bool {
	switch strings.ToLower(tok) {
	case "kgs", "kg", "nos", "no", "nos.", "bag", "bags":
		return true
	default:
		return false
	}
}

func applySupplementPriceOverride(item *parsedLedgerItem) string {
	if item == nil || item.Rate == nil || !strings.Contains(strings.ToUpper(item.ItemName), "SUPPL") {
		return ""
	}
	for price, med := range medicinePriceMap {
		if math.Abs(*item.Rate-price) < 0.01 {
			item.ItemName = strings.ToUpper(med)
			return med
		}
	}
	return ""
}

func classifyItemByName(name string) string {
	n := strings.ToUpper(strings.TrimSpace(name))
	if strings.Contains(n, "SUPPL") {
		return "medicine"
	}
	for _, k := range eggKeywords {
		if strings.Contains(n, k) {
			return "egg"
		}
	}
	for _, k := range feedKeywords {
		if strings.Contains(n, k) {
			return "feed"
		}
	}
	for _, k := range medicinesKnown {
		if strings.Contains(n, k) {
			return "medicine"
		}
	}
	for _, k := range []string{"GRIT", "D3", "FRA", "VET", "CTC", "NECRO", "TOX", "ROVIMIX"} {
		if strings.Contains(n, k) {
			return "medicine"
		}
	}
	return "other"
}

func determineLedgerCategory(itemName string, unit string) (string, string) {
	base := classifyItemByName(itemName)
	u := strings.ToUpper(unit)
	if base != "egg" && autoEggRe.MatchString(strings.ToUpper(strings.TrimSpace(itemName))) && (u == "NOS" || u == "NO") {
		return "egg", "auto_classified_to_egg"
	}
	return base, ""
}

func extractDiscountAmount(line string) *float64 {
	if m := discountMultRe.FindStringSubmatch(line); len(m) == 3 {
		q, ok1 := cleanLedgerNumber(m[1])
		r, ok2 := cleanLedgerNumber(m[2])
		if ok1 && ok2 {
			v := round2(q * r)
			return &v
		}
	}
	matches := numTokenRe.FindAllString(line, -1)
	var maxVal float64
	found := false
	for _, tok := range matches {
		if v, ok := cleanLedgerNumber(tok); ok {
			if !found || v > maxVal {
				maxVal = v
				found = true
			}
		}
	}
	if !found {
		return nil
	}
	return &maxVal
}

func aggregateCategory(items []parsedLedgerItem, category string) []parsedAggRow {
	type acc struct {
		qty float64
		amt float64
	}
	byName := map[string]*acc{}
	for _, it := range items {
		if it.Category != category {
			continue
		}
		entry, ok := byName[it.ItemName]
		if !ok {
			entry = &acc{}
			byName[it.ItemName] = entry
		}
		if it.Qty != nil {
			entry.qty += *it.Qty
		}
		entry.amt += it.Amount
	}
	var out []parsedAggRow
	for name, a := range byName {
		out = append(out, parsedAggRow{
			ItemName:    name,
			TotalQty:    round2(a.qty),
			TotalAmount: round2(a.amt),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ItemName < out[j].ItemName })
	return out
}

func deriveLedgerYearMonth(filename string, ps *parsedStatement, fallbackYear int, fallbackMonth int) (int, int) {
	if m := filenameYMRe.FindStringSubmatch(strings.ToUpper(filename)); len(m) == 4 {
		yy, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		if yy >= 20 && mm >= 1 && mm <= 12 {
			return 2000 + yy, mm
		}
	}
	for _, it := range ps.RawItems {
		return it.Date.Year(), int(it.Date.Month())
	}
	for _, p := range ps.Payments {
		return p.Date.Year(), int(p.Date.Month())
	}
	if fallbackYear > 2000 && fallbackMonth >= 1 && fallbackMonth <= 12 {
		return fallbackYear, fallbackMonth
	}
	now := time.Now()
	return now.Year(), int(now.Month())
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func commitParsedLedgerToDB(
	tenantID uuid.UUID,
	pdfName string,
	year int,
	month int,
	summary map[string]interface{},
	ps *parsedStatement,
	eggsLargeQty, eggsMediumQty, eggsSmallQty, feedsTotalKg float64,
) (int, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var ledgerParseID int
	err = tx.QueryRow(`
		INSERT INTO ledger_parses (
			tenant_id, pdf_filename, parse_date, month, year,
			opening_balance, closing_balance,
			total_eggs, total_feeds, total_medicines, net_profit,
			eggs_large_qty, eggs_medium_qty, eggs_small_qty, feeds_total_kg,
			account_holder, ledger_period
		)
		VALUES ($1, $2, CURRENT_DATE, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id
	`,
		tenantID, pdfName, month, year,
		ps.OpeningBalance, ps.ClosingBalance,
		summary["total_eggs"], summary["total_feeds"], summary["total_medicines"], summary["net_profit"],
		eggsLargeQty, eggsMediumQty, eggsSmallQty, feedsTotalKg,
		nullIfEmptyStr(ps.AccountHolder), nullIfEmptyStr(ps.LedgerPeriod),
	).Scan(&ledgerParseID)
	if err != nil {
		return 0, err
	}

	insertBreakdownGroup := func(prefix string, rows []parsedAggRow) error {
		for _, r := range rows {
			if strings.TrimSpace(r.ItemName) == "" {
				continue
			}
			_, err := tx.Exec(`
				INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
				VALUES ($1, $2, $3)
			`, ledgerParseID, prefix+"_"+strings.ToUpper(r.ItemName), r.TotalQty)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := insertBreakdownGroup("EGG", ps.EggsAgg); err != nil {
		return 0, err
	}
	if err := insertBreakdownGroup("FEED", ps.FeedsAgg); err != nil {
		return 0, err
	}
	if err := insertBreakdownGroup("MEDICINE", ps.MedsAgg); err != nil {
		return 0, err
	}
	if err := insertBreakdownGroup("OTHER", ps.OtherAgg); err != nil {
		return 0, err
	}

	importNote := "Ledger import: " + pdfName
	if _, err := tx.Exec(`DELETE FROM transactions WHERE tenant_id = $1 AND notes = $2`, tenantID, importNote); err != nil {
		return 0, err
	}

	for _, it := range ps.RawItems {
		txnType := "EXPENSE"
		cat := "OTHER"
		switch it.Category {
		case "egg":
			txnType, cat = "SALE", "EGG"
		case "feed":
			txnType, cat = "PURCHASE", "FEED"
		case "medicine":
			txnType, cat = "PURCHASE", "MEDICINE"
		}
		var qtyVal, rateVal sql.NullFloat64
		if it.Qty != nil {
			qtyVal = sql.NullFloat64{Float64: *it.Qty, Valid: true}
		}
		if it.Rate != nil {
			rateVal = sql.NullFloat64{Float64: *it.Rate, Valid: true}
		}
		if _, err := tx.Exec(`
			INSERT INTO transactions (
				tenant_id, transaction_date, transaction_type, category,
				item_name, quantity, unit, rate, amount, notes
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, tenantID, it.Date, txnType, cat, it.ItemName, qtyVal, nullIfEmptyStr(it.Unit), rateVal, it.Amount, importNote); err != nil {
			return 0, err
		}
	}

	insertAmountEntries := func(entries []parsedLedgerAmountEntry, txnType string, label string) error {
		for _, e := range entries {
			if _, err := tx.Exec(`
				INSERT INTO transactions (
					tenant_id, transaction_date, transaction_type, category,
					item_name, amount, notes
				)
				VALUES ($1, $2, $3, 'OTHER', $4, $5, $6)
			`, tenantID, e.Date, txnType, label, e.Amount, importNote); err != nil {
				return err
			}
		}
		return nil
	}

	if err := insertAmountEntries(ps.Payments, "PAYMENT", "Payment"); err != nil {
		return 0, err
	}
	if err := insertAmountEntries(ps.Discounts, "DISCOUNT", "Discount"); err != nil {
		return 0, err
	}
	if err := insertAmountEntries(ps.TDSEntries, "TDS", "Tds"); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return ledgerParseID, nil
}

func nullIfEmptyStr(s string) sql.NullString {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}
