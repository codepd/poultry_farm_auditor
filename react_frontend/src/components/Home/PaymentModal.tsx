import React, { useState, useEffect } from 'react';
import { analyticsAPI, transactionsAPI, Transaction } from '../../services/api';
import './PaymentModal.css';

interface PaymentModalProps {
  isOpen: boolean;
  onClose: () => void;
  year: number;
  month: number;
  canEdit: boolean;
  onUpdate: () => void;
}

const PaymentModal: React.FC<PaymentModalProps> = ({
  isOpen,
  onClose,
  year,
  month,
  canEdit,
  onUpdate,
}) => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editForm, setEditForm] = useState<Partial<Transaction>>({});
  const [showAddForm, setShowAddForm] = useState(false);
  const [newPayment, setNewPayment] = useState({
    transaction_date: '',
    amount: '',
    item_name: '',
    notes: '',
  });
  const [breakdownData, setBreakdownData] = useState<{
    average_price: number;
    grouped_by_date: Array<{
      date: string;
      transactions: Transaction[];
      total_amount: number;
    }>;
  } | null>(null);

  useEffect(() => {
    if (isOpen) {
      fetchBreakdown();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, year, month]);

  const fetchBreakdown = async () => {
    setLoading(true);
    try {
      const breakdown = await analyticsAPI.getMonthlyBreakdown(year, month, 'PAYMENT');
      setTransactions(breakdown.transactions || []);
      setBreakdownData({
        average_price: 0,
        grouped_by_date: breakdown.grouped_by_date || [],
      });
    } catch (err) {
      console.error('Failed to fetch breakdown:', err);
      setTransactions([]);
      setBreakdownData({
        average_price: 0,
        grouped_by_date: [],
      });
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (transaction: Transaction) => {
    setEditingId(transaction.id);
    setEditForm({
      amount: transaction.amount,
      item_name: transaction.item_name,
      transaction_date: transaction.transaction_date,
      notes: transaction.notes,
    });
  };

  const handleSave = async (id: number) => {
    try {
      await transactionsAPI.updateTransaction(id, editForm);
      setEditingId(null);
      fetchBreakdown();
      onUpdate();
    } catch (err) {
      console.error('Failed to update transaction:', err);
      alert('Failed to update transaction. Please try again.');
    }
  };

  const handleCancel = () => {
    setEditingId(null);
    setEditForm({});
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this payment?')) {
      return;
    }
    try {
      await transactionsAPI.deleteTransaction(id);
      fetchBreakdown();
      onUpdate();
    } catch (err) {
      console.error('Failed to delete payment:', err);
      alert('Failed to delete payment. Please try again.');
    }
  };

  const handleAddPayment = async () => {
    if (!newPayment.transaction_date || !newPayment.amount) {
      alert('Please fill in date and amount');
      return;
    }

    try {
      await transactionsAPI.createTransaction({
        transaction_date: newPayment.transaction_date,
        transaction_type: 'PAYMENT',
        category: 'OTHER',
        item_name: newPayment.item_name || 'Cash Payment',
        amount: parseFloat(newPayment.amount),
        notes: newPayment.notes || undefined,
      });
      setShowAddForm(false);
      setNewPayment({
        transaction_date: '',
        amount: '',
        item_name: '',
        notes: '',
      });
      fetchBreakdown();
      onUpdate();
    } catch (err) {
      console.error('Failed to add payment:', err);
      alert('Failed to add payment. Please try again.');
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-IN', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
  };

  const formatCurrency = (num: number | null | undefined) => {
    if (num === null || num === undefined) return '₹0.00';
    return `₹${num.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  if (!isOpen) return null;

  return (
    <div className="breakdown-modal-overlay" onClick={onClose}>
      <div className="breakdown-modal payment-modal" onClick={(e) => e.stopPropagation()}>
        <div className="breakdown-modal-header">
          <h2>Payments Received - {new Date(year, month - 1).toLocaleString('default', { month: 'long', year: 'numeric' })}</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="breakdown-modal-content">
          {canEdit && (
            <div className="add-payment-section">
              {!showAddForm ? (
                <button onClick={() => setShowAddForm(true)} className="add-payment-btn">
                  + Add Cash Payment
                </button>
              ) : (
                <div className="add-payment-form">
                  <h3>Add New Payment</h3>
                  <div className="form-row">
                    <label>Date:</label>
                    <input
                      type="date"
                      value={newPayment.transaction_date}
                      onChange={(e) => setNewPayment({ ...newPayment, transaction_date: e.target.value })}
                    />
                  </div>
                  <div className="form-row">
                    <label>Amount (₹):</label>
                    <input
                      type="number"
                      step="0.01"
                      value={newPayment.amount}
                      onChange={(e) => setNewPayment({ ...newPayment, amount: e.target.value })}
                      placeholder="Enter amount"
                    />
                  </div>
                  <div className="form-row">
                    <label>Description (optional):</label>
                    <input
                      type="text"
                      value={newPayment.item_name}
                      onChange={(e) => setNewPayment({ ...newPayment, item_name: e.target.value })}
                      placeholder="e.g., Cash Payment, Bank Transfer"
                    />
                  </div>
                  <div className="form-row">
                    <label>Notes (optional):</label>
                    <textarea
                      value={newPayment.notes}
                      onChange={(e) => setNewPayment({ ...newPayment, notes: e.target.value })}
                      placeholder="Additional notes"
                      rows={2}
                    />
                  </div>
                  <div className="form-actions">
                    <button onClick={handleAddPayment} className="save-btn">Add Payment</button>
                    <button onClick={() => {
                      setShowAddForm(false);
                      setNewPayment({
                        transaction_date: '',
                        amount: '',
                        item_name: '',
                        notes: '',
                      });
                    }} className="cancel-btn">Cancel</button>
                  </div>
                </div>
              )}
            </div>
          )}

          {loading ? (
            <div className="loading">Loading payments...</div>
          ) : !transactions || transactions.length === 0 ? (
            <div className="no-data">No payments found for this month</div>
          ) : (
            <>
              <div className="summary-info">
                <div className="summary-item">
                  <span className="summary-label">Total Payments:</span>
                  <span className="summary-value">{transactions?.length || 0}</span>
                </div>
                <div className="summary-item">
                  <span className="summary-label">Total Amount:</span>
                  <span className="summary-value">
                    {formatCurrency(transactions.reduce((sum, t) => sum + (t.amount || 0), 0))}
                  </span>
                </div>
              </div>

              <div className="transactions-list">
                {((breakdownData && breakdownData.grouped_by_date) || []).map((group: {
                  date: string;
                  transactions: Transaction[];
                  total_amount: number;
                }) => (
                  <div key={group.date} className="date-group">
                    <div className="date-header">
                      <span className="date-label">{formatDate(group.date)}</span>
                      <span className="date-total">
                        Total: {formatCurrency(group.total_amount)}
                      </span>
                    </div>
                    {group.transactions.map((transaction) => (
                      <div key={transaction.id} className="transaction-item">
                        {editingId === transaction.id ? (
                          <div className="edit-form">
                            <div className="form-row">
                              <label>Description:</label>
                              <input
                                type="text"
                                value={editForm.item_name || ''}
                                onChange={(e) => setEditForm({ ...editForm, item_name: e.target.value })}
                                disabled={!canEdit}
                              />
                            </div>
                            <div className="form-row">
                              <label>Date:</label>
                              <input
                                type="date"
                                value={editForm.transaction_date || ''}
                                onChange={(e) => setEditForm({ ...editForm, transaction_date: e.target.value })}
                                disabled={!canEdit}
                              />
                            </div>
                            <div className="form-row">
                              <label>Amount:</label>
                              <input
                                type="number"
                                step="0.01"
                                value={editForm.amount || ''}
                                onChange={(e) => setEditForm({ ...editForm, amount: parseFloat(e.target.value) || 0 })}
                                disabled={!canEdit}
                              />
                            </div>
                            <div className="form-row">
                              <label>Notes:</label>
                              <textarea
                                value={editForm.notes || ''}
                                onChange={(e) => setEditForm({ ...editForm, notes: e.target.value })}
                                disabled={!canEdit}
                                rows={2}
                              />
                            </div>
                            <div className="form-actions">
                              <button onClick={() => handleSave(transaction.id)} className="save-btn" disabled={!canEdit}>
                                Save
                              </button>
                              <button onClick={handleCancel} className="cancel-btn">Cancel</button>
                            </div>
                          </div>
                        ) : (
                          <div className="transaction-details">
                            <div className="transaction-main">
                              <span className="transaction-item-name">{transaction.item_name || 'Cash Payment'}</span>
                              <span className="transaction-amount">{formatCurrency(transaction.amount)}</span>
                              {transaction.notes && (
                                <span className="transaction-notes">{transaction.notes}</span>
                              )}
                            </div>
                            {canEdit && (
                              <div className="transaction-actions">
                                <button onClick={() => handleEdit(transaction)} className="edit-transaction-btn">
                                  Edit
                                </button>
                                <button onClick={() => handleDelete(transaction.id)} className="delete-transaction-btn">
                                  Delete
                                </button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default PaymentModal;


