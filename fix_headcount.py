#!/usr/bin/env python3
"""Fix head count calculation in analytics.go"""

import re

file_path = 'go_api/handlers/analytics.go'

with open(file_path, 'r') as f:
    content = f.read()

# Replace the calculation section
old_pattern = r'// Try to get actual head count from hen_batches for this month.*?// Calculate weighted average for egg percentage.*?weightedAvgHeadCount = actualHeadCount // Fallback to total if no weighted calc\s+\}\s+\}'

new_code = '''// Calculate actual head count based on daily counts for the month
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
	}'''

# Use a more specific pattern matching approach
# Find the section and replace it
lines = content.split('\n')
new_lines = []
i = 0
in_old_section = False
skip_until = None

while i < len(lines):
    line = lines[i]
    
    # Detect start of old section
    if '// Try to get actual head count from hen_batches for this month' in line:
        in_old_section = True
        # Add new code
        new_lines.extend(new_code.split('\n'))
        # Skip until we find the end of the old section
        skip_until = '}'  # Looking for the closing brace after weightedAvgHeadCount assignment
        depth = 0
        i += 1
        while i < len(lines):
            if '{' in lines[i]:
                depth += lines[i].count('{')
            if '}' in lines[i]:
                depth -= lines[i].count('}')
                if depth < 0 or (depth == 0 and skip_until == '}' and 'weightedAvgHeadCount = actualHeadCount' in lines[i-1]):
                    i += 1  # Skip the closing brace
                    break
            i += 1
        in_old_section = False
        skip_until = None
        continue
    
    if not in_old_section:
        new_lines.append(line)
    i += 1

# Also replace the estimatedHens calculation section
final_content = '\n'.join(new_lines)

# Replace the estimatedHens section that uses weightedAvgHeadCount
old_estimated = '''	// Use weighted average head count for egg percentage calculation (more accurate)
	// but display total head count at month end
	estimatedHens := 0.0

	if usingActualCount && weightedAvgHeadCount > 0 {
		// Use weighted average for egg percentage calculation
		estimatedHens = weightedAvgHeadCount
	} else if usingActualCount && actualHeadCount > 0 {
		// Fallback to total if weighted average not available
		estimatedHens = actualHeadCount
	} else if feedPurchasedTonne > 0 && daysInMonth > 0 {
		// Fallback: Estimate hens: 10,000 hens consume 1 tonne per day
		estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0
	}'''

new_estimated = '''	// Use average daily head count for egg percentage calculation (more accurate)
	estimatedHens := 0.0

	if usingActualCount && actualHeadCount > 0 {
		// Use average daily head count for egg percentage calculation
		estimatedHens = actualHeadCount
	} else if feedPurchasedTonne > 0 && daysInMonth > 0 {
		// Fallback: Estimate hens: 10,000 hens consume 1 tonne per day
		estimatedHens = (feedPurchasedTonne / float64(daysInMonth)) * 10000.0
	}'''

final_content = final_content.replace(old_estimated, new_estimated)

with open(file_path, 'w') as f:
    f.write(final_content)

print("Fixed head count calculation!")
