import React, { useState, useEffect, useRef, useCallback } from 'react';
import { analyticsAPI, dailyLogsAPI, usersAPI, DailyLogSubmission } from '../services/api';
import DailyQuickLogModal from '../components/Home/DailyQuickLogModal';
import OnboardingTour from '../onboarding/OnboardingTour';
import MonthlyStatistics from '../components/Home/MonthlyStatistics';
import YearlyStatistics from '../components/Home/YearlyStatistics';
import MonthlyBarChart from '../components/Home/MonthlyBarChart';
import DateRangeSummaryPanel from '../components/Home/DateRangeSummary';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import './HomePage.css';

const HomePage: React.FC = () => {
  const { currentTenant } = useAuth();
  const { t, language } = useLanguage();
  const [viewMode, setViewMode] = useState<'monthly' | 'yearly'>('monthly');
  const [showMobileQuickLog, setShowMobileQuickLog] = useState(false);
  const [showDateRange, setShowDateRange] = useState(false);

  const canEdit = Boolean(currentTenant && ['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(currentTenant.role));
  const isOtherUser = currentTenant?.role === 'OTHER_USER';
  const canLogDaily = Boolean(currentTenant);
  const [canViewSensitiveData, setCanViewSensitiveData] = useState<boolean>(
    () => currentTenant?.role !== 'MANAGER' && currentTenant?.role !== 'OTHER_USER'
  );
  const [pendingLogs, setPendingLogs] = useState<DailyLogSubmission[]>([]);
  const [myLogs, setMyLogs] = useState<DailyLogSubmission[]>([]);
  const [showApprovals, setShowApprovals] = useState(false);

  useEffect(() => {
    if (!currentTenant) return;
    if (currentTenant.role === 'OTHER_USER') {
      setCanViewSensitiveData(false);
      return;
    }
    if (usersAPI?.getMyPermissions) {
      Promise.resolve(usersAPI.getMyPermissions())
        .then((perms) => {
          if (perms && typeof perms.can_view_sensitive_data === 'boolean') {
            setCanViewSensitiveData(perms.can_view_sensitive_data);
          }
        })
        .catch(() => {});
    }
  }, [currentTenant]);

  const loadPendingLogs = useCallback(() => {
    if (isOtherUser) {
      if (!dailyLogsAPI?.list) return;
      Promise.resolve(dailyLogsAPI.list())
        .then((list) => setMyLogs(Array.isArray(list) ? list : []))
        .catch(() => {});
      return;
    }
    if (!canEdit || !dailyLogsAPI?.list) return;
    Promise.resolve(dailyLogsAPI.list('PENDING_APPROVAL'))
      .then((list) => setPendingLogs(Array.isArray(list) ? list : []))
      .catch(() => {});
  }, [canEdit, isOtherUser]);

  useEffect(() => {
    loadPendingLogs();
    window.addEventListener('transactionsUpdated', loadPendingLogs);
    return () => window.removeEventListener('transactionsUpdated', loadPendingLogs);
  }, [loadPendingLogs]);

  const handleApproveLog = async (id: number) => {
    try {
      await dailyLogsAPI.approve(id);
      window.dispatchEvent(new Event('transactionsUpdated'));
    } catch {
      // handled silently or via refresh
    }
  };

  const handleRejectLog = async (id: number) => {
    try {
      await dailyLogsAPI.reject(id);
      loadPendingLogs();
    } catch {
      // handled silently
    }
  };

  // Default to last month until we know the latest month that actually has data
  const lastMonth = new Date();
  lastMonth.setMonth(lastMonth.getMonth() - 1);
  const [selectedYear, setSelectedYear] = useState(lastMonth.getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(lastMonth.getMonth() + 1);
  const userPickedMonthRef = useRef(false);

  // Jump to the most recent month with data (e.g. the last loaded ledger), unless the user already picked a month.
  useEffect(() => {
    let cancelled = false;
    const syncLatest = () => {
      analyticsAPI
        .getLatestDataMonth()
        .then((latest) => {
          if (cancelled || userPickedMonthRef.current) return;
          if (latest?.has_data && latest.year && latest.month) {
            setSelectedYear(latest.year);
            setSelectedMonth(latest.month);
          }
        })
        .catch(() => {});
    };
    syncLatest();
    // After a ledger commit / quick-log entry, re-sync so the newly loaded month is shown
    window.addEventListener('transactionsUpdated', syncLatest);
    return () => {
      cancelled = true;
      window.removeEventListener('transactionsUpdated', syncLatest);
    };
  }, []);

  const handleMonthClick = (year: number, month: number) => {
    userPickedMonthRef.current = true;
    setSelectedYear(year);
    setSelectedMonth(month);
    setViewMode('monthly');
  };

  return (
    <div className="home-page">
      {/* Action shortcuts bar */}
      <div className="home-shortcuts-bar">
        {canLogDaily && (
          <button
            type="button"
            className="shortcut-pill active"
            data-tour="quick-log"
            onClick={() => setShowMobileQuickLog(true)}
          >
            {t('quickLog.btn')}
          </button>
        )}

        {canEdit && pendingLogs.length > 0 && (
          <button
            type="button"
            className={`shortcut-pill ${showApprovals ? 'active' : ''}`}
            onClick={() => setShowApprovals(!showApprovals)}
            style={{ background: showApprovals ? '#d97706' : '#fef3c7', color: showApprovals ? '#fff' : '#92400e', borderColor: '#f59e0b' }}
          >
            ⏳ Pending Approvals ({pendingLogs.length})
          </button>
        )}

        {!isOtherUser && (
          <button
            type="button"
            className={`shortcut-pill ${showDateRange ? 'active' : ''}`}
            onClick={() => setShowDateRange(!showDateRange)}
          >
            {showDateRange ? t('home.dateRangeClose') : t('home.dateRangeOpen')}
          </button>
        )}
      </div>

      {canEdit && showApprovals && pendingLogs.length > 0 && (
        <div className="collapsible-container" style={{ background: '#fff', padding: '1rem', borderRadius: '10px', marginBottom: '1rem', boxShadow: '0 2px 6px rgba(0,0,0,0.06)' }}>
          <h3 style={{ marginTop: 0, marginBottom: '0.75rem' }}>Pending Employee Submissions</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
            {pendingLogs.map((log) => (
              <div
                key={log.id}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  flexWrap: 'wrap',
                  gap: '0.75rem',
                  padding: '0.65rem 0.85rem',
                  border: '1px solid #e2e8f0',
                  borderRadius: '8px',
                }}
              >
                <div>
                  <strong>{log.log_type.replace('_', ' ')}</strong> · {log.log_date} ·{' '}
                  <span>{log.item_name || log.category}</span>
                  {log.quantity > 0 && <span> ({log.quantity} {log.unit})</span>}
                  {log.amount > 0 && <span> — ₹{log.amount.toLocaleString('en-IN')}</span>}
                  <div style={{ fontSize: '0.8rem', color: '#64748b' }}>
                    Submitted by {log.submitted_by_name || 'Employee'}
                    {log.notes ? ` · ${log.notes}` : ''}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button
                    type="button"
                    onClick={() => handleApproveLog(log.id)}
                    style={{ padding: '0.35rem 0.75rem', borderRadius: '6px', border: 'none', background: '#16a34a', color: '#fff', fontWeight: 600, cursor: 'pointer' }}
                  >
                    Approve
                  </button>
                  <button
                    type="button"
                    onClick={() => handleRejectLog(log.id)}
                    style={{ padding: '0.35rem 0.75rem', borderRadius: '6px', border: '1px solid #fecaca', background: '#fef2f2', color: '#dc2626', fontWeight: 600, cursor: 'pointer' }}
                  >
                    Reject
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {showDateRange && (
        <div className="collapsible-container">
          <DateRangeSummaryPanel />
        </div>
      )}

      {canLogDaily && showMobileQuickLog && (
        <DailyQuickLogModal onClose={() => setShowMobileQuickLog(false)} />
      )}

      <OnboardingTour />

      {isOtherUser ? (
        <div
          className="statistics-section"
          data-testid="employee-home-view"
          style={{ background: '#fff', padding: '1.25rem', borderRadius: '12px', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}
        >
          <h2 style={{ marginTop: 0, marginBottom: '0.35rem' }}>My Submitted Daily Logs</h2>
          <p style={{ color: '#64748b', fontSize: '0.9rem', marginTop: 0, marginBottom: '1rem' }}>
            Use <strong>Daily Quick Log</strong> above to record egg sales, feed purchases, or bird mortality. Managers/Owners will review and approve your entries.
          </p>
          {myLogs.length === 0 ? (
            <p style={{ color: '#64748b' }}>No daily log submissions yet.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
              {myLogs.map((log) => (
                <div
                  key={log.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: '0.5rem',
                    padding: '0.65rem 0.9rem',
                    border: '1px solid #e2e8f0',
                    borderRadius: '8px',
                  }}
                >
                  <div>
                    <strong>{log.log_type.replace('_', ' ')}</strong> · {log.log_date} ·{' '}
                    <span>{log.item_name || log.category}</span>
                    {log.quantity > 0 && <span> ({log.quantity} {log.unit})</span>}
                    {log.amount > 0 && <span> — ₹{log.amount.toLocaleString('en-IN')}</span>}
                  </div>
                  <span
                    style={{
                      padding: '0.2rem 0.6rem',
                      borderRadius: '999px',
                      fontSize: '0.78rem',
                      fontWeight: 600,
                      background:
                        log.status === 'APPROVED' || log.status === 'RECONCILED'
                          ? '#dcfce7'
                          : log.status === 'REJECTED'
                            ? '#fee2e2'
                            : '#fef3c7',
                      color:
                        log.status === 'APPROVED' || log.status === 'RECONCILED'
                          ? '#166534'
                          : log.status === 'REJECTED'
                            ? '#991b1b'
                            : '#92400e',
                    }}
                  >
                    {log.status.replace('_', ' ')}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="statistics-section">
          {viewMode === 'monthly' && (
            <MonthlyBarChart
              onMonthClick={handleMonthClick}
              canViewSensitiveData={canViewSensitiveData}
            />
          )}
          
          <div className="view-controls">
            <div className="view-toggle">
              <button
                className={viewMode === 'monthly' ? 'active' : ''}
                onClick={() => setViewMode('monthly')}
              >
                {t('home.monthly')}
              </button>
              <button
                className={viewMode === 'yearly' ? 'active' : ''}
                onClick={() => setViewMode('yearly')}
              >
                {t('home.yearly')}
              </button>
            </div>

            {viewMode === 'monthly' && (
              <div className="date-selector" data-tour="month">
                <select
                  value={selectedYear}
                  onChange={(e) => { userPickedMonthRef.current = true; setSelectedYear(parseInt(e.target.value)); }}
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
                  onChange={(e) => { userPickedMonthRef.current = true; setSelectedMonth(parseInt(e.target.value)); }}
                  title="Select month"
                  aria-label="Select month"
                >
                  {Array.from({ length: 12 }, (_, i) => i + 1).map((month) => (
                    <option key={month} value={month}>
                      {new Date(2000, month - 1).toLocaleString(language === 'ta' ? 'ta-IN' : 'en-IN', { month: 'long' })}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {viewMode === 'monthly' ? (
            <MonthlyStatistics
              year={selectedYear}
              month={selectedMonth}
              canViewSensitiveData={canViewSensitiveData}
            />
          ) : (
            <YearlyStatistics />
          )}
        </div>
      )}
    </div>
  );
};

export default HomePage;


