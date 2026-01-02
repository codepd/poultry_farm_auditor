import React from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import './PriceLineChart.css';

interface PriceHistory {
  id: number;
  tenant_id: string;
  price_date: string;
  price_type: string;
  item_name: string;
  price: number;
  created_at: string;
}

interface PriceLineChartProps {
  prices: PriceHistory[];
  priceType: 'EGG' | 'FEED';
  selectedYears?: number[]; // Years to show for comparison
  onPointClick?: (price: PriceHistory) => void;
}

// Helper function to get Monday of the week for a given date
const getMondayOfWeek = (date: Date): Date => {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust when day is Sunday
  return new Date(d.setDate(diff));
};

// Helper function to format week label
const formatWeekLabel = (date: Date): string => {
  const monday = getMondayOfWeek(date);
  return monday.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
};

// Helper function to group feed prices by week (Monday-based) and forward-fill missing weeks
const groupFeedPricesByWeek = (prices: PriceHistory[]): PriceHistory[] => {
  if (prices.length === 0) return [];
  
  // First, group by item and week
  const itemWeekMap = new Map<string, Map<string, PriceHistory>>();
  
  prices.forEach(price => {
    try {
      const priceDate = new Date(price.price_date);
      if (isNaN(priceDate.getTime())) {
        console.warn('Invalid date in price:', price.price_date);
        return;
      }
      const monday = getMondayOfWeek(priceDate);
      const weekKey = monday.toISOString().split('T')[0];
      const itemName = price.item_name;
      
      if (!itemWeekMap.has(itemName)) {
        itemWeekMap.set(itemName, new Map());
      }
      const weekMap = itemWeekMap.get(itemName)!;
      
      // Use the first price found for the week (or update if this date is later)
      if (!weekMap.has(weekKey)) {
        weekMap.set(weekKey, {
          ...price,
          price_date: weekKey, // Use Monday date
        });
      } else {
        // If we already have a price for this week, keep the one with the later original date
        const existing = weekMap.get(weekKey)!;
        const existingDate = new Date(existing.created_at || existing.price_date);
        const newDate = new Date(price.created_at || price.price_date);
        if (newDate > existingDate) {
          weekMap.set(weekKey, {
            ...price,
            price_date: weekKey,
          });
        }
      }
    } catch (e) {
      console.warn('Error processing price:', price, e);
    }
  });
  
  // Now forward-fill missing weeks for each item
  const result: PriceHistory[] = [];
  
  itemWeekMap.forEach((weekMap, itemName) => {
    // Get all weeks sorted by date (these are already Monday dates)
    const weeks = Array.from(weekMap.keys()).sort();
    
    if (weeks.length === 0) return;
    
    // Start with all actual week keys we have data for
    const allWeeksSet = new Set<string>(weeks);
    
    // Generate all weeks from first to last (inclusive) to fill gaps
    const firstWeek = new Date(weeks[0]);
    const lastWeek = new Date(weeks[weeks.length - 1]);
    
    // Extend to the last week of the month/year if we have data in that month
    const lastWeekYear = lastWeek.getFullYear();
    const lastWeekMonth = lastWeek.getMonth();
    
    // Calculate the last Monday of the month
    const lastDayOfMonth = new Date(lastWeekYear, lastWeekMonth + 1, 0); // Last day of the month
    const lastMondayOfMonth = getMondayOfWeek(lastDayOfMonth);
    
    // Use the later of: last week with data OR last Monday of the month
    // Compare dates properly using getTime()
    let endWeek: Date;
    if (lastMondayOfMonth.getTime() > lastWeek.getTime()) {
      endWeek = new Date(lastMondayOfMonth);
    } else {
      endWeek = new Date(lastWeek);
    }
    
    // Debug logging (can be removed later)
    if (process.env.NODE_ENV === 'development') {
      console.log('Week generation for', itemName, {
        lastWeekWithData: lastWeek.toISOString().split('T')[0],
        lastDayOfMonth: lastDayOfMonth.toISOString().split('T')[0],
        lastMondayOfMonth: lastMondayOfMonth.toISOString().split('T')[0],
        endWeek: endWeek.toISOString().split('T')[0],
      });
    }
    
    // Start from the first week (already a Monday)
    let currentWeek = new Date(firstWeek);
    // Normalize to midnight for accurate comparison
    currentWeek.setHours(0, 0, 0, 0);
    endWeek.setHours(0, 0, 0, 0);
    
    // Generate all weeks up to and including the end week
    // Continue until we've passed the end week to ensure it's included
    while (currentWeek.getTime() <= endWeek.getTime()) {
      const weekKey = currentWeek.toISOString().split('T')[0];
      allWeeksSet.add(weekKey);
      
      // Move to next week
      const nextWeek = new Date(currentWeek);
      nextWeek.setDate(nextWeek.getDate() + 7);
      nextWeek.setHours(0, 0, 0, 0);
      currentWeek = nextWeek;
    }
    
    // Ensure the end week is definitely included (safety check)
    const endWeekKey = endWeek.toISOString().split('T')[0];
    allWeeksSet.add(endWeekKey);
    
    // Convert to sorted array
    const allWeeks = Array.from(allWeeksSet).sort();
    
    // Debug logging
    if (process.env.NODE_ENV === 'development') {
      console.log('All weeks generated for', itemName, ':', allWeeks);
      console.log('End week key:', endWeekKey);
      console.log('Is end week in allWeeks?', allWeeks.includes(endWeekKey));
    }
    
    // Forward-fill missing weeks
    let lastPrice: PriceHistory | null = null;
    
    allWeeks.forEach(weekKey => {
      if (weekMap.has(weekKey)) {
        // Week has data, use it and update lastPrice
        lastPrice = weekMap.get(weekKey)!;
        result.push(lastPrice);
      } else if (lastPrice) {
        // Week is missing, use previous week's price (forward-fill)
        result.push({
          ...lastPrice,
          price_date: weekKey,
          id: 0, // Mark as calculated/forward-filled
        });
      }
      // If no lastPrice yet, skip until we find the first week with data
    });
    
    // Debug logging for result
    if (process.env.NODE_ENV === 'development') {
      const resultDates = result.filter(p => p.item_name === itemName).map(p => p.price_date).sort();
      console.log('Result dates for', itemName, ':', resultDates);
      console.log('Last result date:', resultDates[resultDates.length - 1]);
    }
    
    // Ensure we have forward-filled all weeks up to the end week
    // This is a safety check to make sure the last week of the month is included
    if (allWeeks.length > 0 && lastPrice !== null) {
      const lastWeekKey = allWeeks[allWeeks.length - 1];
      const hasLastWeek = result.some(p => 
        p.item_name === itemName && p.price_date === lastWeekKey
      );
      if (!hasLastWeek && lastPrice !== null) {
        // Add forward-filled price for the last week
        // TypeScript knows lastPrice is not null here due to the outer check
        const priceToUse = lastPrice as PriceHistory; // Type assertion for TypeScript
        const forwardFilled: PriceHistory = {
          id: 0, // Mark as calculated/forward-filled
          tenant_id: priceToUse.tenant_id,
          price_date: lastWeekKey,
          price_type: priceToUse.price_type,
          item_name: priceToUse.item_name,
          price: priceToUse.price,
          created_at: priceToUse.created_at,
        };
        result.push(forwardFilled);
      }
    }
    
    // Final check: ensure the end week (last Monday of month) is included
    if (allWeeks.length > 0 && lastPrice !== null) {
      const endWeekKey = endWeek.toISOString().split('T')[0];
      const hasEndWeek = result.some(p => 
        p.item_name === itemName && p.price_date === endWeekKey
      );
      if (!hasEndWeek && lastPrice !== null) {
        const priceToUse = lastPrice as PriceHistory;
        result.push({
          id: 0,
          tenant_id: priceToUse.tenant_id,
          price_date: endWeekKey,
          price_type: priceToUse.price_type,
          item_name: priceToUse.item_name,
          price: priceToUse.price,
          created_at: priceToUse.created_at,
        });
      }
    }
    
    // Ensure the last week with actual data is always included (safety check)
    if (weeks.length > 0) {
      const lastWeekKey = weeks[weeks.length - 1];
      if (weekMap.has(lastWeekKey)) {
        const lastWeekPrice = weekMap.get(lastWeekKey)!;
        // Check if it's already in result
        const alreadyIncluded = result.some(p => 
          p.item_name === itemName && p.price_date === lastWeekKey
        );
        if (!alreadyIncluded) {
          result.push(lastWeekPrice);
        }
      }
    }
  });
  
  return result;
};

