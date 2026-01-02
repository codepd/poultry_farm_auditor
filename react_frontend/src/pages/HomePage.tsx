import React, { useState } from 'react';
import MultiCategoryEntry from '../components/Home/MultiCategoryEntry';
import MonthlyStatistics from '../components/Home/MonthlyStatistics';
import YearlyStatistics from '../components/Home/YearlyStatistics';
import MonthlyBarChart from '../components/Home/MonthlyBarChart';
import HenAgeDisplay from '../components/Home/HenAgeDisplay';
import './HomePage.css';

const HomePage: React.FC = () => {
  const [viewMode, setViewMode] = useState<'monthly' | 'yearly'>('monthly');
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

      <HenAgeDisplay />

      <MultiCategoryEntry />

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


