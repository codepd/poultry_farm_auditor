import React, { useState, useEffect, useMemo } from 'react';
import { analyticsAPI, EnhancedMonthlySummary } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import BreakdownModal from './BreakdownModal';
import PaymentModal from './PaymentModal';
import './MonthlyStatistics.css';

interface MonthlyStatisticsProps {
  year: number;
  month: number;
}

type ProfitPreset = 'comprehensive' | 'ledger' | 'flock' | 'custom';

const MonthlyStatistics: React.FC<MonthlyStatisticsProps> = ({ year, month }) => {
  const { isAuthenticated, currentTenant } = useAuth();
  const [data, setData] = useState<EnhancedMonthlySummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [breakdownCategory, setBreakdownCategory] = useState<string | null>(null);
  const [showPaymentModal, setShowPaymentModal] = useState(false);

  // Profit Calculation Toggles
  const [includeHenSales, setIncludeHenSales] = useState(true);
  const [includeRearingFeed, setIncludeRearingFeed] = useState(true);
  const [includeOverheads, setIncludeOverheads] = useState(true);
  const [includeDiscounts, setIncludeDiscounts] = useState(true);
  const [includeTds, setIncludeTds] = useState(true);
  const [activePreset, setActivePreset] = useState<ProfitPreset>('comprehensive');

  // Check if user can edit (OWNER, CO_OWNER, ADMIN, or MANAGER role)
  const canEdit = Boolean(currentTenant && ['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(currentTenant.role));

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

  const formatNumber = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '0';
    return num.toLocaleString('en-IN', { maximumFractionDigits: 2 });
  };

  const formatCurrency = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '₹0.00';
    return `₹${num.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  const formatLakhs = (num: number | undefined | null) => {
    if (num === undefined || num === null) return '₹0';
    const abs = Math.abs(num);
    const sign = num < 0 ? '-' : '';
    if (abs >= 100000) {
      return `${sign}₹${(abs / 100000).toFixed(2)}L`;
    }
    if (abs >= 1000) {
      return `${sign}₹${(abs / 1000).toFixed(1)}k`;
    }
    return `${sign}₹${abs.toFixed(0)}`;
  };

  // Preset switchers
  const applyPreset = (preset: ProfitPreset) => {
    setActivePreset(preset);
    if (preset === 'flock') {
      // Flock Laying Operational Profit: Eggs + Discounts - Layer Feed - Meds
      setIncludeHenSales(false);
      setIncludeRearingFeed(false);
      setIncludeOverheads(false);
      setIncludeDiscounts(true);
      setIncludeTds(false);
    } else if (preset === 'ledger') {
      // Statement Ledger Profit: Eggs + Discounts - All Feeds - Meds - TDS
      setIncludeHenSales(false);
      setIncludeRearingFeed(true);
      setIncludeOverheads(false);
      setIncludeDiscounts(true);
      setIncludeTds(true);
    } else if (preset === 'comprehensive') {
      // Farm Comprehensive Profit: All Incomes - All Costs & Overheads
      setIncludeHenSales(true);
      setIncludeRearingFeed(true);
      setIncludeOverheads(true);
      setIncludeDiscounts(true);
      setIncludeTds(true);
    }
  };

  const handleToggle = (setter: React.Dispatch<React.SetStateAction<boolean>>, currentVal: boolean) => {
    setter(!currentVal);
    setActivePreset('custom');
  };

  // Dynamic calculations
  const calcDetails = useMemo(() => {
    if (!data) return null;

    const eggRev = data.total_egg_price || 0;
    const discounts = data.total_discounts || 0;
    const henSales = data.hen_sale_income || data.other_income || 0;

    const totalFeedCost = data.total_feed_price || 0;
    const rearingFeedCost = data.rearing_feed_price ?? 0;
    const layerFeedCost = data.layer_feed_price ?? Math.max(0, totalFeedCost - rearingFeedCost);

    const meds = data.total_medicines || 0;
    const overheads = data.other_expenses || 0;
    const labor = data.labor_expenses || 0;
    const tds = data.total_tds || 0;

    // Dynamic Net Profit
    const profit =
      eggRev +
      (includeDiscounts ? discounts : 0) +
      (includeHenSales ? henSales : 0) -
      layerFeedCost -
      (includeRearingFeed ? rearingFeedCost : 0) -
      meds -
      (includeOverheads ? overheads : 0) -
      (includeTds ? tds : 0);

    const totalInflow = eggRev + (includeDiscounts ? discounts : 0) + (includeHenSales ? henSales : 0);
    const totalOutflow = layerFeedCost + (includeRearingFeed ? rearingFeedCost : 0) + meds + (includeOverheads ? overheads : 0) + (includeTds ? tds : 0);
    const margin = totalInflow > 0 ? (profit / totalInflow) * 100 : 0;

    return {
      eggRev,
      discounts,
      henSales,
      totalFeedCost,
      layerFeedCost,
      rearingFeedCost,
      meds,
      overheads,
      labor,
      tds,
      profit,
      totalInflow,
      totalOutflow,
      margin,
    };
  }, [data, includeHenSales, includeRearingFeed, includeOverheads, includeDiscounts, includeTds]);

  if (loading) {
    return <div className="loading">Loading statistics...</div>;
  }

  if (error) {
    return <div className="error-message">{error}</div>;
  }

  if (!data || !calcDetails) {
    return <div className="no-data">No data available for this month</div>;
  }

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
      {/* Top Header */}
      <div className="monthly-statistics-header">
        <div>
          <h2>Monthly Performance - {new Date(year, month - 1).toLocaleString('default', { month: 'long', year: 'numeric' })}</h2>
          <span className="subtitle">Tiered Profitability & Farm Operational Breakdown</span>
        </div>
        {canEdit && (
          <button onClick={handleRefresh} className="refresh-btn">Refresh</button>
        )}
      </div>

      {/* Hero Profit Banner with Interactive Toggles */}
      <div className="profit-hero-card">
        <div className="profit-hero-top">
          <div className="profit-hero-main">
            <span className="profit-label">
              {activePreset === 'flock' && 'Flock Laying Operating Profit'}
              {activePreset === 'ledger' && 'Feed Provider Ledger Statement Profit'}
              {activePreset === 'comprehensive' && 'Comprehensive Farm Net Profit'}
              {activePreset === 'custom' && 'Custom Selected Net Profit'}
            </span>
            <div className={`profit-amount ${calcDetails.profit >= 0 ? 'positive' : 'negative'}`}>
              {formatCurrency(calcDetails.profit)}
            </div>
            <div className="profit-meta">
              <span className="profit-margin">
                Margin: <strong>{calcDetails.margin.toFixed(1)}%</strong> of effective revenue
              </span>
              <span className="profit-flow">
                Inflows: {formatLakhs(calcDetails.totalInflow)} | Outflows: {formatLakhs(calcDetails.totalOutflow)}
              </span>
            </div>
          </div>

          {/* Quick Presets */}
          <div className="preset-selector">
            <span className="preset-label">Quick Views:</span>
            <div className="preset-buttons">
              <button
                type="button"
                className={`preset-btn ${activePreset === 'flock' ? 'active' : ''}`}
                onClick={() => applyPreset('flock')}
                title="Laying Flock Only: Egg Sales + Discounts - Layer Feed - Meds"
              >
                🥚 Flock Operating
              </button>
              <button
                type="button"
                className={`preset-btn ${activePreset === 'ledger' ? 'active' : ''}`}
                onClick={() => applyPreset('ledger')}
                title="Ledger Statement: Eggs + Discounts - All Feeds - Meds - TDS"
              >
                📑 Statement Ledger
              </button>
              <button
                type="button"
                className={`preset-btn ${activePreset === 'comprehensive' ? 'active' : ''}`}
                onClick={() => applyPreset('comprehensive')}
                title="Comprehensive: All Revenues & Hen Sales - All Feeds, Meds, Overheads & TDS"
              >
                🚜 Farm Overall
              </button>
            </div>
          </div>
        </div>

        {/* Interactive Toggle Pills */}
        <div className="toggle-bar">
          <span className="toggle-bar-label">Include / Exclude from Profit:</span>
          <div className="toggle-pills">
            <label className={`toggle-pill ${includeHenSales ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeHenSales}
                onChange={() => handleToggle(setIncludeHenSales, includeHenSales)}
              />
              <span>Cull Hen Sales ({formatLakhs(calcDetails.henSales)})</span>
            </label>

            <label className={`toggle-pill ${includeRearingFeed ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeRearingFeed}
                onChange={() => handleToggle(setIncludeRearingFeed, includeRearingFeed)}
              />
              <span>Rearing Feed (Chick/Grower) (-{formatLakhs(calcDetails.rearingFeedCost)})</span>
            </label>

            <label className={`toggle-pill ${includeOverheads ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeOverheads}
                onChange={() => handleToggle(setIncludeOverheads, includeOverheads)}
              />
              <span>Farm Overheads (-{formatLakhs(calcDetails.overheads)})</span>
            </label>

            <label className={`toggle-pill ${includeDiscounts ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeDiscounts}
                onChange={() => handleToggle(setIncludeDiscounts, includeDiscounts)}
              />
              <span>Feed Discounts (+{formatLakhs(calcDetails.discounts)})</span>
            </label>

            <label className={`toggle-pill ${includeTds ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeTds}
                onChange={() => handleToggle(setIncludeTds, includeTds)}
              />
              <span>TDS (-{formatLakhs(calcDetails.tds)})</span>
            </label>
          </div>
        </div>

        {/* Dynamic Formula Breadcrumbs */}
        <div className="formula-banner">
          <span className="formula-badge">Formula:</span>
          <span className="formula-item">Eggs ({formatLakhs(calcDetails.eggRev)})</span>
          {includeDiscounts && <span className="formula-item add">+ Discounts ({formatLakhs(calcDetails.discounts)})</span>}
          {includeHenSales && <span className="formula-item add">+ Hen Sales ({formatLakhs(calcDetails.henSales)})</span>}
          <span className="formula-item sub">- Layer Feed ({formatLakhs(calcDetails.layerFeedCost)})</span>
          {includeRearingFeed && <span className="formula-item sub">- Rearing Feed ({formatLakhs(calcDetails.rearingFeedCost)})</span>}
          <span className="formula-item sub">- Meds ({formatLakhs(calcDetails.meds)})</span>
          {includeOverheads && <span className="formula-item sub">- Overheads ({formatLakhs(calcDetails.overheads)})</span>}
          {includeTds && calcDetails.tds > 0 && <span className="formula-item sub">- TDS ({formatLakhs(calcDetails.tds)})</span>}
          <span className="formula-equals">=</span>
          <span className="formula-result">{formatCurrency(calcDetails.profit)}</span>
        </div>
      </div>

      {/* Structured Functional Cards Grid */}
      <div className="sections-grid">
        {/* Section 1: Egg Revenue & Inflows */}
        <div className="section-card revenue-section">
          <div className="section-header">
            <h3>🥚 Egg Revenue & Inflows</h3>
            <span className="section-total">{formatCurrency(calcDetails.eggRev + calcDetails.discounts + calcDetails.henSales)}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('EGG')}>
              <div className="metric-left">
                <span className="metric-title">Eggs Sold</span>
                <span className="metric-sub">{formatNumber(data.total_eggs_sold)} Nos (Click for breakdown)</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_egg_price)}</span>
            </div>

            {data.egg_breakdown && data.egg_breakdown.length > 0 && (
              <div className="sub-breakdown">
                {data.egg_breakdown.map((item, idx) => (
                  <div key={idx} className="sub-breakdown-item">
                    <span>{item.type}: {formatNumber(item.quantity)} Nos</span>
                    {item.amount && <span>{formatCurrency(item.amount)}</span>}
                  </div>
                ))}
              </div>
            )}

            <div className="metric-row">
              <div className="metric-left">
                <span className="metric-title">Discounts & Credit Notes</span>
                <span className="metric-sub">Rebates from feed provider</span>
              </div>
              <span className="metric-value positive">+{formatCurrency(data.total_discounts || 0)}</span>
            </div>

            <div className="metric-row">
              <div className="metric-left">
                <span className="metric-title">Cull Hen Batch Sales</span>
                <span className="metric-sub">Flock realization income</span>
              </div>
              <span className="metric-value positive">+{formatCurrency(data.hen_sale_income || data.other_income || 0)}</span>
            </div>
          </div>
        </div>

        {/* Section 2: Feed & Nutrition */}
        <div className="section-card feed-section">
          <div className="section-header">
            <h3>🌾 Feed & Nutrition</h3>
            <span className="section-total">{formatCurrency(data.total_feed_price)}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('FEED')}>
              <div className="metric-left">
                <span className="metric-title">Total Feed Purchased</span>
                <span className="metric-sub">{formatNumber(data.feed_purchased_tonne)} Tonnes total</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_feed_price)}</span>
            </div>

            {/* Split: Layer Feed vs Rearing Feed */}
            <div className="feed-stage-split">
              <div className="feed-stage-box layer">
                <span className="stage-title">Layer Feed (Production)</span>
                <span className="stage-qty">{formatNumber(data.layer_feed_tonne ?? (data.feed_purchased_tonne - (data.rearing_feed_tonne || 0)))} T</span>
                <span className="stage-amount">{formatCurrency(calcDetails.layerFeedCost)}</span>
              </div>

              <div className="feed-stage-box rearing">
                <span className="stage-title">Rearing Feed (Chicks/Growers)</span>
                <span className="stage-qty">{formatNumber(data.rearing_feed_tonne ?? 0)} T</span>
                <span className="stage-amount">{formatCurrency(calcDetails.rearingFeedCost)}</span>
              </div>
            </div>

            {/* Itemized Feeds */}
            {data.feed_breakdown && data.feed_breakdown.length > 0 && (
              <div className="sub-breakdown">
                {data.feed_breakdown.map((item, idx) => (
                  <div key={idx} className="sub-breakdown-item">
                    <span className="feed-name-tag">
                      {item.type}
                      {item.stage && <span className={`stage-tag ${item.stage.toLowerCase()}`}>{item.stage}</span>}
                    </span>
                    <span>{formatNumber(item.quantity)} T {item.amount && `(${formatCurrency(item.amount)})`}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Section 3: Health & Farm Overheads */}
        <div className="section-card expense-section">
          <div className="section-header">
            <h3>💉 Health & Farm Overheads</h3>
            <span className="section-total">{formatCurrency((data.total_medicines || 0) + (data.other_expenses || 0))}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('MEDICINE')}>
              <div className="metric-left">
                <span className="metric-title">Medicines & Supplements</span>
                <span className="metric-sub">Vaccines, vitamins, grit</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_medicines)}</span>
            </div>

            <div className="metric-row clickable" onClick={() => handleShowBreakdown('OTHER')}>
              <div className="metric-left">
                <span className="metric-title">Farm Overheads & Utilities</span>
                <span className="metric-sub">Labor, electricity, EMI, repairs</span>
              </div>
              <span className="metric-value">{formatCurrency(data.other_expenses || 0)}</span>
            </div>

            {data.labor_expenses && data.labor_expenses > 0 && (
              <div className="sub-breakdown-item subtle">
                <span>↳ Of which Farm Labor wages:</span>
                <span>{formatCurrency(data.labor_expenses)}</span>
              </div>
            )}

            <div className="metric-row clickable" onClick={handleShowPayments}>
              <div className="metric-left">
                <span className="metric-title">Payments Received</span>
                <span className="metric-sub">Cash/Bank received from feed buyer</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_payments || 0)}</span>
            </div>
          </div>
        </div>

        {/* Section 4: Flock Productivity */}
        <div className="section-card productivity-section">
          <div className="section-header">
            <h3>🐔 Flock Productivity</h3>
            <span className="section-badge">{data.using_actual_count ? 'Batch Tracked' : 'Estimated'}</span>
          </div>

          <div className="productivity-grid">
            <div className="prod-box">
              <span className="prod-label">Active Head Count</span>
              <span className="prod-value">{formatNumber(data.estimated_hens || data.actual_head_count)}</span>
              <span className="prod-sub">{data.using_actual_count ? 'Based on hen batches' : 'Estimated from feed'}</span>
            </div>

            <div className="prod-box">
              <span className="prod-label">Laying Percentage</span>
              <span className="prod-value highlight">{formatNumber(data.egg_percentage)}%</span>
              <span className="prod-sub">Eggs per bird per day</span>
            </div>

            {data.chick_stage_expense && data.chick_stage_expense.enabled && (
              <div className="prod-box wide">
                <span className="prod-label">Young Flock Rearing: {data.chick_stage_expense.batch_name || 'Active Batch'}</span>
                <span className="prod-sub">
                  Month Rearing Spend: {formatCurrency(data.chick_stage_expense.monthly_total)} | Cumulative (to 25 wks): {formatCurrency(data.chick_stage_expense.total_till_25_weeks)}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Section 5: Vendor Statement Balance Reconciliation */}
      {(data.opening_balance !== undefined || data.closing_balance !== undefined) && (
        <div className="vendor-balance-card">
          <div className="vendor-balance-header">
            <h4>📑 Vendor Account Statement Reconciliation (Feed Provider)</h4>
          </div>
          <div className="vendor-balance-row">
            <div className="vendor-balance-item">
              <span className="v-label">Opening Balance</span>
              <span className="v-value">{formatCurrency(data.opening_balance)}</span>
            </div>
            <div className="vendor-balance-item">
              <span className="v-label">Net Ledger Activity</span>
              <span className="v-value">+{formatCurrency(calcDetails.eggRev + calcDetails.discounts - calcDetails.totalFeedCost - calcDetails.meds)}</span>
            </div>
            <div className="vendor-balance-item">
              <span className="v-label">Payments Received</span>
              <span className="v-value">-{formatCurrency(data.total_payments || 0)}</span>
            </div>
            <div className="vendor-balance-item highlight">
              <span className="v-label">Closing Balance Due</span>
              <span className="v-value">{formatCurrency(data.closing_balance)}</span>
            </div>
          </div>
        </div>
      )}

      {/* Modal Dialogs */}
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
