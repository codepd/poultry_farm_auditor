import React, { useState, useEffect } from 'react';
import { transactionsAPI, henBatchesAPI, HenBatch } from '../../services/api';
import { useLanguage } from '../../context/LanguageContext';

interface DailyQuickLogModalProps {
  onClose: () => void;
  onSaved?: () => void;
}

type QuickTab = 'egg' | 'expense' | 'mortality';

const DailyQuickLogModal: React.FC<DailyQuickLogModalProps> = ({ onClose, onSaved }) => {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState<QuickTab>('egg');
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10));

  // Egg Sale state
  const [eggType, setEggType] = useState('LARGE EGG');
  const [eggCount, setEggCount] = useState('');
  const [ratePerEgg, setRatePerEgg] = useState('');

  // Feed / Expense state
  const [expenseCategory, setExpenseCategory] = useState<'FEED' | 'MEDICINE' | 'OTHER'>('FEED');
  const [itemName, setItemName] = useState('');
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
      } else if (activeTab === 'expense') {
        const total = parseFloat(amount);
        if (!itemName.trim() || !total || total <= 0) {
          setError('Please enter item description and amount');
          setSaving(false);
          return;
        }
        await transactionsAPI.createTransaction({
          transaction_date: date,
          transaction_type: expenseCategory === 'FEED' || expenseCategory === 'MEDICINE' ? 'PURCHASE' : 'EXPENSE',
          category: expenseCategory,
          item_name: itemName.trim().toUpperCase(),
          amount: total,
        });
      } else {
        const count = parseInt(mortalityCount, 10);
        if (!selectedBatchId || !count || count <= 0) {
          setError('Please select a batch and enter mortality count');
          setSaving(false);
          return;
        }
        await henBatchesAPI.createMortality({
          batch_id: selectedBatchId,
          mortality_date: date,
          count,
          notes: mortalityNotes.trim() || undefined,
        });
      }

      window.dispatchEvent(new Event('transactionsUpdated'));
      setSuccess(t('quickLog.savedSuccess'));
      setEggCount('');
      setRatePerEgg('');
      setItemName('');
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
                  onChange={(e) => setExpenseCategory(e.target.value as any)}
                  className="password-input"
                >
                  <option value="FEED">FEED</option>
                  <option value="MEDICINE">MEDICINE</option>
                  <option value="OTHER">OTHER EXPENSE</option>
                </select>
              </label>
              <label>
                <span>{t('quickLog.itemName')}</span>
                <input
                  type="text"
                  value={itemName}
                  onChange={(e) => setItemName(e.target.value)}
                  placeholder="e.g. LAYER MASH / ELECTRICITY / LABOR"
                  className="password-input"
                  required
                />
              </label>
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
