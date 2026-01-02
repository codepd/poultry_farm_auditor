import React, { useState, useEffect } from 'react';
import { analyticsAPI, transactionsAPI, Transaction } from '../../services/api';
import './BreakdownModal.css';

interface BreakdownModalProps {
  isOpen: boolean;
  onClose: () => void;
  category: string;
  year: number;
  month: number;
  canEdit: boolean;
  onUpdate: () => void;
}

const BreakdownModal: React.FC<BreakdownModalProps> = ({
  isOpen,
  onClose,
  category,
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
  const [newTransaction, setNewTransaction] = useState({
    transaction_date: '',
    item_name: '',
    quantity: '',
    unit: '',
    rate: '',
    amount: '',
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
  }, [isOpen, category, year, month]);

  const fetchBreakdown = async () => {
    setLoading(true);
    try {
      // Use the new monthly breakdown API
      const breakdown = await analyticsAPI.getMonthlyBreakdown(year, month, category);
      setTransactions(breakdown.transactions || []);
      setBreakdownData({
        average_price: breakdown.average_price || 0,
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
      quantity: transaction.quantity,
      rate: transaction.rate,
      amount: transaction.amount,
      item_name: transaction.item_name,
      transaction_date: transaction.transaction_date,
    });
  };

  const handleSave = async (id: number) => {
    try {
      await transactionsAPI.updateTransaction(id, editForm);
      setEditingId(null);
      fetchBreakdown(); // Refresh breakdown data
      onUpdate(); // Refresh parent component
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
    if (!window.confirm('Are you sure you want to delete this transaction?')) {
      return;
    }
    try {
      await transactionsAPI.deleteTransaction(id);
      fetchBreakdown(); // Refresh breakdown data
      onUpdate(); // Refresh parent component
    } catch (err) {
      console.error('Failed to delete transaction:', err);
      alert('Failed to delete transaction. Please try again.');
    }
  };

  const handleAddTransaction = async () => {
    if (!newTransaction.transaction_date || !newTransaction.amount) {
      alert('Please fill in date and amount');
      return;
    }

    // Determine transaction type and category based on category
    let transactionType = 'PURCHASE';
    let transactionCategory = category;
    
    if (category === 'EGG') {
      transactionType = 'SALE';
    } else if (category === 'FEED' || category === 'MEDICINE') {
      transactionType = 'PURCHASE';
    } else if (category === 'OTHER') {
      transactionType = 'PURCHASE';
    }

    try {
      await transactionsAPI.createTransaction({
        transaction_date: newTransaction.transaction_date,
        transaction_type: transactionType,
        category: transactionCategory,
        item_name: newTransaction.item_name || undefined,
        quantity: newTransaction.quantity ? parseFloat(newTransaction.quantity) : undefined,
        unit: newTransaction.unit || undefined,
        rate: newTransaction.rate ? parseFloat(newTransaction.rate) : undefined,
        amount: parseFloat(newTransaction.amount),
        notes: newTransaction.notes || undefined,
      });
      setShowAddForm(false);
      setNewTransaction({
        transaction_date: '',
        item_name: '',
        quantity: '',
        unit: '',
        rate: '',
        amount: '',
        notes: '',
      });
      fetchBreakdown();
      onUpdate();
    } catch (err) {
      console.error('Failed to add transaction:', err);
      alert('Failed to add transaction. Please try again.');
    }
  };

  // Get default unit based on category
  const getDefaultUnit = () => {
    switch (category) {
      case 'EGG':
        return 'NOS';
      case 'FEED':
        return 'KGS';
      case 'MEDICINE':
        return 'KGS';
      default:
        return '';
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-IN', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
  };

  const formatNumber = (num: number | null | undefined) => {
    if (num === null || num === undefined) return '0';
    return num.toLocaleString('en-IN', { maximumFractionDigits: 2 });
  };

  const formatCurrency = (num: number | null | undefined) => {
    if (num === null || num === undefined) return '₹0.00';
    return `₹${num.toLocaleString('en-IN', { maximumFractionDigits: 2 })}`;
  };

  // Use grouped data from API
  const calculateAveragePrice = () => {
    if (breakdownData?.average_price !== undefined) {
      return breakdownData.average_price;
    }
    if (!transactions || transactions.length === 0) return 0;
    const totalAmount = transactions.reduce((sum, t) => sum + (t.amount || 0), 0);
    const totalQuantity = transactions.reduce((sum, t) => sum + (t.quantity || 0), 0);
    if (totalQuantity === 0) return 0;
    return totalAmount / totalQuantity;
  };

  const categoryTitle = {
    EGG: 'Eggs Sold',
    FEED: 'Feed Purchased',
    MEDICINE: 'Medicine Expenses',
    OTHER: 'Other Expenses',
  }[category] || category;

  if (!isOpen) return null;

  return (
    <div className="breakdown-modal-overlay" onClick={onClose}>
      <div className="breakdown-modal" onClick={(e) => e.stopPropagation()}>
        <div className="breakdown-modal-header">
          <h2>{categoryTitle} - {new Date(year, month - 1).toLocaleString('default', { month: 'long', year: 'numeric' })}</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="breakdown-modal-content">
          {canEdit && (
            <div className="add-transaction-section">
              {!showAddForm ? (
                <button onClick={() => setShowAddForm(true)} className="add-transaction-btn">
                  + Add {category === 'EGG' ? 'Egg Sale' : category === 'FEED' ? 'Feed Purchase' : category === 'MEDICINE' ? 'Medicine' : 'Expense'}
                </button>
              ) : (
                <div className="add-transaction-form">
                  <h3>Add New {categoryTitle}</h3>
                  <div className="form-row">
                    <label>Date:</label>
                    <input
                      type="date"
                      value={newTransaction.transaction_date}
                      onChange={(e) => setNewTransaction({ ...newTransaction, transaction_date: e.target.value })}
                    />
                  </div>
                  <div className="form-row">
                    <label>Item Name:</label>
                    <input
                      type="text"
                      value={newTransaction.item_name}
                      onChange={(e) => setNewTransaction({ ...newTransaction, item_name: e.target.value })}
                      placeholder={category === 'EGG' ? 'e.g., LARGE EGG' : category === 'FEED' ? 'e.g., LAYER MASH' : category === 'MEDICINE' ? 'e.g., Medicine name' : 'Item description'}
                    />
                  </div>
                  {(category === 'EGG' || category === 'FEED' || category === 'MEDICINE') && (
                    <>
                      <div className="form-row">
                        <label>Quantity:</label>
                        <input
                          type="number"
                          step="0.01"
                          value={newTransaction.quantity}
                          onChange={(e) => {
                            const qty = e.target.value;
                            const rate = parseFloat(newTransaction.rate) || 0;
                            setNewTransaction({ 
                              ...newTransaction, 
                              quantity: qty,
                              amount: qty ? (parseFloat(qty) * rate).toString() : ''
                            });
                          }}
                          placeholder="Enter quantity"
                        />
                      </div>
                      <div className="form-row">
                        <label>Unit:</label>
                        <input
                          type="text"
                          value={newTransaction.unit || getDefaultUnit()}
                          onChange={(e) => setNewTransaction({ ...newTransaction, unit: e.target.value })}
                          placeholder={getDefaultUnit()}
                        />
                      </div>
                      <div className="form-row">
                        <label>Rate:</label>
                        <input
                          type="number"
                          step="0.01"
                          value={newTransaction.rate}
                          onChange={(e) => {
                            const rate = e.target.value;
                            const qty = parseFloat(newTransaction.quantity) || 0;
                            setNewTransaction({ 
                              ...newTransaction, 
                              rate: rate,
                              amount: rate ? (qty * parseFloat(rate)).toString() : ''
                            });
                          }}
                          placeholder="Enter rate per unit"
                        />
                      </div>
                    </>
                  )}
                  <div className="form-row">
                    <label>Amount:</label>
                    <input
                      type="number"
                      step="0.01"
                      value={newTransaction.amount}
                      onChange={(e) => setNewTransaction({ ...newTransaction, amount: e.target.value })}
                      placeholder="Enter amount"
                    />
                  </div>
                  <div className="form-row">
                    <label>Notes (optional):</label>
                    <textarea
                      value={newTransaction.notes}
                      onChange={(e) => setNewTransaction({ ...newTransaction, notes: e.target.value })}
                      placeholder="Additional notes"
                      rows={2}
                    />
                  </div>
                  <div className="form-actions">
                    <button onClick={handleAddTransaction} className="save-btn">Add Transaction</button>
                    <button onClick={() => {
                      setShowAddForm(false);
                      setNewTransaction({
                        transaction_date: '',
                        item_name: '',
                        quantity: '',
                        unit: '',
                        rate: '',
                        amount: '',
                        notes: '',
                      });
                    }} className="cancel-btn">Cancel</button>
                  </div>
                </div>
              )}
            </div>
          )}

          {loading ? (
            <div className="loading">Loading transactions...</div>
          ) : !transactions || transactions.length === 0 ? (
            <div className="no-data">No transactions found for this month</div>
          ) : (
            <>
              <div className="summary-info">
                <div className="summary-item">
                  <span className="summary-label">Total Transactions:</span>
                  <span className="summary-value">{transactions?.length || 0}</span>
                </div>
                {category === 'EGG' && (
                  <div className="summary-item">
                    <span className="summary-label">Monthly Average Price:</span>
                    <span className="summary-value">{formatCurrency(calculateAveragePrice())}</span>
                  </div>
                )}
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
                              <label>Item Name:</label>
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
                              <label>Quantity:</label>
                              <input
                                type="number"
                                step="0.01"
                                value={editForm.quantity || ''}
                                onChange={(e) => {
                                  const qty = parseFloat(e.target.value) || 0;
                                  const rate = editForm.rate || 0;
                                  setEditForm({ ...editForm, quantity: qty, amount: qty * rate });
                                }}
                                disabled={!canEdit}
                              />
                            </div>
                            <div className="form-row">
                              <label>Rate:</label>
                              <input
                                type="number"
                                step="0.01"
                                value={editForm.rate || ''}
                                onChange={(e) => {
                                  const rate = parseFloat(e.target.value) || 0;
                                  const qty = editForm.quantity || 0;
                                  setEditForm({ ...editForm, rate: rate, amount: qty * rate });
                                }}
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
                              <span className="transaction-item-name">{transaction.item_name}</span>
                              <span className="transaction-quantity">
                                {formatNumber(transaction.quantity)} {transaction.unit || ''}
                              </span>
                              <span className="transaction-rate">@ {formatCurrency(transaction.rate)}</span>
                              <span className="transaction-amount">{formatCurrency(transaction.amount)}</span>
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

export default BreakdownModal;

