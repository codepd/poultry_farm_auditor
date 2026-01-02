import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import api from '../services/api';
import PriceLineChart from '../components/PriceHistory/PriceLineChart';
import './PriceHistoryPage.css';

interface PriceHistory {
  id: number;
  tenant_id: string;
  price_date: string;
  price_type: string;
  item_name: string;
  price: number;
  created_at: string;
}

type ViewMode = 'chart' | 'table';

const PriceHistoryPage: React.FC = () => {
  const { currentTenant } = useAuth();
  const [prices, setPrices] = useState<PriceHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>('chart');
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingPrice, setEditingPrice] = useState<PriceHistory | null>(null);
  const [formData, setFormData] = useState({
    price_date: new Date().toISOString().split('T')[0],
    price_type: 'EGG',
    item_name: '',
    price: 0,
  });
  const [selectedYear, setSelectedYear] = useState<number | null>(null);
  const [yearOnYearMode, setYearOnYearMode] = useState(false);
  const [selectedYears, setSelectedYears] = useState<number[]>([]);
  const [selectedItems, setSelectedItems] = useState<string[]>([]); // Filter by item names

  // Check if user can edit (OWNER or MANAGER role)
  const canEdit = currentTenant && ['OWNER', 'MANAGER'].includes(currentTenant.role);

  // Get available years from prices
  const availableYears = React.useMemo(() => {
    const years = Array.from(
      new Set(prices.map(p => new Date(p.price_date).getFullYear()))
    ).sort((a, b) => b - a); // Most recent first
    return years;
  }, [prices]);

  // Set default year to current year or most recent available year
  React.useEffect(() => {
    if (availableYears.length > 0 && selectedYear === null) {
      const currentYear = new Date().getFullYear();
      setSelectedYear(availableYears.includes(currentYear) ? currentYear : availableYears[0]);
    }
  }, [availableYears, selectedYear]);

  // Initialize year-on-year selection with last 2 years
  React.useEffect(() => {
    if (yearOnYearMode && availableYears.length >= 2 && selectedYears.length === 0) {
      setSelectedYears([availableYears[0], availableYears[1]]);
    } else if (!yearOnYearMode) {
      setSelectedYears([]);
    }
  }, [yearOnYearMode, availableYears, selectedYears.length]);

  useEffect(() => {
    fetchPrices();
  }, []);

  const fetchPrices = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await api.get<{ success: boolean; data: PriceHistory[] | null }>('/prices');
      setPrices(Array.isArray(response.data.data) ? response.data.data : []);
    } catch (err: any) {
      setError('Failed to load price history');
      console.error(err);
      setPrices([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingPrice) {
        await api.put(`/prices/${editingPrice.id}`, formData);
      } else {
        await api.post('/prices', formData);
      }
      setShowAddForm(false);
      setEditingPrice(null);
      setFormData({
        price_date: new Date().toISOString().split('T')[0],
        price_type: 'EGG',
        item_name: '',
        price: 0,
      });
      fetchPrices();
    } catch (err: any) {
      setError(err.response?.data?.error || `Failed to ${editingPrice ? 'update' : 'create'} price entry`);
    }
  };

  const handleEdit = (price: PriceHistory) => {
    setEditingPrice(price);
    setFormData({
      price_date: price.price_date,
      price_type: price.price_type,
      item_name: price.item_name,
      price: price.price,
    });
    setShowAddForm(true);
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this price entry?')) {
      return;
    }
    try {
      await api.delete(`/prices/${id}`);
      fetchPrices();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete price entry');
    }
  };

  const handleChartPointClick = (price: PriceHistory) => {
    if (canEdit && price.id !== 0) {
      // Only allow editing of actual price entries, not calculated monthly averages
      handleEdit(price);
    }
  };

  // Group prices by type and item
  const groupedPrices = (prices || []).reduce((acc, price) => {
    if (!price) return acc;
    const key = `${price.price_type}-${price.item_name}`;
    if (!acc[key]) {
      acc[key] = [];
    }
    acc[key].push(price);
    return acc;
  }, {} as Record<string, PriceHistory[]>);

  // Get unique item names for filtering
  const eggItemNames = React.useMemo(() => {
    const items = Array.from(new Set(prices.filter(p => p.price_type === 'EGG').map(p => p.item_name)))
      .sort((a, b) => {
        const order = ['LARGE EGG', 'MEDIUM EGG', 'SMALL EGG'];
        const aIndex = order.findIndex(o => a.includes(o));
        const bIndex = order.findIndex(o => b.includes(o));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
        return a.localeCompare(b);
      });
    return items;
  }, [prices]);

  const feedItemNames = React.useMemo(() => {
    const items = Array.from(new Set(prices.filter(p => p.price_type === 'FEED').map(p => p.item_name)))
      .sort((a, b) => {
        // Prioritize common feed types
        const order = ['LAYER MASH', 'PRE-LAYER MASH', 'PRE LAYER MASH', 'PLM', 'GROWER MASH', 'CHICK MASH'];
        const aIndex = order.findIndex(o => a.toUpperCase().includes(o.toUpperCase()));
        const bIndex = order.findIndex(o => b.toUpperCase().includes(o.toUpperCase()));
        if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
        if (aIndex !== -1) return -1;
        if (bIndex !== -1) return 1;
        return a.localeCompare(b);
      });
    return items;
  }, [prices]);

  // Initialize selected items to all items if empty
  React.useEffect(() => {
    if (selectedItems.length === 0 && eggItemNames.length > 0) {
      setSelectedItems([...eggItemNames, ...feedItemNames]);
    }
  }, [eggItemNames, feedItemNames, selectedItems.length]);

  // Separate Egg and Feed prices, filtered by selected items
  const eggPrices = prices.filter(p => 
    p.price_type === 'EGG' && 
    (selectedItems.length === 0 || selectedItems.includes(p.item_name))
  );
  const feedPrices = prices.filter(p => 
    p.price_type === 'FEED' && 
    (selectedItems.length === 0 || selectedItems.includes(p.item_name))
  );

  return (
    <div className="price-history-page">
      <div className="page-header">
        <h1>Price History</h1>
        <div className="header-actions">
          <div className="view-toggle">
            <button
              className={viewMode === 'chart' ? 'active' : ''}
              onClick={() => setViewMode('chart')}
            >
              📊 Chart View
            </button>
            <button
              className={viewMode === 'table' ? 'active' : ''}
              onClick={() => setViewMode('table')}
            >
              📋 Table View
            </button>
          </div>
          {canEdit && (
            <button onClick={() => {
              setShowAddForm(!showAddForm);
              setEditingPrice(null);
              setFormData({
                price_date: new Date().toISOString().split('T')[0],
                price_type: 'EGG',
                item_name: '',
                price: 0,
              });
            }} className="add-btn">
              {showAddForm ? 'Cancel' : '+ Add Price'}
            </button>
          )}
        </div>
      </div>

      {showAddForm && (
        <div className="add-price-form">
          <h2>{editingPrice ? 'Edit Price' : 'Add New Price'}</h2>
          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label>Date</label>
                <input
                  type="date"
                  value={formData.price_date}
                  onChange={(e) => setFormData({ ...formData, price_date: e.target.value })}
                  required
                />
              </div>
              <div className="form-group">
                <label>Type</label>
                <select
                  value={formData.price_type}
                  onChange={(e) => setFormData({ ...formData, price_type: e.target.value })}
                  required
                >
                  <option value="EGG">Egg</option>
                  <option value="FEED">Feed</option>
                </select>
              </div>
              <div className="form-group">
                <label>Item Name</label>
                <input
                  type="text"
                  value={formData.item_name}
                  onChange={(e) => setFormData({ ...formData, item_name: e.target.value })}
                  placeholder="e.g., LARGE EGG, LAYER MASH"
                  required
                />
              </div>
              <div className="form-group">
                <label>Price (₹)</label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.price || ''}
                  onChange={(e) => setFormData({ ...formData, price: parseFloat(e.target.value) || 0 })}
                  required
                />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" className="submit-btn">
                {editingPrice ? 'Update Price' : 'Add Price'}
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowAddForm(false);
                  setEditingPrice(null);
                }}
                className="cancel-btn"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading">Loading price history...</div>
      ) : prices.length === 0 ? (
        <div className="no-data">No price history found</div>
      ) : viewMode === 'chart' ? (
        <>
          {/* Year Selection Controls */}
          <div className="year-controls">
            <div className="year-control-group">
              <label>
                <input
                  type="checkbox"
                  checked={yearOnYearMode}
                  onChange={(e) => setYearOnYearMode(e.target.checked)}
                />
                <span>Year-on-Year Comparison</span>
              </label>
            </div>
            {yearOnYearMode ? (
              <>
                <div className="year-control-group">
                  <label>Select Years to Compare:</label>
                  <div className="year-checkboxes">
                    {availableYears.map(year => (
                      <label key={year} className="year-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedYears.includes(year)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedYears([...selectedYears, year].sort((a, b) => b - a));
                            } else {
                              setSelectedYears(selectedYears.filter(y => y !== year));
                            }
                          }}
                        />
                        <span>{year}</span>
                      </label>
                    ))}
                  </div>
                </div>
                <div className="year-control-group">
                  <label>Select Egg Items:</label>
                  <div className="item-checkboxes">
                    {eggItemNames.map(item => (
                      <label key={item} className="item-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedItems.includes(item)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedItems([...selectedItems, item]);
                            } else {
                              setSelectedItems(selectedItems.filter(i => i !== item));
                            }
                          }}
                        />
                        <span>{item}</span>
                      </label>
                    ))}
                  </div>
                </div>
                <div className="year-control-group">
                  <label>Select Feed Items:</label>
                  <div className="item-checkboxes">
                    {feedItemNames.map(item => (
                      <label key={item} className="item-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedItems.includes(item)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedItems([...selectedItems, item]);
                            } else {
                              setSelectedItems(selectedItems.filter(i => i !== item));
                            }
                          }}
                        />
                        <span>{item}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </>
            ) : (
              <>
                <div className="year-control-group">
                  <label>Select Year:</label>
                  <select
                    value={selectedYear || ''}
                    onChange={(e) => setSelectedYear(parseInt(e.target.value))}
                    className="year-select"
                  >
                    <option value="">All Years</option>
                    {availableYears.map(year => (
                      <option key={year} value={year}>{year}</option>
                    ))}
                  </select>
                </div>
                <div className="year-control-group">
                  <label>Filter Egg Items:</label>
                  <div className="item-checkboxes">
                    {eggItemNames.map(item => (
                      <label key={item} className="item-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedItems.includes(item)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedItems([...selectedItems, item]);
                            } else {
                              setSelectedItems(selectedItems.filter(i => i !== item));
                            }
                          }}
                        />
                        <span>{item}</span>
                      </label>
                    ))}
                  </div>
                </div>
                <div className="year-control-group">
                  <label>Filter Feed Items:</label>
                  <div className="item-checkboxes">
                    {feedItemNames.map(item => (
                      <label key={item} className="item-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedItems.includes(item)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedItems([...selectedItems, item]);
                            } else {
                              setSelectedItems(selectedItems.filter(i => i !== item));
                            }
                          }}
                        />
                        <span>{item}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>

          <div className="chart-panes">
            <div className="chart-pane">
              <div className="pane-header">
                <h2>🥚 Egg Prices</h2>
                <p className="pane-subtitle">
                  {yearOnYearMode 
                    ? `Year-on-year comparison: ${selectedYears.join(' vs ')}`
                    : selectedYear 
                      ? `Price trends for ${selectedYear}`
                      : 'Price trends over time for all egg types'
                  }
                </p>
              </div>
              <PriceLineChart
                prices={eggPrices}
                priceType="EGG"
                selectedYears={yearOnYearMode ? selectedYears : (selectedYear ? [selectedYear] : undefined)}
                onPointClick={handleChartPointClick}
              />
            </div>
            <div className="chart-pane">
              <div className="pane-header">
                <h2>🌾 Feed Prices</h2>
                <p className="pane-subtitle">
                  {yearOnYearMode 
                    ? `Year-on-year comparison: ${selectedYears.join(' vs ')}`
                    : selectedYear 
                      ? `Price trends for ${selectedYear}`
                      : 'Price trends over time for all feed types'
                  }
                </p>
              </div>
              <PriceLineChart
                prices={feedPrices}
                priceType="FEED"
                selectedYears={yearOnYearMode ? selectedYears : (selectedYear ? [selectedYear] : undefined)}
                onPointClick={handleChartPointClick}
              />
            </div>
          </div>
        </>
      ) : (
        <div className="table-panes">
          <div className="table-pane">
            <div className="pane-header">
              <h2>🥚 Egg Prices</h2>
            </div>
            <div className="price-groups">
              {Object.entries(groupedPrices)
                .filter(([key]) => key.startsWith('EGG-'))
                .map(([key, items]) => {
                  const [, itemName] = key.split('-');
                  return (
                    <div key={key} className="price-group">
                      <h3>
                        {itemName}
                        {items.some(p => p.id === 0) && (
                          <span className="monthly-avg-badge">(Monthly Averages)</span>
                        )}
                      </h3>
                      <table>
                        <thead>
                          <tr>
                            <th>Date</th>
                            <th>Price (₹)</th>
                            <th>Created At</th>
                            {canEdit && <th>Actions</th>}
                          </tr>
                        </thead>
                        <tbody>
                          {items
                            .sort((a, b) => new Date(b.price_date).getTime() - new Date(a.price_date).getTime())
                            .map((price, idx) => {
                              const dateDisplay = price.price_date.endsWith('-01')
                                ? new Date(price.price_date).toLocaleDateString('en-IN', { month: 'long', year: 'numeric' })
                                : new Date(price.price_date).toLocaleDateString();
                              const rowKey = price.id || `calc-${idx}-${price.price_date}`;
                              const isCalculated = price.id === 0;

                              return (
                                <tr key={rowKey}>
                                  <td>{dateDisplay}</td>
                                  <td>₹{price.price.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</td>
                                  <td>
                                    {isCalculated ? (
                                      <span className="calculated-badge">Monthly Average</span>
                                    ) : (
                                      new Date(price.created_at).toLocaleString()
                                    )}
                                  </td>
                                  {canEdit && (
                                    <td>
                                      {!isCalculated && (
                                        <div className="action-buttons">
                                          <button
                                            onClick={() => handleEdit(price)}
                                            className="edit-btn"
                                            title="Edit"
                                          >
                                            ✏️
                                          </button>
                                          <button
                                            onClick={() => handleDelete(price.id)}
                                            className="delete-btn"
                                            title="Delete"
                                          >
                                            🗑️
                                          </button>
                                        </div>
                                      )}
                                    </td>
                                  )}
                                </tr>
                              );
                            })}
                        </tbody>
                      </table>
                    </div>
                  );
                })}
            </div>
          </div>
          <div className="table-pane">
            <div className="pane-header">
              <h2>🌾 Feed Prices</h2>
            </div>
            <div className="price-groups">
              {Object.entries(groupedPrices)
                .filter(([key]) => key.startsWith('FEED-'))
                .map(([key, items]) => {
                  const [, itemName] = key.split('-');
                  return (
                    <div key={key} className="price-group">
                      <h3>{itemName}</h3>
                      <table>
                        <thead>
                          <tr>
                            <th>Date</th>
                            <th>Price (₹)</th>
                            <th>Created At</th>
                            {canEdit && <th>Actions</th>}
                          </tr>
                        </thead>
                        <tbody>
                          {items
                            .sort((a, b) => new Date(b.price_date).getTime() - new Date(a.price_date).getTime())
                            .map((price) => (
                              <tr key={price.id}>
                                <td>{new Date(price.price_date).toLocaleDateString()}</td>
                                <td>₹{price.price.toLocaleString('en-IN', { minimumFractionDigits: 2 })}</td>
                                <td>{new Date(price.created_at).toLocaleString()}</td>
                                {canEdit && (
                                  <td>
                                    <div className="action-buttons">
                                      <button
                                        onClick={() => handleEdit(price)}
                                        className="edit-btn"
                                        title="Edit"
                                      >
                                        ✏️
                                      </button>
                                      <button
                                        onClick={() => handleDelete(price.id)}
                                        className="delete-btn"
                                        title="Delete"
                                      >
                                        🗑️
                                      </button>
                                    </div>
                                  </td>
                                )}
                              </tr>
                            ))}
                        </tbody>
                      </table>
                    </div>
                  );
                })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default PriceHistoryPage;
