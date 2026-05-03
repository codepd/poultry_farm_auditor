import React, { useEffect, useState } from 'react';
import { analyticsAPI, DateRangeSummary } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import './DateRangeSummary.css';

const toIsoDate = (d: Date) => d.toISOString().split('T')[0];

const DateRangeSummaryPanel: React.FC = () => {
  const { isAuthenticated } = useAuth();
  const today = new Date();
  const monthStart = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState<string>(toIsoDate(monthStart));
  const [endDate, setEndDate] = useState<string>(toIsoDate(today));
  const [summary, setSummary] = useState<DateRangeSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const fetchSummary = async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    setError('');
    try {
      const data = await analyticsAPI.getDateRangeSummary(startDate, endDate);
      setSummary(data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load date range summary');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!isAuthenticated) return;
    fetchSummary();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated]);

  const formatCurrency = (num: number | undefined) =>
    `₹${(num || 0).toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;

  return (
    <div className="date-range-summary-panel">
      <div className="date-range-header">
        <h2>Date Range Summary</h2>
        <div className="date-range-controls">
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            aria-label="Start date"
          />
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            aria-label="End date"
          />
          <button onClick={fetchSummary} disabled={loading || !startDate || !endDate}>
            {loading ? 'Loading...' : 'Apply'}
          </button>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      {summary && (
        <div className="date-range-grid">
          <div className="date-range-card">
            <div className="label">Total Sales</div>
            <div className="value">{formatCurrency(summary.total_sales)}</div>
          </div>
          <div className="date-range-card">
            <div className="label">Total Expense</div>
            <div className="value">{formatCurrency(summary.total_expense)}</div>
          </div>
          <div className="date-range-card">
            <div className={`value ${summary.net_profit >= 0 ? 'positive' : 'negative'}`}>
              {formatCurrency(summary.net_profit)}
            </div>
            <div className="label">Net Profit</div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DateRangeSummaryPanel;
