import React, { useState, useEffect } from 'react';
import { analyticsAPI, YearlySummary } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import './YearlyStatistics.css';

const YearlyStatistics: React.FC = () => {
  const { isAuthenticated } = useAuth();
  const [data, setData] = useState<YearlySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const summaries = await analyticsAPI.getAllYearsSummary();
        setData(summaries);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [isAuthenticated]);

  const formatCurrency = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '₹0.00';
    return `₹${num.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  if (loading) {
    return <div className="loading">Loading statistics...</div>;
  }

  if (error) {
    return <div className="error-message">{error}</div>;
  }

  const currentYear = new Date().getFullYear();
  const currentMonth = new Date().getMonth() + 1;
  const currentYearData = data.find(d => d.year === currentYear);

  return (
    <div className="yearly-statistics">
      <h2>Yearly Statistics</h2>
      {currentYearData && (
        <div className="yearly-note">
          <strong>Note:</strong> {currentYear} data shows up to {new Date(2000, currentMonth - 1).toLocaleString('default', { month: 'long' })} only.
        </div>
      )}
      <div className="yearly-table">
        <table>
          <thead>
            <tr>
              <th>Year</th>
              <th>Total Sales</th>
              <th>Total Expense</th>
              <th>Net Profit</th>
            </tr>
          </thead>
          <tbody>
            {data.length === 0 ? (
              <tr>
                <td colSpan={4} className="no-data">No data available</td>
              </tr>
            ) : (
              data.map((item) => (
                <tr key={item.year}>
                  <td>{item.year}</td>
                  <td>{formatCurrency(item.total_sales)}</td>
                  <td>{formatCurrency(item.total_expense)}</td>
                  <td className={(item.net_profit || 0) >= 0 ? 'positive' : 'negative'}>
                    {formatCurrency(item.net_profit)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default YearlyStatistics;


