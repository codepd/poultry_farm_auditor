import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { analyticsAPI, EnhancedMonthlySummary, FlockAlert } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';
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
  const { language, t, tItem, tStage, formatMonthYear } = useLanguage();
  const navigate = useNavigate();
  const [data, setData] = useState<EnhancedMonthlySummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [breakdownCategory, setBreakdownCategory] = useState<string | null>(null);
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [copiedSummary, setCopiedSummary] = useState(false);

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

  const buildShareReportText = () => {
    const periodLabel = formatMonthYear(year, month);
    const tenantName = currentTenant?.name || 'MyKoliPannai';
    const avgEggPrice = data.egg_selling_price_per_egg ?? (data.total_eggs_sold > 0 ? calcDetails.eggRev / data.total_eggs_sold : 0);
    // Always compute Net Profit / Egg from the active preset's dynamic profit (calcDetails.profit)
    // so it matches the displayed Net Profit and is identical across English & Tamil views
    const netPerEgg = data.total_eggs_sold > 0 ? calcDetails.profit / data.total_eggs_sold : 0;
    const presetTitle =
      activePreset === 'flock'
        ? t('stats.profitFlock')
        : activePreset === 'ledger'
        ? t('stats.profitLedger')
        : activePreset === 'comprehensive'
        ? t('stats.profitComprehensive')
        : t('stats.profitCustom');

    if (language === 'ta') {
      return [
        `[கோழி] *${tenantName} - மாதாந்திர கணக்கு (${periodLabel})*`,
        `*${presetTitle}*`,
        `━━━━━━━━━━━━━━━━━━━━`,
        `• *விற்ற முட்டைகள்:* ${formatNumber(data.total_eggs_sold)} (₹${avgEggPrice.toFixed(2)}/முட்டை)`,
        `• *முட்டை வருவாய்:* ${formatCurrency(calcDetails.eggRev)}`,
        includeDiscounts && calcDetails.discounts > 0 ? `• *தீவன தள்ளுபடி:* +${formatCurrency(calcDetails.discounts)}` : null,
        includeHenSales && calcDetails.henSales > 0 ? `• *கழிவுக் கோழி விற்பனை:* +${formatCurrency(calcDetails.henSales)}` : null,
        `• *மொத்த தீவனம்:* ${formatNumber(data.feed_purchased_tonne)} டன் (${formatCurrency(calcDetails.totalFeedCost)})`,
        `  - முட்டை தீவனம்: ${formatNumber(data.layer_feed_tonne ?? 0)} T (${formatCurrency(calcDetails.layerFeedCost)})`,
        (data.rearing_feed_tonne || 0) > 0
          ? `  - வளர்ப்பு தீவனம்: ${formatNumber(data.rearing_feed_tonne)} T (${formatCurrency(calcDetails.rearingFeedCost)})${!includeRearingFeed ? ' [கணக்கில் நீக்கப்பட்டது]' : ''}`
          : null,
        `• *மருந்து செலவு:* ${formatCurrency(calcDetails.meds)}`,
        includeOverheads && calcDetails.overheads > 0
          ? `• *பண்ணை செலவுகள் (சம்பளம்/EB/EMI):* ${formatCurrency(calcDetails.overheads)}`
          : null,
        includeTds && calcDetails.tds > 0 ? `• *TDS:* ${formatCurrency(calcDetails.tds)}` : null,
        `━━━━━━━━━━━━━━━━━━━━`,
        `• *முட்டை உற்பத்தி %:* ${formatNumber(data.egg_percentage)}% (${formatNumber(data.estimated_hens || data.actual_head_count)} கோழிகள்)`,
        `• *நிகர லாபம் / முட்டை:* ₹${netPerEgg.toFixed(2)}`,
        `✓ *நிகர லாபம்:* *${formatCurrency(calcDetails.profit)}* (${calcDetails.margin.toFixed(1)}%)`,
      ]
        .filter(Boolean)
        .join('\n');
    }

    return [
      `[Farm] *${tenantName} - Monthly Report (${periodLabel})*`,
      `*${presetTitle}*`,
      `━━━━━━━━━━━━━━━━━━━━`,
      `• *Eggs Sold:* ${formatNumber(data.total_eggs_sold)} Nos (₹${avgEggPrice.toFixed(2)}/egg)`,
      `• *Egg Revenue:* ${formatCurrency(calcDetails.eggRev)}`,
      includeDiscounts && calcDetails.discounts > 0 ? `• *Feed Discounts:* +${formatCurrency(calcDetails.discounts)}` : null,
      includeHenSales && calcDetails.henSales > 0 ? `• *Cull Hen Sales:* +${formatCurrency(calcDetails.henSales)}` : null,
      `• *Total Feed:* ${formatNumber(data.feed_purchased_tonne)} T (${formatCurrency(calcDetails.totalFeedCost)})`,
      `  - Layer Feed: ${formatNumber(data.layer_feed_tonne ?? 0)} T (${formatCurrency(calcDetails.layerFeedCost)})`,
      (data.rearing_feed_tonne || 0) > 0
        ? `  - Rearing Feed: ${formatNumber(data.rearing_feed_tonne)} T (${formatCurrency(calcDetails.rearingFeedCost)})${!includeRearingFeed ? ' [Excluded from Profit]' : ''}`
        : null,
      `• *Medicines:* ${formatCurrency(calcDetails.meds)}`,
      includeOverheads && calcDetails.overheads > 0
        ? `• *Farm Overheads (Labor/EB/EMI):* ${formatCurrency(calcDetails.overheads)}`
        : null,
      includeTds && calcDetails.tds > 0 ? `• *TDS:* ${formatCurrency(calcDetails.tds)}` : null,
      `━━━━━━━━━━━━━━━━━━━━`,
      `• *Laying %:* ${formatNumber(data.egg_percentage)}% (${formatNumber(data.estimated_hens || data.actual_head_count)} hens)`,
      `• *Net Profit / Egg:* ₹${netPerEgg.toFixed(2)}`,
      `✓ *Net Profit:* *${formatCurrency(calcDetails.profit)}* (${calcDetails.margin.toFixed(1)}%)`,
    ]
      .filter(Boolean)
      .join('\n');
  };

  const handleCopySummary = async () => {
    const text = buildShareReportText();
    try {
      await navigator.clipboard.writeText(text);
      setCopiedSummary(true);
      setTimeout(() => setCopiedSummary(false), 2500);
    } catch {
      // Fallback
    }
  };

  const handleWhatsAppShare = () => {
    const text = buildShareReportText();
    const url = `https://wa.me/?text=${encodeURIComponent(text)}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  const renderFlockAlert = (alert: FlockAlert, idx: number) => {
    if (alert.code === 'UNREGISTERED_CHICK_BATCH' || alert.code === 'UNREGISTERED_GROWER_BATCH') {
      const isChick = alert.code === 'UNREGISTERED_CHICK_BATCH';
      const title = isChick ? t('alert.unregisteredChickTitle') : t('alert.unregisteredGrowerTitle');
      const desc = t('alert.unregisteredDesc', {
        tonnes: formatNumber(alert.feed_tonne),
        amount: formatCurrency(alert.feed_amount),
        stage: tStage(alert.stage),
        weeks: alert.estimated_current_age_weeks || (isChick ? 6 : 12),
      });
      return (
        <div key={idx} className={`flock-alert-banner ${alert.severity}`}>
          <div className="flock-alert-content">
            <span className="flock-alert-icon">🐣</span>
            <div>
              <div className="flock-alert-title">{title}</div>
              <div className="flock-alert-desc">{desc}</div>
            </div>
          </div>
          {canEdit && (
            <button
              type="button"
              className="flock-alert-action-btn"
              onClick={() => navigate('/hen-batches')}
            >
              {t('alert.registerBatchBtn')}
            </button>
          )}
        </div>
      );
    }

    if (alert.code === 'FEED_INTAKE_ANOMALY') {
      const gramsVal = formatNumber(alert.layer_feed_grams_per_hen_day ?? data.layer_feed_grams_per_hen_per_day);
      return (
        <div key={idx} className={`flock-alert-banner ${alert.severity}`}>
          <div className="flock-alert-content">
            <span className="flock-alert-icon">⚖️</span>
            <div>
              <div className="flock-alert-title">{t('alert.feedIntakeTitle')}</div>
              <div className="flock-alert-desc">{t('alert.feedIntakeDesc', { grams: gramsVal })}</div>
            </div>
          </div>
        </div>
      );
    }

    return null;
  };

  return (
    <div className="monthly-statistics">
      {/* Top Header */}
      <div className="monthly-statistics-header">
        <div>
          <h2>{t('stats.monthlyPerformance')} - {formatMonthYear(year, month)}</h2>
          <span className="subtitle">{t('stats.subtitle')}</span>
        </div>
        <div className="stats-header-actions">
          <button
            type="button"
            onClick={handleWhatsAppShare}
            className="share-whatsapp-btn"
            title="Share Monthly Summary via WhatsApp"
          >
            {t('stats.shareWhatsApp')}
          </button>
          <button
            type="button"
            onClick={handleCopySummary}
            className="copy-summary-btn"
            title="Copy Monthly Summary to Clipboard"
          >
            {copiedSummary ? t('stats.copied') : t('stats.copySummary')}
          </button>
          {canEdit && (
            <button onClick={handleRefresh} className="refresh-btn">{t('stats.refresh')}</button>
          )}
        </div>
      </div>

      {/* Smart Flock & Feed Intelligence Alerts */}
      {data.flock_alerts && data.flock_alerts.length > 0 && (
        <div className="flock-alerts-stack">
          {data.flock_alerts.map((alert, idx) => renderFlockAlert(alert, idx))}
        </div>
      )}

      {/* Hero Profit Banner with Interactive Toggles */}
      <div className="profit-hero-card">
        <div className="profit-hero-top">
          <div className="profit-hero-main">
            <span className="profit-label">
              {activePreset === 'flock' && t('stats.profitFlock')}
              {activePreset === 'ledger' && t('stats.profitLedger')}
              {activePreset === 'comprehensive' && t('stats.profitComprehensive')}
              {activePreset === 'custom' && t('stats.profitCustom')}
            </span>
            <div className={`profit-amount ${calcDetails.profit >= 0 ? 'positive' : 'negative'}`}>
              {formatCurrency(calcDetails.profit)}
            </div>
            <div className="profit-meta">
              <span className="profit-margin">
                {t('stats.marginLabel')} <strong>{calcDetails.margin.toFixed(1)}%</strong> ({t('stats.marginOfRevenue')})
              </span>
              <span className="profit-flow">
                {t('stats.inflows')}: {formatLakhs(calcDetails.totalInflow)} | {t('stats.outflows')}: {formatLakhs(calcDetails.totalOutflow)}
              </span>
            </div>
          </div>

          {/* Quick Presets */}
          <div className="preset-selector">
            <span className="preset-label">{t('stats.quickViews')}</span>
            <div className="preset-buttons">
              <button
                type="button"
                className={`preset-btn ${activePreset === 'flock' ? 'active' : ''}`}
                onClick={() => applyPreset('flock')}
                title="Laying Flock Only: Egg Sales + Discounts - Layer Feed - Meds"
              >
                {t('stats.presetFlock')}
              </button>
              <button
                type="button"
                className={`preset-btn ${activePreset === 'ledger' ? 'active' : ''}`}
                onClick={() => applyPreset('ledger')}
                title="Ledger Statement: Eggs + Discounts - All Feeds - Meds - TDS"
              >
                {t('stats.presetLedger')}
              </button>
              <button
                type="button"
                className={`preset-btn ${activePreset === 'comprehensive' ? 'active' : ''}`}
                onClick={() => applyPreset('comprehensive')}
                title="Comprehensive: All Revenues & Hen Sales - All Feeds, Meds, Overheads & TDS"
              >
                {t('stats.presetFarm')}
              </button>
            </div>
          </div>
        </div>

        {/* Interactive Toggle Pills */}
        <div className="toggle-bar">
          <span className="toggle-bar-label">{t('stats.includeExclude')}</span>
          <div className="toggle-pills">
            <label className={`toggle-pill ${includeHenSales ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeHenSales}
                onChange={() => handleToggle(setIncludeHenSales, includeHenSales)}
              />
              <span>{t('stats.toggleHenSales')} ({formatLakhs(calcDetails.henSales)})</span>
            </label>

            <label className={`toggle-pill ${includeRearingFeed ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeRearingFeed}
                onChange={() => handleToggle(setIncludeRearingFeed, includeRearingFeed)}
              />
              <span>{t('stats.toggleRearingFeed')} (-{formatLakhs(calcDetails.rearingFeedCost)})</span>
            </label>

            <label className={`toggle-pill ${includeOverheads ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeOverheads}
                onChange={() => handleToggle(setIncludeOverheads, includeOverheads)}
              />
              <span>{t('stats.toggleOverheads')} (-{formatLakhs(calcDetails.overheads)})</span>
            </label>

            <label className={`toggle-pill ${includeDiscounts ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeDiscounts}
                onChange={() => handleToggle(setIncludeDiscounts, includeDiscounts)}
              />
              <span>{t('stats.toggleDiscounts')} (+{formatLakhs(calcDetails.discounts)})</span>
            </label>

            <label className={`toggle-pill ${includeTds ? 'active' : 'inactive'}`}>
              <input
                type="checkbox"
                checked={includeTds}
                onChange={() => handleToggle(setIncludeTds, includeTds)}
              />
              <span>{t('stats.toggleTds')} (-{formatLakhs(calcDetails.tds)})</span>
            </label>
          </div>
        </div>

        {/* Dynamic Formula Breadcrumbs */}
        <div className="formula-banner">
          <span className="formula-badge">{t('stats.formula')}</span>
          <span className="formula-item">{t('stats.formulaEggs')} ({formatLakhs(calcDetails.eggRev)})</span>
          {includeDiscounts && <span className="formula-item add">+ {t('stats.formulaDiscounts')} ({formatLakhs(calcDetails.discounts)})</span>}
          {includeHenSales && <span className="formula-item add">+ {t('stats.formulaHenSales')} ({formatLakhs(calcDetails.henSales)})</span>}
          <span className="formula-item sub">- {t('stats.formulaLayerFeed')} ({formatLakhs(calcDetails.layerFeedCost)})</span>
          {includeRearingFeed && <span className="formula-item sub">- {t('stats.formulaRearingFeed')} ({formatLakhs(calcDetails.rearingFeedCost)})</span>}
          <span className="formula-item sub">- {t('stats.formulaMeds')} ({formatLakhs(calcDetails.meds)})</span>
          {includeOverheads && <span className="formula-item sub">- {t('stats.formulaOverheads')} ({formatLakhs(calcDetails.overheads)})</span>}
          {includeTds && calcDetails.tds > 0 && <span className="formula-item sub">- {t('stats.toggleTds')} ({formatLakhs(calcDetails.tds)})</span>}
          <span className="formula-equals">=</span>
          <span className="formula-result">{formatCurrency(calcDetails.profit)}</span>
        </div>
      </div>

      {/* Per-Egg Unit Economics & Feed Efficiency Strip */}
      {data.total_eggs_sold > 0 && (
        <div className="unit-economics-card">
          <div className="unit-economics-header">
            <h3>{t('stats.unitEconomicsTitle')}</h3>
            <span className="section-badge">{t('stats.unitEconomicsBadge')}</span>
          </div>
          <div className="unit-economics-grid">
            <div className="ue-item">
              <span className="ue-label">{t('stats.avgSellingPricePerEgg')}</span>
              <span className="ue-val">
                ₹{(calcDetails.eggRev / data.total_eggs_sold).toFixed(2)}
              </span>
            </div>
            <div className="ue-item">
              <span className="ue-label">{t('stats.layerFeedCostPerEgg')}</span>
              <span className="ue-val">
                ₹{(calcDetails.layerFeedCost / data.total_eggs_sold).toFixed(2)}
              </span>
            </div>
            <div className="ue-item">
              <span className="ue-label">{t('stats.allInCostPerEgg')}</span>
              <span className="ue-val">
                ₹{(calcDetails.totalOutflow / data.total_eggs_sold).toFixed(2)}
              </span>
            </div>
            <div className="ue-item highlight">
              <span className="ue-label">{t('stats.netMarginPerEgg')}</span>
              <span className={`ue-val ${calcDetails.profit >= 0 ? 'positive' : 'negative'}`}>
                ₹{(calcDetails.profit / data.total_eggs_sold).toFixed(2)}
              </span>
            </div>
            <div className="ue-item">
              <span className="ue-label">{t('stats.eggsPerKgFeed')}</span>
              <span className="ue-val">
                {(data.eggs_per_kg_layer_feed ?? 0).toFixed(2)}
              </span>
            </div>
            <div className="ue-item">
              <span className="ue-label">{t('stats.feedGramsPerHenDay')}</span>
              <span className="ue-val">
                {(data.layer_feed_grams_per_hen_per_day ?? 0).toFixed(1)} g
              </span>
              <span className="ue-sub">{t('stats.benchmarkNormal')}</span>
            </div>
          </div>
        </div>
      )}

      {/* Structured Functional Cards Grid */}
      <div className="sections-grid">
        {/* Section 1: Egg Revenue & Inflows */}
        <div className="section-card revenue-section">
          <div className="section-header">
            <h3>{t('stats.eggSectionTitle')}</h3>
            <span className="section-total">{formatCurrency(calcDetails.eggRev + calcDetails.discounts + calcDetails.henSales)}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('EGG')}>
              <div className="metric-left">
                <span className="metric-title">{t('stats.eggsSold')}</span>
                <span className="metric-sub">{formatNumber(data.total_eggs_sold)} {t('stats.clickForBreakdown')}</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_egg_price)}</span>
            </div>

            {data.egg_breakdown && data.egg_breakdown.length > 0 && (
              <div className="sub-breakdown">
                {data.egg_breakdown.map((item, idx) => (
                  <div key={idx} className="sub-breakdown-item">
                    <span>{tItem(item.type)}: {formatNumber(item.quantity)} Nos</span>
                    {item.amount && <span>{formatCurrency(item.amount)}</span>}
                  </div>
                ))}
              </div>
            )}

            <div className="metric-row">
              <div className="metric-left">
                <span className="metric-title">{t('stats.discountsCreditNotes')}</span>
                <span className="metric-sub">{t('stats.rebatesSub')}</span>
              </div>
              <span className="metric-value positive">+{formatCurrency(data.total_discounts || 0)}</span>
            </div>

            <div className="metric-row">
              <div className="metric-left">
                <span className="metric-title">{t('stats.cullHenSales')}</span>
                <span className="metric-sub">{t('stats.cullHenSub')}</span>
              </div>
              <span className="metric-value positive">+{formatCurrency(data.hen_sale_income || data.other_income || 0)}</span>
            </div>
          </div>
        </div>

        {/* Section 2: Feed & Nutrition */}
        <div className="section-card feed-section">
          <div className="section-header">
            <h3>{t('stats.feedSectionTitle')}</h3>
            <span className="section-total">{formatCurrency(data.total_feed_price)}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('FEED')}>
              <div className="metric-left">
                <span className="metric-title">{t('stats.totalFeedPurchased')}</span>
                <span className="metric-sub">{formatNumber(data.feed_purchased_tonne)} {t('stats.tonnesTotal')}</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_feed_price)}</span>
            </div>

            {/* Split: Layer Feed vs Rearing Feed */}
            <div className="feed-stage-split">
              <div className="feed-stage-box layer">
                <span className="stage-title">{t('stats.layerFeedProd')}</span>
                <span className="stage-qty">{formatNumber(data.layer_feed_tonne ?? (data.feed_purchased_tonne - (data.rearing_feed_tonne || 0)))} T</span>
                <span className="stage-amount">{formatCurrency(calcDetails.layerFeedCost)}</span>
              </div>

              <div className="feed-stage-box rearing">
                <span className="stage-title">{t('stats.rearingFeedSub')}</span>
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
                      {tItem(item.type)}
                      {item.stage && <span className={`stage-tag ${item.stage.toLowerCase()}`}>{tStage(item.stage)}</span>}
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
            <h3>{t('stats.healthOverheadsTitle')}</h3>
            <span className="section-total">{formatCurrency((data.total_medicines || 0) + (data.other_expenses || 0))}</span>
          </div>

          <div className="metrics-list">
            <div className="metric-row clickable" onClick={() => handleShowBreakdown('MEDICINE')}>
              <div className="metric-left">
                <span className="metric-title">{t('stats.medicinesSupplements')}</span>
                <span className="metric-sub">{t('stats.medicinesSub')}</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_medicines)}</span>
            </div>

            <div className="metric-row clickable" onClick={() => handleShowBreakdown('OTHER')}>
              <div className="metric-left">
                <span className="metric-title">{t('stats.farmOverheadsUtilities')}</span>
                <span className="metric-sub">{t('stats.overheadsSub')}</span>
              </div>
              <span className="metric-value">{formatCurrency(data.other_expenses || 0)}</span>
            </div>

            {data.labor_expenses && data.labor_expenses > 0 && (
              <div className="sub-breakdown-item subtle">
                <span>{t('stats.ofWhichLabor')}</span>
                <span>{formatCurrency(data.labor_expenses)}</span>
              </div>
            )}

            <div className="metric-row clickable" onClick={handleShowPayments}>
              <div className="metric-left">
                <span className="metric-title">{t('stats.paymentsReceived')}</span>
                <span className="metric-sub">{t('stats.paymentsSub')}</span>
              </div>
              <span className="metric-value">{formatCurrency(data.total_payments || 0)}</span>
            </div>
          </div>
        </div>

        {/* Section 4: Flock Productivity */}
        <div className="section-card productivity-section">
          <div className="section-header">
            <h3>{t('stats.productivityTitle')}</h3>
            <span className="section-badge">{data.using_actual_count ? t('stats.batchTracked') : t('stats.estimated')}</span>
          </div>

          <div className="productivity-grid">
            <div className="prod-box">
              <span className="prod-label">{t('stats.activeHeadCount')}</span>
              <span className="prod-value">{formatNumber(data.estimated_hens || data.actual_head_count)}</span>
              <span className="prod-sub">{data.using_actual_count ? t('stats.basedOnBatches') : t('stats.estimatedFromFeed')}</span>
            </div>

            <div className="prod-box">
              <span className="prod-label">{t('stats.layingPercentage')}</span>
              <span className="prod-value highlight">{formatNumber(data.egg_percentage)}%</span>
              <span className="prod-sub">{t('stats.eggsPerBirdDay')}</span>
            </div>

            {data.chick_stage_expense && data.chick_stage_expense.enabled && (
              <div className="prod-box wide">
                <span className="prod-label">{t('stats.youngFlockRearing')}: {data.chick_stage_expense.batch_name || 'Active Batch'}</span>
                <span className="prod-sub">
                  {t('stats.monthRearingSpend')}: {formatCurrency(data.chick_stage_expense.monthly_total)} | {t('stats.cumulativeTo25Wks')}: {formatCurrency(data.chick_stage_expense.total_till_25_weeks)}
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
            <h4>{t('stats.vendorReconciliation')}</h4>
          </div>
          <div className="vendor-balance-row">
            <div className="vendor-balance-item">
              <span className="v-label">{t('stats.openingBalance')}</span>
              <span className="v-value">{formatCurrency(data.opening_balance)}</span>
            </div>
            <div className="vendor-balance-item">
              <span className="v-label">{t('stats.netLedgerActivity')}</span>
              <span className="v-value">+{formatCurrency(calcDetails.eggRev + calcDetails.discounts - calcDetails.totalFeedCost - calcDetails.meds)}</span>
            </div>
            <div className="vendor-balance-item">
              <span className="v-label">{t('stats.paymentsReceived')}</span>
              <span className="v-value">-{formatCurrency(data.total_payments || 0)}</span>
            </div>
            <div className="vendor-balance-item highlight">
              <span className="v-label">{t('stats.closingBalanceDue')}</span>
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
