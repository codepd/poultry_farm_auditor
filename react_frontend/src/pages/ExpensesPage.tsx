import React, { useState, useEffect, useMemo } from 'react';
import { analyticsAPI, EnhancedMonthlySummary, transactionsAPI, Transaction } from '../services/api';
import api from '../services/api';
import { parseDateValue } from '../utils/dateUtils';
import './ExpensesPage.css';

// Common expense items
const COMMON_EXPENSES = [
  'Electricity Bill',
  'Water Bill',
  'Maintenance',
  'Repair',
  'Transportation',
  'Labor',
  'Office Supplies',
  'Insurance',
  'Security',
  'Legal Fees',
  'Bank Charges',
  'Telephone/Internet',
  'Fuel',
  'Rent',
  'Miscellaneous',
];

const OTHER_OPTION = 'Other';
const MONTH_OPTIONS = [
  { value: 1, label: 'January' },
  { value: 2, label: 'February' },
  { value: 3, label: 'March' },
  { value: 4, label: 'April' },
  { value: 5, label: 'May' },
  { value: 6, label: 'June' },
  { value: 7, label: 'July' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'October' },
  { value: 11, label: 'November' },
  { value: 12, label: 'December' },
];

const ExpensesPage: React.FC = () => {
  const [expenses, setExpenses] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [selectedYear, setSelectedYear] = useState<number>(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState<number>(new Date().getMonth() + 1);
  const [monthlySummary, setMonthlySummary] = useState<EnhancedMonthlySummary | null>(null);
  const [formData, setFormData] = useState({
    transaction_date: new Date().toISOString().split('T')[0],
    category: 'OTHER',
    selected_expense: '',        // Selected expense from dropdown or 'Other'
    item_name: '',               // Custom description when 'Other' is selected
    amount: 0,
    notes: '',
    payment_date: '',          // Optional: defaults to transaction_date
    period_month: '',           // Optional: defaults to payment_date month
    period_week: undefined as number | undefined,    // Optional
    period_days: undefined as number | undefined,    // Optional
  });

  useEffect(() => {
    fetchExpenses();
  }, []);

  useEffect(() => {
    const fetchMonthlySummary = async () => {
      try {
        const summary = await analyticsAPI.getEnhancedMonthlySummary(selectedYear, selectedMonth);
        setMonthlySummary(summary);
      } catch (err) {
        console.error('Failed to load monthly summary', err);
      }
    };
    fetchMonthlySummary();
  }, [selectedYear, selectedMonth]);

  const fetchExpenses = async () => {
    try {
      setLoading(true);
      setError(''); // Clear any previous errors
      const response = await api.get<{ success: boolean; data: Transaction[]; message?: string }>('/transactions', {
        params: {
          category: 'OTHER',
          transaction_type: 'EXPENSE',
        },
      });
      setExpenses(response.data.data || []);
      // Don't show error if message says no expenses found - that's normal
      if (response.data.message && !response.data.message.includes('No expenses found')) {
        console.warn(response.data.message);
      }
    } catch (err: any) {
      // Only show error for real failures, not empty data
      const status = err.response?.status;
      if (status && status >= 500) {
        setError('Server error. Please try again later.');
        console.error(err);
      } else {
        // Network errors or client errors - just set empty array
        setExpenses([]);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Determine item_name: use selected expense or custom description
    let itemName = '';
    if (formData.selected_expense === OTHER_OPTION) {
      itemName = formData.item_name.trim();
      if (!itemName) {
        setError('Please enter a description for "Other" expense');
        return;
      }
    } else if (formData.selected_expense) {
      itemName = formData.selected_expense;
    } else {
      setError('Please select an expense type');
      return;
    }
    
    try {
      // Prepare transaction data, only include payment period fields if they have values
      const transactionData: any = {
        transaction_date: formData.transaction_date,
        category: formData.category,
        item_name: itemName,
        amount: formData.amount,
        notes: formData.notes || undefined,
        transaction_type: 'EXPENSE',
        quantity: 1,
        unit: 'NOS',
        rate: formData.amount,
      };
      
      // Add payment period fields only if provided
      if (formData.payment_date) {
        transactionData.payment_date = formData.payment_date;
      }
      if (formData.period_month) {
        // Convert month input (YYYY-MM) to date string (YYYY-MM-01) for API
        transactionData.period_month = `${formData.period_month}-01`;
      }
      if (formData.period_week !== undefined && formData.period_week !== null) {
        transactionData.period_week = formData.period_week;
      }
      if (formData.period_days !== undefined && formData.period_days !== null) {
        transactionData.period_days = formData.period_days;
      }
      
      await transactionsAPI.createTransaction(transactionData);
      setShowAddForm(false);
      setFormData({
        transaction_date: new Date().toISOString().split('T')[0],
        category: 'OTHER',
        selected_expense: '',
        item_name: '',
        amount: 0,
        notes: '',
        payment_date: '',
        period_month: '',
        period_week: undefined,
        period_days: undefined,
      });
      setError(''); // Clear any previous errors
      fetchExpenses();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create expense');
    }
  };

  const totalExpenses = expenses.reduce((sum, exp) => sum + exp.amount, 0);

  const availableYears = useMemo(() => {
    const years = new Set<number>();
    years.add(new Date().getFullYear());
    expenses.forEach((exp) => {
      years.add(parseDateValue(exp.transaction_date).getFullYear());
    });
    return Array.from(years).sort((a, b) => b - a);
  }, [expenses]);

  const selectedMonthExpenses = expenses.filter((exp) => {
    const expenseDate = parseDateValue(exp.transaction_date);
    return expenseDate.getFullYear() === selectedYear && expenseDate.getMonth() + 1 === selectedMonth;
  });

  const selectedMonthTotal = selectedMonthExpenses.reduce((sum, exp) => sum + exp.amount, 0);
  const selectedMonthCount = selectedMonthExpenses.length;
  const selectedMonthItemBreakdown = useMemo(() => {
    const grouped = new Map<string, { item_name: string; total_amount: number; count: number }>();

    selectedMonthExpenses.forEach((expense) => {
      const key = (expense.item_name || 'Uncategorized').trim() || 'Uncategorized';
      const existing = grouped.get(key) || { item_name: key, total_amount: 0, count: 0 };
      existing.total_amount += expense.amount || 0;
      existing.count += 1;
      grouped.set(key, existing);
    });

    return Array.from(grouped.values()).sort((a, b) => b.total_amount - a.total_amount);
  }, [selectedMonthExpenses]);

  const recurringByType = selectedMonthExpenses.reduce(
    (acc, exp) => {
      const name = (exp.item_name || '').toUpperCase();
      if (name.includes('LABOR')) acc.labor += exp.amount;
      if (name.includes('ELECTRICITY')) acc.electricity += exp.amount;
      if (name.includes('EMI')) acc.emi += exp.amount;
      return acc;
    },
    { labor: 0, electricity: 0, emi: 0 }
  );

  const oneTimeExpenses = selectedMonthExpenses.filter((exp) =>
    (exp.item_name || '').toUpperCase().includes('ONE-TIME')
  );

  return (
    <div className="expenses-page">
      <div className="page-header">
        <h1>Miscellaneous Expenses</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-btn">
          {showAddForm ? 'Cancel' : '+ Add Expense'}
        </button>
      </div>

      <div className="add-expense-form">
        <h2>Month Selection</h2>
        <div className="form-row">
          <div className="form-group">
            <label>Year</label>
            <select
              value={selectedYear}
              onChange={(e) => setSelectedYear(parseInt(e.target.value, 10))}
              title="Select year"
              aria-label="Select year"
            >
              {availableYears.map((year) => (
                <option key={year} value={year}>
                  {year}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Month</label>
            <select
              value={selectedMonth}
              onChange={(e) => setSelectedMonth(parseInt(e.target.value, 10))}
              title="Select month"
              aria-label="Select month"
            >
              {MONTH_OPTIONS.map((month) => (
                <option key={month.value} value={month.value}>
                  {month.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="expenses-summary">
        <div className="summary-card">
          <div className="summary-label">Selected Month</div>
          <div className="summary-value">
            {new Date(selectedYear, selectedMonth - 1).toLocaleString('default', { month: 'long', year: 'numeric' })}
          </div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Month Total</div>
          <div className="summary-value">₹{selectedMonthTotal.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Month Transactions</div>
          <div className="summary-value">{selectedMonthCount}</div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Labor (Monthly)</div>
          <div className="summary-value">₹{recurringByType.labor.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Electricity (Monthly)</div>
          <div className="summary-value">₹{recurringByType.electricity.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="summary-label">EMI (Monthly)</div>
          <div className="summary-value">₹{recurringByType.emi.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</div>
        </div>
      </div>

      {monthlySummary?.chick_stage_expense?.enabled && (
        <div className="add-expense-form">
          <h2>Chick Rearing Expense (Till 25 Weeks)</h2>
          <div className="form-row">
            <div className="summary-card">
              <div className="summary-label">Batch</div>
              <div className="summary-value">{monthlySummary.chick_stage_expense.batch_name || '-'}</div>
            </div>
            <div className="summary-card">
              <div className="summary-label">This Month Total</div>
              <div className="summary-value">
                ₹{(monthlySummary.chick_stage_expense.monthly_total || 0).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
            <div className="summary-card">
              <div className="summary-label">Total Till 25 Weeks</div>
              <div className="summary-value">
                ₹{(monthlySummary.chick_stage_expense.total_till_25_weeks || 0).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
          </div>
          <div className="form-row">
            <div className="summary-card">
              <div className="summary-label">Chick Stage</div>
              <div className="summary-value">
                ₹{(monthlySummary.chick_stage_expense.total_till_25_weeks_chick || 0).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
            <div className="summary-card">
              <div className="summary-label">Grower Stage</div>
              <div className="summary-value">
                ₹{(monthlySummary.chick_stage_expense.total_till_25_weeks_grower || 0).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
            <div className="summary-card">
              <div className="summary-label">Pre-layer Stage</div>
              <div className="summary-value">
                ₹{(monthlySummary.chick_stage_expense.total_till_25_weeks_prelayer || 0).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
              </div>
            </div>
          </div>
        </div>
      )}

      {oneTimeExpenses.length > 0 && (
        <div className="add-expense-form">
          <h2>One-time Expenses ({new Date(selectedYear, selectedMonth - 1).toLocaleString('default', { month: 'short', year: 'numeric' })})</h2>
          <div className="expenses-mobile-list">
            {oneTimeExpenses.map((expense) => (
              <div key={expense.id} className="expense-mobile-card">
                <div className="expense-mobile-top">
                  <div className="expense-mobile-item">{expense.item_name || '-'}</div>
                </div>
                <div className="expense-mobile-amount">
                  ₹{expense.amount.toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="add-expense-form">
        <h2>
          Monthly Expense Breakdown ({new Date(selectedYear, selectedMonth - 1).toLocaleString('default', { month: 'short', year: 'numeric' })})
        </h2>
        {selectedMonthItemBreakdown.length === 0 ? (
          <div className="no-data">No expense entries in this month.</div>
        ) : (
          <>
            <div className="expenses-mobile-list">
              {selectedMonthItemBreakdown.map((item) => (
                <div key={item.item_name} className="expense-mobile-card">
                  <div className="expense-mobile-top">
                    <div className="expense-mobile-item">{item.item_name}</div>
                  </div>
                  <div className="expense-mobile-amount">
                    ₹{item.total_amount.toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                  </div>
                  <div className="expense-mobile-meta">
                    <span>{item.count} transaction{item.count > 1 ? 's' : ''}</span>
                  </div>
                </div>
              ))}
            </div>
            <div className="expenses-table">
              <table>
                <thead>
                  <tr>
                    <th>Item / Description</th>
                    <th>Transactions</th>
                    <th>Total Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedMonthItemBreakdown.map((item) => (
                    <tr key={item.item_name}>
                      <td>{item.item_name}</td>
                      <td>{item.count}</td>
                      <td>₹{item.total_amount.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>

      {showAddForm && (
        <div className="add-expense-form">
          <h2>Add New Expense</h2>
          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label>Date</label>
                <input
                  type="date"
                  value={formData.transaction_date}
                  onChange={(e) => setFormData({ ...formData, transaction_date: e.target.value })}
                  title="Expense date"
                  aria-label="Expense date"
                  required
                />
              </div>
              <div className="form-group">
                <label>Expense Type</label>
                <select
                  value={formData.selected_expense}
                  onChange={(e) => {
                    const value = e.target.value;
                    setFormData({ 
                      ...formData, 
                      selected_expense: value,
                      item_name: value === OTHER_OPTION ? formData.item_name : '' // Clear item_name if not Other
                    });
                  }}
                  required
                  className="expense-select"
                  title="Expense type"
                  aria-label="Expense type"
                >
                  <option value="">-- Select Expense Type --</option>
                  {COMMON_EXPENSES.map((expense) => (
                    <option key={expense} value={expense}>
                      {expense}
                    </option>
                  ))}
                  <option value={OTHER_OPTION}>{OTHER_OPTION}</option>
                </select>
                {formData.selected_expense === OTHER_OPTION && (
                  <div className="form-group other-expense-description-group">
                    <label>Description</label>
                <input
                  type="text"
                  value={formData.item_name}
                  onChange={(e) => setFormData({ ...formData, item_name: e.target.value })}
                      placeholder="Enter expense description"
                  required
                />
                  </div>
                )}
              </div>
              <div className="form-group">
                <label>Amount (₹)</label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.amount || ''}
                  onChange={(e) => setFormData({ ...formData, amount: parseFloat(e.target.value) || 0 })}
                  title="Expense amount"
                  aria-label="Expense amount"
                  required
                />
              </div>
            </div>
            
            <div className="form-section-header">
              <h3>Payment Period Information</h3>
              <p className="form-hint">Optional: Used to track when payment was made and which period it covers</p>
            </div>
            
            <div className="form-row">
              <div className="form-group">
                <label>Payment Date</label>
                <input
                  type="date"
                  value={formData.payment_date}
                  onChange={(e) => setFormData({ ...formData, payment_date: e.target.value })}
                  title="Date when payment was made (defaults to transaction date if not specified)"
                />
                <small className="form-hint">Defaults to transaction date if not specified</small>
              </div>
              <div className="form-group">
                <label>Period Month</label>
                <input
                  type="month"
                  value={formData.period_month ? formData.period_month.substring(0, 7) : ''}
                  onChange={(e) => setFormData({ ...formData, period_month: e.target.value || '' })}
                  title="Month the payment is for (e.g., for electricity bills)"
                />
                <small className="form-hint">Month the payment is for</small>
              </div>
            </div>
            
            <div className="form-row">
              <div className="form-group">
                <label>Period Week (Optional)</label>
                <input
                  type="number"
                  min="1"
                  max="52"
                  value={formData.period_week || ''}
                  onChange={(e) => setFormData({ ...formData, period_week: e.target.value ? parseInt(e.target.value) : undefined })}
                  placeholder="Week number"
                />
              </div>
              <div className="form-group">
                <label>Period Days (Optional)</label>
                <input
                  type="number"
                  min="1"
                  value={formData.period_days || ''}
                  onChange={(e) => setFormData({ ...formData, period_days: e.target.value ? parseInt(e.target.value) : undefined })}
                  placeholder="Number of days"
                />
              </div>
            </div>
            
            <div className="form-group">
              <label>Notes</label>
              <textarea
                value={formData.notes}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                rows={3}
                placeholder="Additional details..."
              />
            </div>
            <div className="form-actions">
              <button type="submit" className="submit-btn">Add Expense</button>
              <button type="button" onClick={() => setShowAddForm(false)} className="cancel-btn">
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="error-message">{error}</div>}

      {!loading && expenses.length === 0 && !error && (
        <div className="info-message">
          No expenses recorded yet. Add your first expense using the button above.
        </div>
      )}

      <div className="expenses-summary">
        <div className="summary-card">
          <div className="summary-label">Total Expenses</div>
          <div className="summary-value">₹{totalExpenses.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Number of Expenses</div>
          <div className="summary-value">{expenses.length}</div>
        </div>
      </div>

      {loading ? (
        <div className="loading">Loading expenses...</div>
      ) : selectedMonthExpenses.length === 0 ? (
        <div className="no-data">No expenses found. Add your first expense above.</div>
      ) : (
        <div className="expenses-results">
          <div className="expenses-mobile-list">
            {selectedMonthExpenses.map((expense) => (
              <div key={expense.id} className="expense-mobile-card">
                <div className="expense-mobile-top">
                  <div className="expense-mobile-item">{expense.item_name || '-'}</div>
                  <span className={`status-badge status-${expense.status.toLowerCase()}`}>
                    {expense.status}
                  </span>
                </div>
                <div className="expense-mobile-amount">
                  ₹{expense.amount.toLocaleString('en-IN', { minimumFractionDigits: 2 })}
                </div>
                <div className="expense-mobile-meta">
                  <span>{new Date(expense.transaction_date).toLocaleDateString()}</span>
                  {expense.notes && <span className="expense-mobile-notes">{expense.notes}</span>}
                </div>
              </div>
            ))}
          </div>
          <div className="expenses-table">
            <table>
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Item/Description</th>
                  <th>Amount</th>
                  <th>Notes</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {selectedMonthExpenses.map((expense) => (
                  <tr key={expense.id}>
                    <td>{new Date(expense.transaction_date).toLocaleDateString()}</td>
                    <td>{expense.item_name || '-'}</td>
                    <td>₹{expense.amount.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</td>
                    <td>{expense.notes || '-'}</td>
                    <td>
                      <span className={`status-badge status-${expense.status.toLowerCase()}`}>
                        {expense.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};

export default ExpensesPage;

