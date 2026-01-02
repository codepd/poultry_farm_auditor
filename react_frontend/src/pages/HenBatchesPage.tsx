import React, { useState, useEffect } from 'react';
import { henBatchesAPI, HenBatch } from '../services/api';
import { useAuth } from '../context/AuthContext';
import './HenBatchesPage.css';

const HenBatchesPage: React.FC = () => {
  const { currentTenant, user } = useAuth();
  const [batches, setBatches] = useState<HenBatch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [formData, setFormData] = useState({
    batch_name: '',
    initial_count: 0,
    current_count: 0,
    age_weeks: 0,
    age_days: 0,
    date_added: new Date().toISOString().split('T')[0],
    notes: '',
  });

  // Check if user has permission to add batches (OWNER, CO_OWNER, ADMIN, MANAGER)
  const canAddBatch = user && ['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(user.tenants[0]?.role || '');

  useEffect(() => {
    fetchBatches();
  }, []);

  const fetchBatches = async () => {
    try {
      setLoading(true);
      setError('');
      const data = await henBatchesAPI.getHenBatches();
      // Ensure data is always an array, never null or undefined
      setBatches(Array.isArray(data) ? data : []);
    } catch (err: any) {
      setError('Failed to load hen batches');
      console.error(err);
      setBatches([]); // Set to empty array on error
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await henBatchesAPI.createHenBatch(formData);
      setShowAddForm(false);
      setFormData({
        batch_name: '',
        initial_count: 0,
        current_count: 0,
        age_weeks: 0,
        age_days: 0,
        date_added: new Date().toISOString().split('T')[0],
        notes: '',
      });
      fetchBatches();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create batch');
    }
  };

  const formatAge = (weeks: number, days: number) => {
    if (weeks === 0 && days === 0) return 'New';
    if (days === 0) return `${weeks}W`;
    return `${weeks}W ${days}D`;
  };

  // Safely calculate total count, handling null/undefined batches
  const totalCount = (batches || []).reduce((sum, batch) => sum + (batch?.current_count || 0), 0);

  return (
    <div className="hen-batches-page">
      <div className="page-header">
        <div>
          <h1>Hen Batches</h1>
          <p className="page-subtitle">Total Head Count: <strong>{totalCount.toLocaleString()}</strong></p>
        </div>
        {canAddBatch && (
          <button onClick={() => setShowAddForm(!showAddForm)} className="add-btn">
            {showAddForm ? 'Cancel' : '+ Add Batch'}
          </button>
        )}
      </div>

      {!canAddBatch && (
        <div className="info-message">
          You don't have permission to add hen batches. Contact an owner or administrator.
        </div>
      )}

      {showAddForm && canAddBatch && (
        <div className="add-batch-form">
          <h2>Add New Hen Batch</h2>
          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label>Batch Name</label>
                <input
                  type="text"
                  value={formData.batch_name}
                  onChange={(e) => setFormData({ ...formData, batch_name: e.target.value })}
                  placeholder="e.g., Batch 1, Batch A"
                  required
                />
              </div>
              <div className="form-group">
                <label>Date Added</label>
                <input
                  type="date"
                  value={formData.date_added}
                  onChange={(e) => setFormData({ ...formData, date_added: e.target.value })}
                  required
                />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Initial Head Count</label>
                <input
                  type="number"
                  value={formData.initial_count || ''}
                  onChange={(e) => {
                    const count = parseInt(e.target.value) || 0;
                    setFormData({ ...formData, initial_count: count, current_count: count });
                  }}
                  required
                  min="0"
                />
              </div>
              <div className="form-group">
                <label>Current Head Count</label>
                <input
                  type="number"
                  value={formData.current_count || ''}
                  onChange={(e) => setFormData({ ...formData, current_count: parseInt(e.target.value) || 0 })}
                  required
                  min="0"
                />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Age (Weeks)</label>
                <input
                  type="number"
                  value={formData.age_weeks || ''}
                  onChange={(e) => setFormData({ ...formData, age_weeks: parseInt(e.target.value) || 0 })}
                  min="0"
                  required
                />
              </div>
              <div className="form-group">
                <label>Age (Days)</label>
                <input
                  type="number"
                  value={formData.age_days || ''}
                  onChange={(e) => setFormData({ ...formData, age_days: parseInt(e.target.value) || 0 })}
                  min="0"
                  max="6"
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
                placeholder="Additional details about this batch..."
              />
            </div>
            <div className="form-actions">
              <button type="submit" className="submit-btn">Add Batch</button>
              <button type="button" onClick={() => setShowAddForm(false)} className="cancel-btn">
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="error-message">{error}</div>}

      {loading ? (
        <div className="loading">Loading hen batches...</div>
      ) : batches.length === 0 ? (
        <div className="no-data">
          {canAddBatch ? 'No hen batches found. Add your first batch above.' : 'No hen batches found.'}
        </div>
      ) : (
        <div className="batches-grid">
          {batches.map((batch) => (
            <div key={batch.id} className="batch-card">
              <div className="batch-header">
                <h3>{batch.batch_name}</h3>
                <span className="batch-age">{formatAge(batch.age_weeks, batch.age_days)}</span>
              </div>
              <div className="batch-details">
                <div className="detail-item">
                  <span className="label">Current Count:</span>
                  <span className="value">{batch.current_count.toLocaleString()}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Initial Count:</span>
                  <span className="value">{batch.initial_count.toLocaleString()}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Mortality:</span>
                  <span className="value mortality">
                    {(batch.initial_count - batch.current_count).toLocaleString()}
                  </span>
                </div>
                <div className="detail-item">
                  <span className="label">Date Added:</span>
                  <span className="value">
                    {new Date(batch.date_added).toLocaleDateString()}
                  </span>
                </div>
                {batch.notes && (
                  <div className="batch-notes">
                    <span className="label">Notes:</span>
                    <span className="value">{batch.notes}</span>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default HenBatchesPage;



