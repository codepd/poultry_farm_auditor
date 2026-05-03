/**
 * Date utility functions for timezone-aware date formatting
 */

/**
 * Parse date values safely for date-only strings.
 * Prevents timezone shifts when input is in YYYY-MM-DD format.
 */
export function parseDateValue(date: string | Date): Date {
  if (date instanceof Date) {
    return date;
  }

  if (/^\d{4}-\d{2}-\d{2}$/.test(date)) {
    const [year, month, day] = date.split('-').map(Number);
    return new Date(Date.UTC(year, month - 1, day, 12, 0, 0));
  }

  return new Date(date);
}

/**
 * Format a date string or Date object according to the tenant's timezone and date format
 * @param date - Date string (ISO format) or Date object
 * @param timezone - IANA timezone identifier (e.g., 'Asia/Kolkata')
 * @param dateFormat - Date format string (e.g., 'DD-MM-YYYY', 'MM/DD/YYYY')
 * @returns Formatted date string
 */
export function formatDateForTenant(
  date: string | Date,
  timezone: string = 'Asia/Kolkata',
  dateFormat: string = 'DD-MM-YYYY'
): string {
  try {
    let dateObj: Date;
    
    dateObj = parseDateValue(date);
    
    // Use Intl.DateTimeFormat for timezone-aware formatting
    const formatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });

    const parts = formatter.formatToParts(dateObj);
    const year = parts.find(p => p.type === 'year')?.value || '';
    const month = parts.find(p => p.type === 'month')?.value || '';
    const day = parts.find(p => p.type === 'day')?.value || '';

    // Format according to dateFormat pattern
    return dateFormat
      .replace('YYYY', year)
      .replace('MM', month)
      .replace('DD', day)
      .replace('YY', year.slice(-2));
  } catch (error) {
    console.error('Error formatting date:', error);
    // Fallback to simple formatting
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return dateObj.toLocaleDateString();
  }
}

/**
 * Get current date in tenant's timezone
 * @param timezone - IANA timezone identifier
 * @returns Date object in tenant's timezone
 */
export function getCurrentDateInTimezone(timezone: string = 'Asia/Kolkata'): Date {
  try {
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });

    const parts = formatter.formatToParts(now);
    const year = parseInt(parts.find(p => p.type === 'year')?.value || '0');
    const month = parseInt(parts.find(p => p.type === 'month')?.value || '0') - 1; // Month is 0-indexed
    const day = parseInt(parts.find(p => p.type === 'day')?.value || '0');
    const hour = parseInt(parts.find(p => p.type === 'hour')?.value || '0');
    const minute = parseInt(parts.find(p => p.type === 'minute')?.value || '0');
    const second = parseInt(parts.find(p => p.type === 'second')?.value || '0');

    return new Date(year, month, day, hour, minute, second);
  } catch (error) {
    console.error('Error getting current date in timezone:', error);
    return new Date();
  }
}

/**
 * Convert a date to ISO string in tenant's timezone
 * @param date - Date string or Date object
 * @param timezone - IANA timezone identifier
 * @returns ISO string
 */
export function toISODateInTimezone(
  date: string | Date,
  timezone: string = 'Asia/Kolkata'
): string {
  try {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });

    return formatter.format(dateObj);
  } catch (error) {
    console.error('Error converting date to ISO in timezone:', error);
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return dateObj.toISOString().split('T')[0];
  }
}

/**
 * Format date and time for display
 * @param date - Date string or Date object
 * @param timezone - IANA timezone identifier
 * @param includeTime - Whether to include time in the output
 * @returns Formatted date/time string
 */
export function formatDateTimeForTenant(
  date: string | Date,
  timezone: string = 'Asia/Kolkata',
  includeTime: boolean = false
): string {
  try {
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    const options: Intl.DateTimeFormatOptions = {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    };

    if (includeTime) {
      options.hour = '2-digit';
      options.minute = '2-digit';
    }

    return new Intl.DateTimeFormat('en-US', options).format(dateObj);
  } catch (error) {
    console.error('Error formatting date/time:', error);
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    return includeTime ? dateObj.toLocaleString() : dateObj.toLocaleDateString();
  }
}

/**
 * Calculate age in weeks and days from a date, optionally adding initial age
 * @param dateAdded - Date string (ISO format) or Date object when the batch was added
 * @param timezone - IANA timezone identifier
 * @param initialWeeks - Initial age in weeks when batch was added (default: 0)
 * @param initialDays - Initial age in days when batch was added (default: 0)
 * @returns Object with weeks and days
 */
export function calculateAgeFromDate(
  dateAdded: string | Date,
  timezone: string = 'Asia/Kolkata',
  initialWeeks: number = 0,
  initialDays: number = 0
): { weeks: number; days: number } {
  try {
    // Parse the date added
    let addedDate: Date;
    if (typeof dateAdded === 'string') {
      if (/^\d{4}-\d{2}-\d{2}$/.test(dateAdded)) {
        // For date-only strings, parse as local date to avoid timezone issues
        const [year, month, day] = dateAdded.split('-').map(Number);
        addedDate = new Date(year, month - 1, day);
      } else {
        addedDate = new Date(dateAdded);
      }
    } else {
      addedDate = dateAdded;
    }

    // Get current date in tenant's timezone
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });

    // Get today's date in the tenant's timezone as YYYY-MM-DD
    const todayStr = formatter.format(now);
    const [todayYear, todayMonth, todayDay] = todayStr.split('-').map(Number);
    const todayDate = new Date(todayYear, todayMonth - 1, todayDay);

    // Get the date added in the tenant's timezone
    const addedStr = formatter.format(addedDate);
    const [addedYear, addedMonth, addedDay] = addedStr.split('-').map(Number);
    const addedDateLocal = new Date(addedYear, addedMonth - 1, addedDay);

    // Calculate difference in days
    const diffMs = todayDate.getTime() - addedDateLocal.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    // Calculate weeks and remaining days elapsed since date_added
    const elapsedWeeks = Math.floor(diffDays / 7);
    const elapsedDays = diffDays % 7;

    // Add initial age to elapsed time
    // Convert everything to days first, then back to weeks and days
    const totalDays = (initialWeeks * 7) + initialDays + (elapsedWeeks * 7) + elapsedDays;
    const totalWeeks = Math.floor(totalDays / 7);
    const finalDays = totalDays % 7;

    return { weeks: Math.max(0, totalWeeks), days: Math.max(0, finalDays) };
  } catch (error) {
    console.error('Error calculating age from date:', error);
    return { weeks: initialWeeks, days: initialDays };
  }
}

