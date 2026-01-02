import React, { useState, useEffect } from 'react';
import { transactionsAPI, Transaction } from '../services/api';
import api from '../services/api';
import { useAuth } from '../context/AuthContext';
import MonthlyBarChart from '../components/Home/MonthlyBarChart';
import './ExpensesPage.css';

const ExpensesPage: React.FC = () => {
  const { currentTenant } = useAuth();
  const [expenses, setExpenses] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [formData, setFormData] = useState({
    transaction_date: new Date().toISOString().split('T')[0],
    category: 'OTHER',
    item_name: '',
    amount: 0,
    notes: '',
  });

  useEffect(() => {
    fetchExpenses();
  }, []);

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
    try {
      await transactionsAPI.createTransaction({
        ...formData,
        transaction_type: 'EXPENSE',
        quantity: 1,
        unit: 'NOS',
        rate: formData.amount,
      });
      setShowAddForm(false);
      setFormData({
        transaction_date: new Date().toISOString().split('T')[0],
        category: 'OTHER',
        item_name: '',
        amount: 0,
        notes: '',
      });
      fetchExpenses();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create expense');
    }
  };

  const totalExpenses = expenses.reduce((sum, exp) => sum + exp.amount, 0);

  const handleMonthClick = (year: number, month: number) => {
    // Filter expenses to show selected month
    const filtered = expenses.filter(exp => {
      const expDate = new Date(exp.transaction_date);
      return expDate.getFullYear() === year && expDate.getMonth() + 1 === month;
    });
    // Could navigate or show details - for now just log
    console.log(`Selected month: ${year}-${month}, Expenses:`, filtered);
  };

  return (
    <div className="expenses-page">
      <div className="page-header">
        <h1>Miscellaneous Expenses</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-btn">
          {showAddForm ? 'Cancel' : '+ Add Expense'}
        </button>
      </div>

      <MonthlyBarChart onMonthClick={handleMonthClick} />

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
                  required
                />
              </div>
              <div className="form-group">
                <label>Item/Description</label>
                <input
                  type="text"
                  value={formData.item_name}
                  onChange={(e) => setFormData({ ...formData, item_name: e.target.value })}
                  placeholder="e.g., Electricity Bill, Maintenance"
                  required
                />
              </div>
              <div className="form-group">
                <label>Amount (₹)</label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.amount || ''}
                  onChange={(e) => setFormData({ ...formData, amount: parseFloat(e.target.value) || 0 })}
                  required
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
      ) : expenses.length === 0 ? (
        <div className="no-data">No expenses found. Add your first expense above.</div>
      ) : (
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
              {expenses.map((expense) => (
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
      )}
    </div>
  );
};

export default ExpensesPage;

