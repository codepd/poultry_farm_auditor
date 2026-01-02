import React, { useState, useEffect } from 'react';
import { analyticsAPI, EnhancedMonthlySummary } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import BreakdownModal from './BreakdownModal';
import PaymentModal from './PaymentModal';
import './MonthlyStatistics.css';

interface MonthlyStatisticsProps {
  year: number;
  month: number;
}

const MonthlyStatistics: React.FC<MonthlyStatisticsProps> = ({ year, month }) => {
  const { isAuthenticated, currentTenant } = useAuth();
  const [data, setData] = useState<EnhancedMonthlySummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [breakdownCategory, setBreakdownCategory] = useState<string | null>(null);
  const [showPaymentModal, setShowPaymentModal] = useState(false);

  // Check if user can edit (OWNER or Manager role)
  const canEdit = currentTenant && ['OWNER', 'MANAGER'].includes(currentTenant.role);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const summary = await analyticsAPI.getEnhancedMonthlySummary(year, month);
        setData(summary);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [year, month, isAuthenticated]);

  if (loading) {
    return <div className="loading">Loading statistics...</div>;
  }

  if (error) {
    return <div className="error-message">{error}</div>;
  }

  if (!data) {
    return <div className="no-data">No data available for this month</div>;
  }

  const formatNumber = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '0';
    return num.toLocaleString('en-IN', { maximumFractionDigits: 2 });
  };

  const formatCurrency = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '₹0.00';
    return `₹${num.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  const handleShowBreakdown = (category: string) => {
    setBreakdownCategory(category);
  };

  const handleCloseBreakdown = () => {
    setBreakdownCategory(null);
  };

  const handleShowPayments = () => {
    setShowPaymentModal(true);
  };

  const handleClosePaymentModal = () => {
    setShowPaymentModal(false);
  };

  const handleRefresh = () => {
    setLoading(true);
    analyticsAPI.getEnhancedMonthlySummary(year, month)
      .then(setData)
      .catch((err: any) => setError(err.response?.data?.error || 'Failed to load statistics'))
      .finally(() => setLoading(false));
  };

  return (
    <div className="monthly-statistics">
      <div className="monthly-statistics-header">
        <h2>Monthly Statistics - {new Date(year, month - 1).toLocaleString('default', { month: 'long', year: 'numeric' })}</h2>
        {canEdit && (
          <button onClick={handleRefresh} className="refresh-btn">Refresh</button>
        )}
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <h3>Total Eggs Sold</h3>
          <div
            className="stat-value clickable"
            onClick={() => handleShowBreakdown('EGG')}
            title="Click to view breakdown"
          >
            {formatNumber(data.total_eggs_sold)} Nos
          </div>
          {data.egg_breakdown && data.egg_breakdown.length > 0 && (
            <div className="breakdown">
              <div className="breakdown-header">Breakdown by Type:</div>
              {data.egg_breakdown.map((item, idx) => (
                <div key={idx} className="breakdown-item">
                  <span className="breakdown-name">{item.type}:</span>
                  <span className="breakdown-value">{formatNumber(item.quantity)} Nos</span>
                  {item.amount && (
                    <span className="breakdown-amount">({formatCurrency(item.amount)})</span>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="stat-card">
          <h3>Total Egg Price</h3>
          <div className="stat-value">{formatCurrency(data.total_egg_price)}</div>
        </div>

        <div className="stat-card">
          <h3>Feed Purchased</h3>
          <div
            className="stat-value clickable"
            onClick={() => handleShowBreakdown('FEED')}
            title="Click to view breakdown"
          >
            {formatNumber(data.feed_purchased_tonne)} Tonnes
          </div>
          {data.feed_breakdown && data.feed_breakdown.length > 0 && (
            <div className="breakdown">
              <div className="breakdown-header">Breakdown by Type:</div>
              {data.feed_breakdown.map((item, idx) => (
                <div key={idx} className="breakdown-item">
                  <span className="breakdown-name">{item.type}:</span>
                  <span className="breakdown-value">{formatNumber(item.quantity)} T</span>
                  {item.amount && (
                    <span className="breakdown-amount">({formatCurrency(item.amount)})</span>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="stat-card">
          <h3>Total Feed Price</h3>
          <div className="stat-value">{formatCurrency(data.total_feed_price)}</div>
        </div>

        <div className="stat-card">
          <h3>Medicine Expenses</h3>
          <div
            className="stat-value clickable"
            onClick={() => handleShowBreakdown('MEDICINE')}
            title="Click to view breakdown"
          >
            {formatCurrency(data.total_medicines)}
          </div>
          {data.medicine_breakdown && data.medicine_breakdown.length > 0 && (
            <div className="breakdown">
              <div className="breakdown-header">Breakdown by Type:</div>
              {data.medicine_breakdown.map((item, idx) => (
                <div key={idx} className="breakdown-item">
                  <span className="breakdown-name">{item.type}:</span>
                  <span className="breakdown-value">{formatNumber(item.quantity)}</span>
                  <span className="breakdown-amount">({formatCurrency(item.amount)})</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="stat-card">
          <h3>Other Expenses</h3>
          <div
            className="stat-value clickable"
            onClick={() => handleShowBreakdown('OTHER')}
            title="Click to view breakdown"
          >
            {formatCurrency(data.other_expenses || 0)}
          </div>
        </div>

        <div className="stat-card">
          <h3>Payments Received</h3>
          <div
            className="stat-value clickable"
            onClick={handleShowPayments}
            title="Click to view payments"
          >
            {formatCurrency(data.total_payments || 0)}
          </div>
          {data.payment_breakdown && data.payment_breakdown.length > 0 && (
            <div className="breakdown">
              <div className="breakdown-header">Breakdown:</div>
              {data.payment_breakdown.map((item, idx) => (
                <div key={idx} className="breakdown-item">
                  <span className="breakdown-name">{item.type}:</span>
                  <span className="breakdown-amount">{formatCurrency(item.amount)}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="stat-card">
          <h3>Estimated Hens</h3>
          <div className="stat-value">{formatNumber(data.estimated_hens)}</div>
        </div>

        <div className="stat-card">
          <h3>Egg Percentage</h3>
          <div className="stat-value">{formatNumber(data.egg_percentage)}%</div>
        </div>

        <div className="stat-card net-profit">
          <h3>Net Profit</h3>
          <div className={`stat-value ${(data.net_profit || 0) >= 0 ? 'positive' : 'negative'}`}>
            {formatCurrency(data.net_profit)}
          </div>
        </div>
      </div>

      {breakdownCategory && (
        <BreakdownModal
          isOpen={true}
          onClose={handleCloseBreakdown}
          category={breakdownCategory}
          year={year}
          month={month}
          canEdit={canEdit || false}
          onUpdate={handleRefresh}
        />
      )}

      {showPaymentModal && (
        <PaymentModal
          isOpen={true}
          onClose={handleClosePaymentModal}
          year={year}
          month={month}
          canEdit={canEdit || false}
          onUpdate={handleRefresh}
        />
      )}
    </div>
  );
};

export default MonthlyStatistics;


