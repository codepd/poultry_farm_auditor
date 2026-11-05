import React, { useState } from 'react';
import MultiCategoryEntry from '../components/Home/MultiCategoryEntry';
import MonthlyStatistics from '../components/Home/MonthlyStatistics';
import YearlyStatistics from '../components/Home/YearlyStatistics';
import MonthlyBarChart from '../components/Home/MonthlyBarChart';
import DateRangeSummaryPanel from '../components/Home/DateRangeSummary';
import { useAuth } from '../context/AuthContext';
import './HomePage.css';

const HomePage: React.FC = () => {
  const { currentTenant } = useAuth();
  const [viewMode, setViewMode] = useState<'monthly' | 'yearly'>('monthly');
  const [showQuickEntry, setShowQuickEntry] = useState(false);
  const [showDateRange, setShowDateRange] = useState(false);

  const canEdit = Boolean(currentTenant && ['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(currentTenant.role));

  // Default to last month
  const lastMonth = new Date();
  lastMonth.setMonth(lastMonth.getMonth() - 1);
  const [selectedYear, setSelectedYear] = useState(lastMonth.getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(lastMonth.getMonth() + 1);

  const handleMonthClick = (year: number, month: number) => {
    setSelectedYear(year);
    setSelectedMonth(month);
    setViewMode('monthly');
  };

  return (
    <div className="home-page">
      {/* Action shortcuts bar */}
      <div className="home-shortcuts-bar">
        {canEdit && (
          <button
            type="button"
            className={`shortcut-pill ${showQuickEntry ? 'active' : ''}`}
            onClick={() => setShowQuickEntry(!showQuickEntry)}
          >
            {showQuickEntry ? '✕ Close Quick Entry' : '➕ Quick Data Entry'}
          </button>
        )}

        <button
          type="button"
          className={`shortcut-pill ${showDateRange ? 'active' : ''}`}
          onClick={() => setShowDateRange(!showDateRange)}
        >
          {showDateRange ? '✕ Hide Date Range' : '📅 Custom Date Range Filter'}
        </button>
      </div>

      {canEdit && showQuickEntry && (
        <div className="collapsible-container">
          <MultiCategoryEntry />
        </div>
      )}

      {showDateRange && (
        <div className="collapsible-container">
          <DateRangeSummaryPanel />
        </div>
      )}

      <div className="statistics-section">
        {viewMode === 'monthly' && <MonthlyBarChart onMonthClick={handleMonthClick} />}
        
        <div className="view-controls">
          <div className="view-toggle">
            <button
              className={viewMode === 'monthly' ? 'active' : ''}
              onClick={() => setViewMode('monthly')}
            >
              Monthly
            </button>
            <button
              className={viewMode === 'yearly' ? 'active' : ''}
              onClick={() => setViewMode('yearly')}
            >
              Yearly
            </button>
          </div>

          {viewMode === 'monthly' && (
            <div className="date-selector">
              <select
                value={selectedYear}
                onChange={(e) => setSelectedYear(parseInt(e.target.value))}
                title="Select year"
                aria-label="Select year"
              >
                {Array.from({ length: 5 }, (_, i) => new Date().getFullYear() - i).map((year) => (
                  <option key={year} value={year}>
                    {year}
                  </option>
                ))}
              </select>
              <select
                value={selectedMonth}
                onChange={(e) => setSelectedMonth(parseInt(e.target.value))}
                title="Select month"
                aria-label="Select month"
              >
                {Array.from({ length: 12 }, (_, i) => i + 1).map((month) => (
                  <option key={month} value={month}>
                    {new Date(2000, month - 1).toLocaleString('default', { month: 'long' })}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        {viewMode === 'monthly' ? (
          <MonthlyStatistics year={selectedYear} month={selectedMonth} />
        ) : (
          <YearlyStatistics />
        )}
      </div>
    </div>
  );
};

export default HomePage;


