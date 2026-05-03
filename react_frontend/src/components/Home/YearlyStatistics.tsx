import React, { useState, useEffect } from 'react';
import { analyticsAPI, YearlySummary, tenantsAPI } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import './YearlyStatistics.css';

const YearlyStatistics: React.FC = () => {
  const { isAuthenticated, currentTenant } = useAuth();
  const [data, setData] = useState<YearlySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [periodType, setPeriodType] = useState<'calendar' | 'financial'>('calendar');
  const [financialYearStartMonth, setFinancialYearStartMonth] = useState<number>(4);

  useEffect(() => {
    if (!isAuthenticated || !currentTenant?.tenant_id) return;
    const fetchTenant = async () => {
      try {
        const tenant = await tenantsAPI.getTenant(currentTenant.tenant_id);
        if (tenant?.financial_year_start_month && tenant.financial_year_start_month >= 1 && tenant.financial_year_start_month <= 12) {
          setFinancialYearStartMonth(tenant.financial_year_start_month);
        }
      } catch {
        setFinancialYearStartMonth(4);
      }
    };
    fetchTenant();
  }, [isAuthenticated, currentTenant?.tenant_id]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const summaries = await analyticsAPI.getYearlyPeriodSummary(periodType, financialYearStartMonth);
        setData(summaries);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [isAuthenticated, periodType, financialYearStartMonth]);

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

  const currentPeriodLabel = data.length > 0 ? data[0].period_label : '';
  const fyMonthLabel = new Date(2000, financialYearStartMonth - 1, 1).toLocaleString('default', { month: 'long' });

  return (
    <div className="yearly-statistics">
      <h2>Yearly Statistics</h2>
      <div className="view-toggle yearly-view-toggle">
        <button
          className={periodType === 'calendar' ? 'active' : ''}
          onClick={() => setPeriodType('calendar')}
        >
          Calendar Year
        </button>
        <button
          className={periodType === 'financial' ? 'active' : ''}
          onClick={() => setPeriodType('financial')}
        >
          Financial Year
        </button>
      </div>
      {periodType === 'financial' && (
        <div className="yearly-note">
          <strong>FY Config:</strong> Financial year starts in {fyMonthLabel} (month {financialYearStartMonth}).
        </div>
      )}
      {currentPeriodLabel && (
        <div className="yearly-note">
          <strong>Latest Period:</strong> {currentPeriodLabel}
        </div>
      )}
      <div className="yearly-table">
        <table>
          <thead>
            <tr>
              <th>{periodType === 'financial' ? 'Financial Year' : 'Year'}</th>
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
                <tr key={`${item.period_label || item.year}`}>
                  <td>{item.period_label || item.year}</td>
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


