import React, { useState, useEffect } from 'react';
import { transactionsAPI, tenantItemsAPI, TenantItem } from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import './MultiCategoryEntry.css';

interface CategoryEntry {
  category: string;
  item_name?: string;
  quantity?: number;
  unit?: string;
  rate?: number;
  amount: number;
}

const MultiCategoryEntry: React.FC = () => {
  const { currentTenant } = useAuth();
  const [transactionDate, setTransactionDate] = useState(
    new Date().toISOString().split('T')[0]
  );
  const [transactionType, setTransactionType] = useState<'SALE' | 'PURCHASE'>('SALE');
  const [entries, setEntries] = useState<CategoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [availableItems, setAvailableItems] = useState<TenantItem[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  
  // Helper function to create default items
  const createDefaultItems = (category: 'EGG' | 'FEED', itemNames: string[]): TenantItem[] => {
    return itemNames.map((name, index) => ({
      id: index + 1,
      tenant_id: currentTenant?.tenant_id || '',
      category,
      item_name: name,
      display_order: index + 1,
      is_active: true,
    }));
  };
  
  // Default items as fallback
  const DEFAULT_EGG_ITEMS = createDefaultItems('EGG', [
    'LARGE EGG',
    'MEDIUM EGG',
    'SMALL EGG',
    'BROKEN EGG',
    'DOUBLE YOLK',
    'DIRT EGG',
    'CORRECT EGG',
    'EXPORT EGG',
  ]);

  const DEFAULT_FEED_ITEMS = createDefaultItems('FEED', [
    'LAYER MASH',
    'GROWER MASH',
    'PRE LAYER MASH',
    'LAYER MASH BULK',
    'GROWER MASH BULK',
    'PRE LAYER MASH BULK',
  ]);

  // Fetch items from API based on transaction type
  useEffect(() => {
    const fetchItems = async () => {
      setItemsLoading(true);
      try {
        const category = transactionType === 'SALE' ? 'EGG' : 'FEED';
        const items = await tenantItemsAPI.getTenantItems(category);
        // Use API items if available, otherwise use defaults
        setAvailableItems(items.length > 0 ? items : (transactionType === 'SALE' ? DEFAULT_EGG_ITEMS : DEFAULT_FEED_ITEMS));
      } catch (err: any) {
        console.error('Failed to fetch tenant items:', err);
        // Use default items as fallback
        setAvailableItems(transactionType === 'SALE' ? DEFAULT_EGG_ITEMS : DEFAULT_FEED_ITEMS);
      } finally {
        setItemsLoading(false);
      }
    };

    fetchItems();
  }, [transactionType]);

  const addEntry = () => {
    setEntries([
      ...entries,
      {
        category: transactionType === 'SALE' ? 'EGG' : 'FEED',
        amount: 0,
      },
    ]);
  };

  const updateEntry = (index: number, field: keyof CategoryEntry, value: any) => {
    const updated = [...entries];
    updated[index] = { ...updated[index], [field]: value };
    
    // Auto-calculate amount if quantity and rate are provided
    if (field === 'quantity' || field === 'rate') {
      const qty = field === 'quantity' ? value : updated[index].quantity || 0;
      const rt = field === 'rate' ? value : updated[index].rate || 0;
      updated[index].amount = qty * rt;
    }
    
    setEntries(updated);
  };

  const removeEntry = (index: number) => {
    setEntries(entries.filter((_, i) => i !== index));
  };

  const handleSubmit = async () => {
    if (entries.length === 0) {
      setError('Please add at least one entry');
      return;
    }

    setLoading(true);
    setError('');

    try {
      // Create transactions for each entry
      const promises = entries.map((entry) =>
        transactionsAPI.createTransaction({
          transaction_date: transactionDate,
          transaction_type: transactionType,
          category: entry.category,
          item_name: entry.item_name,
          quantity: entry.quantity,
          unit: entry.unit,
          rate: entry.rate,
          amount: entry.amount,
        })
      );

      await Promise.all(promises);
      
      // Reset form
      setEntries([]);
      alert('Transactions created successfully!');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create transactions');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="multi-category-entry">
      <h2>Quick Entry</h2>
      
      <div className="form-header">
        <div className="form-group">
          <label>Date</label>
          <input
            type="date"
            value={transactionDate}
            onChange={(e) => setTransactionDate(e.target.value)}
          />
        </div>
        <div className="form-group">
          <label>Type</label>
          <select
            value={transactionType}
            onChange={(e) => {
              setTransactionType(e.target.value as 'SALE' | 'PURCHASE');
              setEntries([]); // Clear entries when type changes
            }}
          >
            <option value="SALE">Sale (Eggs)</option>
            <option value="PURCHASE">Purchase (Feed)</option>
          </select>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="entries-list">
        {entries.map((entry, index) => (
          <div key={index} className="entry-row">
            <div className="form-group">
              <label>Item Name</label>
              {availableItems.length > 0 ? (
                <select
                  value={entry.item_name || ''}
                  onChange={(e) => updateEntry(index, 'item_name', e.target.value)}
                  required
                  disabled={itemsLoading}
                >
                  <option value="">{itemsLoading ? 'Loading items...' : 'Select Item'}</option>
                  {availableItems.map((item) => (
                    <option key={item.id} value={item.item_name}>
                      {item.item_name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  type="text"
                  value={entry.item_name || ''}
                  onChange={(e) => updateEntry(index, 'item_name', e.target.value)}
                  placeholder={transactionType === 'SALE' ? 'e.g., LARGE EGG, MEDIUM EGG' : 'e.g., LAYER MASH, GROWER MASH'}
                  required
                  disabled={itemsLoading}
                />
              )}
            </div>
            <div className="form-group">
              <label>Quantity</label>
              <input
                type="number"
                step="0.001"
                value={entry.quantity || ''}
                onChange={(e) => updateEntry(index, 'quantity', parseFloat(e.target.value) || 0)}
              />
            </div>
            <div className="form-group">
              <label>Unit</label>
              <input
                type="text"
                value={entry.unit || ''}
                onChange={(e) => updateEntry(index, 'unit', e.target.value)}
                placeholder={transactionType === 'SALE' ? 'NOS' : 'KGS'}
              />
            </div>
            <div className="form-group">
              <label>Rate</label>
              <input
                type="number"
                step="0.01"
                value={entry.rate || ''}
                onChange={(e) => updateEntry(index, 'rate', parseFloat(e.target.value) || 0)}
              />
            </div>
            <div className="form-group">
              <label>Amount</label>
              <input
                type="number"
                step="0.01"
                value={entry.amount || 0}
                onChange={(e) => updateEntry(index, 'amount', parseFloat(e.target.value) || 0)}
                readOnly
              />
            </div>
            <button
              type="button"
              onClick={() => removeEntry(index)}
              className="remove-btn"
            >
              Remove
            </button>
          </div>
        ))}
      </div>

      <div className="actions">
        <button type="button" onClick={addEntry} className="add-btn">
          + Add Entry
        </button>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={loading || entries.length === 0}
          className="submit-btn"
        >
          {loading ? 'Submitting...' : 'Submit'}
        </button>
      </div>
    </div>
  );
};

export default MultiCategoryEntry;


