import React, { useState, useEffect } from 'react';
import { transactionsAPI, henBatchesAPI, dailyLogsAPI, HenBatch } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';

interface DailyQuickLogModalProps {
  onClose: () => void;
  onSaved?: () => void;
}

type QuickTab = 'egg' | 'expense' | 'mortality';
type ExpenseCategory = 'FEED' | 'MEDICINE' | 'OTHER';

const CUSTOM_ITEM_VALUE = '__CUSTOM__';

const EXPENSE_ITEM_OPTIONS: Record<ExpenseCategory, string[]> = {
  FEED: ['LAYER MASH', 'PRE-LAYER MASH', 'GROWER MASH', 'CHICK MASH'],
  MEDICINE: ['FEED SUPPLEMENTS', 'D3 FORTE', 'VETMULIN', 'OXYCYCLINE', 'TIAZIN', 'BPPS FORTE', 'CTC', 'SHELL GRIT'],
  OTHER: ['LABOR', 'ELECTRICITY', 'MAINTENANCE', 'TRANSPORT', 'DIESEL / GENERATOR'],
};

const DailyQuickLogModal: React.FC<DailyQuickLogModalProps> = ({ onClose, onSaved }) => {
  const { currentTenant } = useAuth();
  const { t } = useLanguage();
  const requiresApproval = Boolean(
    currentTenant && !['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(currentTenant.role)
  );
  const [activeTab, setActiveTab] = useState<QuickTab>('egg');
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10));

  // Egg Sale state
  const [eggType, setEggType] = useState('LARGE EGG');
  const [eggCount, setEggCount] = useState('');
  const [ratePerEgg, setRatePerEgg] = useState('');

  // Feed / Expense state
  const [expenseCategory, setExpenseCategory] = useState<ExpenseCategory>('FEED');
  const [selectedItemOption, setSelectedItemOption] = useState<string>(EXPENSE_ITEM_OPTIONS.FEED[0]);
  const [customItemName, setCustomItemName] = useState('');
  const [amount, setAmount] = useState('');

  // Mortality state
  const [batches, setBatches] = useState<HenBatch[]>([]);
  const [selectedBatchId, setSelectedBatchId] = useState<number>(0);
  const [mortalityCount, setMortalityCount] = useState('');
  const [mortalityNotes, setMortalityNotes] = useState('');

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    henBatchesAPI
      .getHenBatches()
      .then((list) => {
        setBatches(list);
        if (list.length > 0) {
          setSelectedBatchId(list[0].id);
        }
      })
      .catch(() => {});
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setSaving(true);

    try {
      if (activeTab === 'egg') {
        const qty = parseFloat(eggCount);
        const rate = parseFloat(ratePerEgg);
        if (!qty || qty <= 0 || !rate || rate <= 0) {
          setError('Please enter valid egg count and rate per egg');
          setSaving(false);
          return;
        }
        if (requiresApproval) {
          await dailyLogsAPI.submit({
            log_type: 'EGG_SALE',
            log_date: date,
            category: 'EGG',
            item_name: eggType,
            quantity: qty,
            unit: 'NOS',
            rate,
            amount: Number((qty * rate).toFixed(2)),
          });
        } else {
          await transactionsAPI.createTransaction({
            transaction_date: date,
            transaction_type: 'SALE',
            category: 'EGG',
            item_name: eggType,
            quantity: qty,
            unit: 'NOS',
            rate,
            amount: Number((qty * rate).toFixed(2)),
          });
        }
      } else if (activeTab === 'expense') {
        const total = parseFloat(amount);
        const resolvedItemName =
          selectedItemOption === CUSTOM_ITEM_VALUE ? customItemName.trim() : selectedItemOption;
        if (!resolvedItemName || !total || total <= 0) {
          setError('Please enter item description and amount');
          setSaving(false);
          return;
        }
        if (requiresApproval) {
          await dailyLogsAPI.submit({
            log_type: expenseCategory === 'FEED' ? 'FEED_PURCHASE' : 'EXPENSE',
            log_date: date,
            category: expenseCategory,
            item_name: resolvedItemName.toUpperCase(),
            amount: total,
          });
        } else {
          await transactionsAPI.createTransaction({
            transaction_date: date,
            transaction_type: expenseCategory === 'FEED' || expenseCategory === 'MEDICINE' ? 'PURCHASE' : 'EXPENSE',
            category: expenseCategory,
            item_name: resolvedItemName.toUpperCase(),
            amount: total,
          });
        }
      } else {
        const count = parseInt(mortalityCount, 10);
        if (!selectedBatchId || !count || count <= 0) {
          setError('Please select a batch and enter mortality count');
          setSaving(false);
          return;
        }
        if (requiresApproval) {
          await dailyLogsAPI.submit({
            log_type: 'MORTALITY',
            log_date: date,
            category: 'MORTALITY',
            item_name: 'Mortality',
            quantity: count,
            unit: 'BIRDS',
            hen_batch_id: selectedBatchId,
            notes: mortalityNotes.trim() || undefined,
          });
        } else {
          await henBatchesAPI.createMortality({
            batch_id: selectedBatchId,
            mortality_date: date,
            count,
            notes: mortalityNotes.trim() || undefined,
          });
        }
      }

      window.dispatchEvent(new Event('transactionsUpdated'));
      setSuccess(requiresApproval ? 'Submitted for Manager / Owner approval.' : t('quickLog.savedSuccess'));
      setEggCount('');
      setRatePerEgg('');
      setCustomItemName('');
      setAmount('');
      setMortalityCount('');
      setMortalityNotes('');
      if (onSaved) onSaved();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save entry');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className="header-modal-backdrop"
      onClick={onClose}
      data-testid="daily-quick-log-modal"
    >
      <div
        className="header-modal-card"
        style={{ maxWidth: '460px' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
          <h3 style={{ margin: 0 }}>{t('quickLog.title')}</h3>
          <button
            type="button"
            onClick={onClose}
            style={{ border: 'none', background: 'transparent', fontSize: '1.2rem', cursor: 'pointer', color: '#64748b' }}
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <p className="header-modal-desc" style={{ marginBottom: '0.9rem' }}>
          {t('quickLog.subtitle')}
        </p>

        <div style={{ display: 'flex', gap: '0.4rem', marginBottom: '1rem' }}>
          {(['egg', 'expense', 'mortality'] as QuickTab[]).map((tab) => {
            const label =
              tab === 'egg'
                ? t('quickLog.tabEggs')
                : tab === 'expense'
                ? t('quickLog.tabExpense')
                : t('quickLog.tabMortality');
            const active = activeTab === tab;
            return (
              <button
                key={tab}
                type="button"
                onClick={() => { setActiveTab(tab); setError(''); setSuccess(''); }}
                style={{
                  flex: 1,
                  padding: '0.55rem 0.4rem',
                  borderRadius: '8px',
                  border: active ? '1px solid #4f46e5' : '1px solid #e2e8f0',
                  background: active ? '#eef2ff' : '#f8fafc',
                  color: active ? '#4338ca' : '#475569',
                  fontWeight: 600,
                  fontSize: '0.8rem',
                  cursor: 'pointer',
                }}
              >
                {label}
              </button>
            );
          })}
        </div>

        {error && <div className="mandatory-password-error">{error}</div>}
        {success && (
          <div
            style={{
              background: '#f0fdf4',
              color: '#15803d',
              border: '1px solid #bbf7d0',
              padding: '0.55rem 0.75rem',
              borderRadius: '8px',
              fontSize: '0.85rem',
              marginBottom: '0.75rem',
              fontWeight: 600,
            }}
          >
            {success}
          </div>
        )}

        <form onSubmit={handleSubmit} className="mandatory-password-form">
          <label>
            <span>{t('quickLog.date')}</span>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="password-input"
              required
            />
          </label>

          {activeTab === 'egg' && (
            <>
              <label>
                <span>{t('quickLog.eggType')}</span>
                <select
                  value={eggType}
                  onChange={(e) => setEggType(e.target.value)}
                  className="password-input"
                >
                  <option value="LARGE EGG">LARGE EGG</option>
                  <option value="MEDIUM EGG">MEDIUM EGG</option>
                  <option value="SMALL EGG">SMALL EGG</option>
                  <option value="BROKEN EGG">BROKEN EGG</option>
                </select>
              </label>
              <label>
                <span>{t('quickLog.eggCount')}</span>
                <input
                  type="number"
                  value={eggCount}
                  onChange={(e) => setEggCount(e.target.value)}
                  placeholder="e.g. 25200"
                  className="password-input"
                  required
                />
              </label>
              <label>
                <span>{t('quickLog.ratePerEgg')}</span>
                <input
                  type="number"
                  step="0.01"
                  value={ratePerEgg}
                  onChange={(e) => setRatePerEgg(e.target.value)}
                  placeholder="e.g. 5.45"
                  className="password-input"
                  required
                />
              </label>
            </>
          )}

          {activeTab === 'expense' && (
            <>
              <label>
                <span>{t('quickLog.category')}</span>
                <select
                  value={expenseCategory}
                  onChange={(e) => {
                    const nextCat = e.target.value as ExpenseCategory;
                    setExpenseCategory(nextCat);
                    setSelectedItemOption(EXPENSE_ITEM_OPTIONS[nextCat][0]);
                    setCustomItemName('');
                  }}
                  className="password-input"
                >
                  <option value="FEED">FEED</option>
                  <option value="MEDICINE">MEDICINE</option>
                  <option value="OTHER">OTHER EXPENSE</option>
                </select>
              </label>
              <label>
                <span>{t('quickLog.itemName')}</span>
                <select
                  value={selectedItemOption}
                  onChange={(e) => setSelectedItemOption(e.target.value)}
                  className="password-input"
                  aria-label="Expense item"
                >
                  {EXPENSE_ITEM_OPTIONS[expenseCategory].map((opt) => (
                    <option key={opt} value={opt}>
                      {opt}
                    </option>
                  ))}
                  <option value={CUSTOM_ITEM_VALUE}>Other (Custom)...</option>
                </select>
              </label>
              {selectedItemOption === CUSTOM_ITEM_VALUE && (
                <label>
                  <span>Custom {expenseCategory === 'FEED' ? 'Feed' : expenseCategory === 'MEDICINE' ? 'Medicine' : 'Expense'} Name</span>
                  <input
                    type="text"
                    value={customItemName}
                    onChange={(e) => setCustomItemName(e.target.value)}
                    placeholder={
                      expenseCategory === 'FEED'
                        ? 'e.g. STARTER CRUMBLE'
                        : expenseCategory === 'MEDICINE'
                        ? 'e.g. LIVER TONIC'
                        : 'e.g. WATER TANKER'
                    }
                    className="password-input"
                    required
                  />
                </label>
              )}
              <label>
                <span>{t('quickLog.amount')}</span>
                <input
                  type="number"
                  step="0.01"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="e.g. 15000"
                  className="password-input"
                  required
                />
              </label>
            </>
          )}

          {activeTab === 'mortality' && (
            <>
              <label>
                <span>{t('quickLog.batch')}</span>
                <select
                  value={selectedBatchId}
                  onChange={(e) => setSelectedBatchId(Number(e.target.value))}
                  className="password-input"
                >
                  {batches.map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.batch_name} ({b.current_count} birds)
                    </option>
                  ))}
                </select>
              </label>
              <label>
                <span>{t('quickLog.mortalityCount')}</span>
                <input
                  type="number"
                  value={mortalityCount}
                  onChange={(e) => setMortalityCount(e.target.value)}
                  placeholder="e.g. 5"
                  className="password-input"
                  required
                />
              </label>
              <label>
                <span>{t('quickLog.reason')}</span>
                <input
                  type="text"
                  value={mortalityNotes}
                  onChange={(e) => setMortalityNotes(e.target.value)}
                  placeholder="Optional notes"
                  className="password-input"
                />
              </label>
            </>
          )}

          <button type="submit" className="mandatory-save-btn" disabled={saving}>
            {saving ? t('quickLog.saving') : t('quickLog.saveBtn')}
          </button>
        </form>
      </div>
    </div>
  );
};

export default DailyQuickLogModal;