const PriceLineChart: React.FC<PriceLineChartProps> = ({ 
  prices, 
  priceType, 
  selectedYears,
  onPointClick 
}) => {
  // Filter prices by type
  let filteredPrices = prices.filter(p => p.price_type === priceType);

  // For FEED prices, group by week (Monday-based)
  if (priceType === 'FEED' && filteredPrices.length > 0) {
    filteredPrices = groupFeedPricesByWeek(filteredPrices);
  }

  if (filteredPrices.length === 0) {
    return (
      <div className="chart-no-data">
        No {priceType === 'EGG' ? 'egg' : 'feed'} price data available
      </div>
    );
  }

  // If selectedYears is provided, filter by those years and enable year-on-year comparison
  const isYearOnYear = selectedYears && selectedYears.length > 0;
  
  let processedPrices = filteredPrices;
  if (isYearOnYear) {
    processedPrices = filteredPrices.filter(p => {
      try {
        const priceDate = new Date(p.price_date);
        // Handle both date strings and Date objects
        if (isNaN(priceDate.getTime())) {
          return false; // Invalid date, exclude
        }
        const year = priceDate.getFullYear();
        return selectedYears.includes(year);
      } catch (e) {
        console.warn('Error parsing date:', p.price_date, e);
        return false;
      }
    });
  }

  // Get all available years from the data
  const availableYears = Array.from(
    new Set(processedPrices.map(p => new Date(p.price_date).getFullYear()))
  ).sort((a, b) => b - a); // Most recent first

  // Color palette for different items
  const itemColors = [
    '#667eea', // Purple
    '#f093fb', // Pink
    '#4facfe', // Blue
    '#43e97b', // Green
    '#fa709a', // Rose
    '#fee140', // Yellow
    '#30cfd0', // Cyan
    '#a8edea', // Light Cyan
    '#ff9a9e', // Light Pink
    '#fecfef', // Light Purple
  ];

  // Year line styles for year-on-year comparison
  const yearLineStyles = ['solid', 'dashed', 'dotted', '5 5', '10 5'];

  let chartData: any[] = [];
  let linesToRender: any[] = [];

  if (isYearOnYear && selectedYears && selectedYears.length > 0) {
    // Year-on-year comparison mode
    // Group by item name, then by year, then by month (for EGG) or week (for FEED)
    const itemYearMonthMap = new Map<string, Map<number, Map<number, PriceHistory>>>();
    
    // For FEED, we need to process prices by year first to forward-fill within each year
    if (priceType === 'FEED') {
      // Group prices by year and item, then forward-fill within each year
      const yearItemPrices = new Map<number, Map<string, PriceHistory[]>>();
      
      processedPrices.forEach(price => {
        const year = new Date(price.price_date).getFullYear();
        if (!yearItemPrices.has(year)) {
          yearItemPrices.set(year, new Map());
        }
        const itemMap = yearItemPrices.get(year)!;
        if (!itemMap.has(price.item_name)) {
          itemMap.set(price.item_name, []);
        }
        itemMap.get(price.item_name)!.push(price);
      });
      
      // Forward-fill missing weeks for each year-item combination
      yearItemPrices.forEach((itemMap, year) => {
        itemMap.forEach((prices, itemName) => {
          const filledPrices = groupFeedPricesByWeek(prices);
          filledPrices.forEach(price => {
            const date = new Date(price.price_date);
            const startOfYear = new Date(date.getFullYear(), 0, 1);
            const days = Math.floor((date.getTime() - startOfYear.getTime()) / (24 * 60 * 60 * 1000));
            const period = Math.ceil((days + startOfYear.getDay() + 1) / 7);
            
            if (!itemYearMonthMap.has(itemName)) {
              itemYearMonthMap.set(itemName, new Map());
            }
            const yearMap = itemYearMonthMap.get(itemName)!;
            if (!yearMap.has(year)) {
              yearMap.set(year, new Map());
            }
            const periodMap = yearMap.get(year)!;
            periodMap.set(period, price);
          });
        });
      });
    } else {
      // For EGG, use original logic
      processedPrices.forEach(price => {
        const year = new Date(price.price_date).getFullYear();
        const period = new Date(price.price_date).getMonth() + 1; // 1-12
        
        if (!itemYearMonthMap.has(price.item_name)) {
          itemYearMonthMap.set(price.item_name, new Map());
        }
        const yearMap = itemYearMonthMap.get(price.item_name)!;
        
        if (!yearMap.has(year)) {
          yearMap.set(year, new Map());
        }
        const periodMap = yearMap.get(year)!;
        
        // For EGG, prefer month-start dates (ending in -01)
        if (price.price_date.endsWith('-01') || !periodMap.has(period)) {
          periodMap.set(period, price);
        }
      });
    }

    // Get all periods (months for EGG, weeks for FEED)
    const allPeriods = priceType === 'EGG' 
      ? Array.from({ length: 12 }, (_, i) => i + 1) // 1-12 months
      : Array.from({ length: 52 }, (_, i) => i + 1); // 1-52 weeks
    
    // Build chart data - one entry per period
    chartData = allPeriods.map(period => {
      let periodLabel: string;
      if (priceType === 'EGG') {
        periodLabel = new Date(2024, period - 1, 1).toLocaleDateString('en-IN', { month: 'short' });
      } else {
        // For feed, calculate the Monday date for this week
        const year = selectedYears[0] || new Date().getFullYear();
        const startOfYear = new Date(year, 0, 1);
        const daysToAdd = (period - 1) * 7 - (startOfYear.getDay() === 0 ? 6 : startOfYear.getDay() - 1);
        const weekMonday = new Date(startOfYear);
        weekMonday.setDate(startOfYear.getDate() + daysToAdd);
        periodLabel = formatWeekLabel(weekMonday);
      }
      
      const dataPoint: any = {
        period,
        periodLabel,
      };

      // For each item and year, add the price for this period
      itemYearMonthMap.forEach((yearMap, itemName) => {
        selectedYears.forEach(year => {
          const periodMap = yearMap.get(year);
          if (periodMap && periodMap.has(period)) {
            const priceForPeriod = periodMap.get(period)!;
            const dataKey = `${itemName} (${year})`;
            dataPoint[dataKey] = priceForPeriod.price;
            dataPoint[`${dataKey}_id`] = priceForPeriod.id;
            dataPoint[`${dataKey}_data`] = priceForPeriod;
          }
        });
      });

      return dataPoint;
    });

    // Create lines for each item-year combination
    const itemNames = Array.from(itemYearMonthMap.keys()).sort((a, b) => {
      if (priceType === 'EGG') {
        const order = ['LARGE EGG', 'MEDIUM EGG', 'SMALL EGG'];
        const aIndex = order.findIndex(o => a.includes(o));
        const bIndex = order.findIndex(o => b.includes(o));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
      } else {
        const order = ['LAYER MASH', 'PRE-LAYER MASH', 'PRE LAYER MASH', 'PLM', 'GROWER MASH', 'CHICK MASH'];
        const aIndex = order.findIndex(o => a.toUpperCase().includes(o.toUpperCase()));
        const bIndex = order.findIndex(o => b.toUpperCase().includes(o.toUpperCase()));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
      }
      return a.localeCompare(b);
    });

    itemNames.forEach((itemName, itemIndex) => {
      selectedYears.forEach((year, yearIndex) => {
        const yearMap = itemYearMonthMap.get(itemName);
        if (yearMap && yearMap.has(year)) {
          const dataKey = `${itemName} (${year})`;
          const color = itemColors[itemIndex % itemColors.length];
          const strokeDasharray = yearLineStyles[yearIndex % yearLineStyles.length];
          
          linesToRender.push({
            key: dataKey,
            dataKey,
            name: dataKey,
            color,
            strokeDasharray,
            itemName,
            year,
          });
        }
      });
    });
  } else {
    // Single year or all data mode
    let finalPrices = processedPrices;
    
    // For FEED, forward-fill missing weeks
    if (priceType === 'FEED') {
      finalPrices = groupFeedPricesByWeek(processedPrices);
    }
    
    const itemsMap = new Map<string, PriceHistory[]>();
    finalPrices.forEach(price => {
      if (!itemsMap.has(price.item_name)) {
        itemsMap.set(price.item_name, []);
      }
      itemsMap.get(price.item_name)!.push(price);
    });

    itemsMap.forEach((prices, itemName) => {
      prices.sort((a, b) => new Date(a.price_date).getTime() - new Date(b.price_date).getTime());
    });

    const allDates = Array.from(new Set(finalPrices.map(p => p.price_date)))
      .sort((a, b) => new Date(a).getTime() - new Date(b).getTime());

    // Debug logging
    if (process.env.NODE_ENV === 'development' && priceType === 'FEED') {
      console.log('Chart rendering - allDates:', allDates);
      console.log('Chart rendering - last date in allDates:', allDates[allDates.length - 1]);
      console.log('Chart rendering - finalPrices count:', finalPrices.length);
      const layerMashDates = finalPrices.filter(p => p.item_name === 'LAYER MASH').map(p => p.price_date).sort();
      console.log('Chart rendering - LAYER MASH dates in finalPrices:', layerMashDates);
      console.log('Chart rendering - last LAYER MASH date:', layerMashDates[layerMashDates.length - 1]);
    }

    chartData = allDates.map(date => {
      const dateObj = new Date(date);
      let dateLabel: string;
      
      if (priceType === 'EGG') {
        dateLabel = date.endsWith('-01')
          ? dateObj.toLocaleDateString('en-IN', { month: 'short', year: 'numeric' })
          : dateObj.toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' });
      } else {
        // For feed, show week label (Monday date)
        dateLabel = formatWeekLabel(dateObj);
      }
      
      const dataPoint: any = {
        date,
        dateLabel,
      };

      itemsMap.forEach((prices, itemName) => {
        const priceForDate = prices.find(p => p.price_date === date);
        if (priceForDate) {
          dataPoint[itemName] = priceForDate.price;
          dataPoint[`${itemName}_id`] = priceForDate.id;
          dataPoint[`${itemName}_data`] = priceForDate;
        }
      });

      return dataPoint;
    });

    // Debug logging for chartData
    if (process.env.NODE_ENV === 'development' && priceType === 'FEED') {
      console.log('Chart rendering - chartData length:', chartData.length);
      console.log('Chart rendering - last chartData point:', chartData[chartData.length - 1]);
      const hasDec29 = chartData.some(d => d.date === '2025-12-29' || d.dateLabel?.includes('Dec 29'));
      console.log('Chart rendering - has Dec 29 in chartData?', hasDec29);
      
      // Check if last point has any price values
      const lastPoint = chartData[chartData.length - 1];
      if (lastPoint) {
        const priceKeys = Object.keys(lastPoint).filter(k => 
          !['date', 'dateLabel', 'period', 'periodLabel'].includes(k) && 
          !k.endsWith('_id') && 
          !k.endsWith('_data')
        );
        console.log('Chart rendering - last point price keys:', priceKeys);
        console.log('Chart rendering - last point price values:', 
          priceKeys.reduce((acc, k) => ({ ...acc, [k]: lastPoint[k] }), {})
        );
      }
    }

    // Get item names sorted
    const itemNames = Array.from(itemsMap.keys()).sort((a, b) => {
      if (priceType === 'EGG') {
        const order = ['LARGE EGG', 'MEDIUM EGG', 'SMALL EGG'];
        const aIndex = order.findIndex(o => a.includes(o));
        const bIndex = order.findIndex(o => b.includes(o));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
      } else {
        const order = ['LAYER MASH', 'PRE-LAYER MASH', 'PRE LAYER MASH', 'PLM', 'GROWER MASH', 'CHICK MASH'];
        const aIndex = order.findIndex(o => a.toUpperCase().includes(o.toUpperCase()));
        const bIndex = order.findIndex(o => b.toUpperCase().includes(o.toUpperCase()));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
      }
      return a.localeCompare(b);
    });

    itemNames.forEach((itemName, index) => {
      linesToRender.push({
        key: itemName,
        dataKey: itemName,
        name: itemName,
        color: itemColors[index % itemColors.length],
        strokeDasharray: '0',
        itemName,
      });
    });
  }

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="price-chart-tooltip">
          <div className="tooltip-header">{label}</div>
          {payload.map((entry: any, index: number) => (
            <div key={index} className="tooltip-item" style={{ color: entry.color }}>
              <span className="tooltip-label">{entry.dataKey}:</span>
              <span className="tooltip-value">₹{entry.value?.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</span>
            </div>
          ))}
        </div>
      );
    }
    return null;
  };

  // Format Y-axis (prices)
  const formatYAxis = (value: number) => {
    return `₹${value.toFixed(2)}`;
  };

  // Calculate which tick indices to show to prevent overlapping while ensuring first and last are always shown
  const calculateXAxisTickIndices = (dataLength: number): number[] => {
    if (dataLength <= 12) {
      // Show all ticks
      return Array.from({ length: dataLength }, (_, i) => i);
    }
    
    // Calculate desired number of ticks
    let desiredTicks = 12;
    if (dataLength <= 24) desiredTicks = dataLength;
    else if (dataLength <= 52) desiredTicks = 12;
    else desiredTicks = 20;
    
    // Always include first and last
    const tickIndices: number[] = [0];
    
    // Calculate step to get approximately desiredTicks
    const step = Math.max(1, Math.floor((dataLength - 1) / (desiredTicks - 1)));
    
    // Add intermediate ticks
    for (let i = step; i < dataLength - 1; i += step) {
      tickIndices.push(i);
    }
    
    // Always include last tick if not already included
    if (tickIndices[tickIndices.length - 1] !== dataLength - 1) {
      tickIndices.push(dataLength - 1);
    }
    
    return tickIndices;
  };

  // Get tick indices to show
  const xAxisTickIndices = calculateXAxisTickIndices(chartData.length);
  
  // Get labels that should be shown
  const allLabels = chartData.map(d => isYearOnYear ? d.periodLabel : d.dateLabel);
  const labelsToShow = new Set(xAxisTickIndices.map(i => allLabels[i]));
  
  // Custom tick formatter that only shows labels for specified values
  const customTickFormatter = (value: any): string => {
    // Always show if it's in our set of labels to show
    if (labelsToShow.has(value)) {
      return value;
    }
    return ''; // Return empty string to hide this tick
  };
  
  // Debug logging for FEED prices
  if (process.env.NODE_ENV === 'development' && priceType === 'FEED' && !isYearOnYear) {
    console.log('XAxis ticks calculation:', {
      totalDataPoints: chartData.length,
      totalLabels: allLabels.length,
      lastLabel: allLabels[allLabels.length - 1],
      lastDataPoint: chartData[chartData.length - 1],
      tickIndices: xAxisTickIndices,
      lastTickIndex: xAxisTickIndices[xAxisTickIndices.length - 1],
      hasLastIndex: xAxisTickIndices.includes(chartData.length - 1),
      lastDateInData: chartData[chartData.length - 1]?.date,
      lastDateLabelInData: chartData[chartData.length - 1]?.dateLabel,
      labelsToShow: Array.from(labelsToShow),
      lastLabelInSet: labelsToShow.has(allLabels[allLabels.length - 1]),
    });
  }

  return (
    <div className="price-line-chart">
      <ResponsiveContainer width="100%" height={400}>
        <LineChart
          data={chartData}
          margin={{ top: 20, right: 30, left: 20, bottom: 80 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
          <XAxis
            dataKey={isYearOnYear ? "periodLabel" : "dateLabel"}
            angle={-45}
            textAnchor="end"
            height={100}
            tick={{ fontSize: 10, fill: '#666' }}
            interval={0}
            tickFormatter={customTickFormatter}
            domain={['dataMin', 'dataMax']}
          />
          <YAxis
            tick={{ fontSize: 12, fill: '#666' }}
            tickFormatter={formatYAxis}
            label={{ value: 'Price (₹)', angle: -90, position: 'insideLeft', style: { fill: '#666' } }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="line"
          />
          {linesToRender.map((line) => (
            <Line
              key={line.key}
              type="monotone"
              dataKey={line.dataKey}
              name={line.name}
              stroke={line.color}
              strokeWidth={2}
              strokeDasharray={line.strokeDasharray}
              dot={{ r: 4, fill: line.color }}
              activeDot={{ r: 6, onClick: (e: any, payload: any) => {
                const priceData = payload.payload[`${line.dataKey}_data`];
                if (priceData && onPointClick) {
                  onPointClick(priceData);
                }
              }}}
              connectNulls={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
};

export default PriceLineChart;
