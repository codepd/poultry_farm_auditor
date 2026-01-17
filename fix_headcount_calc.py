#!/usr/bin/env python3
"""Fix head count calculation in analytics.go - direct line replacement"""

file_path = 'go_api/handlers/analytics.go'

with open(file_path, 'r') as f:
    lines = f.readlines()

# The new calculation code
new_calc_code = '''	// Calculate actual head count based on daily counts for the month
	// Strategy: For each day in the month, calculate head count by:
	// 1. For each batch that existed on that day (date_added <= day)
	// 2. Start with initial_count and subtract all mortalities up to that day
	// 3. Sum all daily head counts and divide by days in month to get average
	var actualHeadCount float64 // Average daily head count (for display and calculation)
	usingActualCount := false

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0) // Start of next month

	// Get all batches that existed at any point during this month
	batchRows, err := database.DB.Query(`
		SELECT id, date_added, initial_count
		FROM hen_batches
		WHERE tenant_id = $1
		  AND date_added <= $2
		ORDER BY date_added
	`, tenantID, monthEnd)

	if err == nil {
		defer batchRows.Close()

		// Store batch information
		type BatchInfo struct {
			ID           int
			DateAdded    time.Time
			InitialCount int
		}
		var batches []BatchInfo

		for batchRows.Next() {
			var batch BatchInfo
			if err := batchRows.Scan(&batch.ID, &batch.DateAdded, &batch.InitialCount); err != nil {
				continue
			}
			batches = append(batches, batch)
		}

		if len(batches) > 0 {
			// Get all mortality events for these batches up to month end
			batchIDs := make([]interface{}, len(batches))
			for i, b := range batches {
				batchIDs[i] = b.ID
			}

			// Build query for mortality events
			placeholders := ""
			for i := range batchIDs {
				if i > 0 {
					placeholders += ","
				}
				placeholders += fmt.Sprintf("$%d", i+1)
			}

			mortalityQuery := fmt.Sprintf(`
				SELECT batch_id, mortality_date, count
				FROM hen_mortality
				WHERE batch_id IN (%s)
				  AND mortality_date < $%d
				ORDER BY batch_id, mortality_date
			`, placeholders, len(batchIDs)+1)

			mortalityArgs := make([]interface{}, len(batchIDs)+1)
			copy(mortalityArgs, batchIDs)
			mortalityArgs[len(batchIDs)] = monthEnd

			mortalityRows, err := database.DB.Query(mortalityQuery, mortalityArgs...)
			if err == nil {
				defer mortalityRows.Close()

				// Store mortality by batch_id and date
				// Key: batch_id, Value: map of date -> count
				mortalityByBatch := make(map[int]map[time.Time]int)

				for mortalityRows.Next() {
					var batchID int
					var mortalityDate time.Time
					var count int
					if err := mortalityRows.Scan(&batchID, &mortalityDate, &count); err != nil {
						continue
					}

					if mortalityByBatch[batchID] == nil {
						mortalityByBatch[batchID] = make(map[time.Time]int)
					}
					mortalityByBatch[batchID][mortalityDate] += count
				}

				// Calculate total head count days (sum of head count for each day)
				var totalHeadCountDays float64

				// For each day in the month
				for day := 0; day < daysInMonth; day++ {
					currentDay := monthStart.AddDate(0, 0, day)
					dailyHeadCount := 0

					// For each batch, calculate head count on this day
					for _, batch := range batches {
						// Skip if batch didn't exist yet on this day
						if batch.DateAdded.After(currentDay) {
							continue
						}

						// Start with initial count
						batchCountOnDay := batch.InitialCount

						// Subtract all mortalities up to and including this day
						if mortalities, exists := mortalityByBatch[batch.ID]; exists {
							for mortDate, mortCount := range mortalities {
								if !mortDate.After(currentDay) {
									batchCountOnDay -= mortCount
								}
							}
						}

						// Only add if count is positive (avoid negative counts from data issues)
						if batchCountOnDay > 0 {
							dailyHeadCount += batchCountOnDay
						}
					}

					totalHeadCountDays += float64(dailyHeadCount)
				}

				// Calculate average daily head count
				if daysInMonth > 0 {
					actualHeadCount = totalHeadCountDays / float64(daysInMonth)
					usingActualCount = actualHeadCount > 0
				}
			}
		}
	}

'''

# Find replacement points
start_line = None
end_line = None
for i, line in enumerate(lines):
    if '// Try to get actual head count from hen_batches' in line:
        start_line = i
    if start_line and 'weightedAvgHeadCount = actualHeadCount // Fallback' in line:
        # Find the next closing brace that's not part of another block
        brace_count = 0
        for j in range(i, min(i+5, len(lines))):
            if '{' in lines[j]:
                brace_count += lines[j].count('{')
            if '}' in lines[j]:
                brace_count -= lines[j].count('}')
                if brace_count < 0 or (lines[j].strip() == '}' and j > i+1):
                    end_line = j + 1
                    break
        if end_line is None:
            end_line = i + 3  # Fallback
        break

if start_line is None or end_line is None:
    print("Could not find section to replace")
    exit(1)

print(f"Replacing lines {start_line+1} to {end_line}")

# Also find and replace the estimatedHens section
estimated_start = None
estimated_end = None
for i in range(end_line, min(end_line+15, len(lines))):
    if '// Use weighted average head count for egg percentage' in lines[i]:
        estimated_start = i
    if estimated_start and 'estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0' in lines[i]:
        estimated_end = i + 2  # Include the closing brace
        break

# Replace the calculation section
new_lines = lines[:start_line] + new_calc_code.splitlines(keepends=True) + lines[end_line:]

# Replace the estimatedHens section
if estimated_start and estimated_end:
    new_estimated = '''	// Use average daily head count for egg percentage calculation (more accurate)
	estimatedHens := 0.0

	if usingActualCount && actualHeadCount > 0 {
		// Use average daily head count for egg percentage calculation
		estimatedHens = actualHeadCount
	} else if feedPurchasedTonne > 0 && daysInMonth > 0 {
		// Fallback: Estimate hens: 10,000 hens consume 1 tonne per day
		estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0
	}
'''
    # Adjust indices since we already modified the array
    adj_estimated_start = estimated_start - (end_line - start_line) + len(new_calc_code.splitlines())
    adj_estimated_end = estimated_end - (end_line - start_line) + len(new_calc_code.splitlines())
    new_lines = new_lines[:adj_estimated_start] + new_estimated.splitlines(keepends=True) + new_lines[adj_estimated_end:]

with open(file_path, 'w') as f:
    f.writelines(new_lines)

print("Successfully updated head count calculation!")
