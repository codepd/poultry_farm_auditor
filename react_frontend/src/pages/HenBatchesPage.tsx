import React, { useState, useEffect } from 'react';
import { henBatchesAPI, HenBatch } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { useTenant } from '../hooks/useTenant';
import { formatDateForTenant, calculateAgeFromDate } from '../utils/dateUtils';
import MortalityChart from '../components/HenBatches/MortalityChart';
import './HenBatchesPage.css';

const HenBatchesPage: React.FC = () => {
  const { user } = useAuth();
  const { timezone, dateFormat } = useTenant();
  const [batches, setBatches] = useState<HenBatch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingBatch, setEditingBatch] = useState<HenBatch | null>(null);
  const [showMortalityForm, setShowMortalityForm] = useState(false);
  const [showMortalityChart, setShowMortalityChart] = useState(false);
  const [selectedBatchId, setSelectedBatchId] = useState<number | null>(null);
  const [selectedBatchName, setSelectedBatchName] = useState<string>('');
  const [mortalityBatchId, setMortalityBatchId] = useState<number | null>(null);
  const [mortalityFormData, setMortalityFormData] = useState({
    mortality_date: new Date().toISOString().split('T')[0],
    count: 0,
    reason: '',
    notes: '',
  });
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
      if (editingBatch) {
        await henBatchesAPI.updateHenBatch(editingBatch.id, {
          batch_name: formData.batch_name,
          current_count: formData.current_count,
          age_weeks: formData.age_weeks,
          age_days: formData.age_days,
          notes: formData.notes || undefined,
        });
        setEditingBatch(null);
      } else {
        await henBatchesAPI.createHenBatch(formData);
        setShowAddForm(false);
      }
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
      setError(err.response?.data?.error || `Failed to ${editingBatch ? 'update' : 'create'} batch`);
    }
  };

  const handleEdit = (batch: HenBatch) => {
    setEditingBatch(batch);
    setFormData({
      batch_name: batch.batch_name,
      initial_count: batch.initial_count,
      current_count: batch.current_count,
      age_weeks: batch.age_weeks,
      age_days: batch.age_days,
      date_added: batch.date_added.split('T')[0],
      notes: batch.notes || '',
    });
    setShowAddForm(false);
  };

  const handleDelete = async (batch: HenBatch) => {
    if (!window.confirm(`Are you sure you want to delete batch "${batch.batch_name}"? This action cannot be undone.`)) {
      return;
    }
    try {
      await henBatchesAPI.deleteHenBatch(batch.id);
      fetchBatches();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete batch');
    }
  };

  const handleCancel = () => {
    setEditingBatch(null);
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
  };

  const formatAge = (weeks: number, days: number) => {
    if (weeks === 0 && days === 0) return 'Day 0 (Chicks)';
    if (days === 0) return `${weeks}W`;
    return `${weeks}W ${days}D`;
  };

  const getCalculatedAge = (batch: HenBatch) => {
    // Use the initial age stored in the batch (age_weeks, age_days) when it was added
    // and add the elapsed time since date_added
    return calculateAgeFromDate(batch.date_added, timezone, batch.age_weeks, batch.age_days);
  };

  const handleAddMortality = (batchId: number) => {
    setMortalityBatchId(batchId);
    setMortalityFormData({
      mortality_date: new Date().toISOString().split('T')[0],
      count: 0,
      reason: '',
      notes: '',
    });
    setShowMortalityForm(true);
  };

  const handleMortalitySubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!mortalityBatchId) return;

    try {
      await henBatchesAPI.createMortality({
        batch_id: mortalityBatchId,
        mortality_date: mortalityFormData.mortality_date,
        count: mortalityFormData.count,
        reason: mortalityFormData.reason || undefined,
        notes: mortalityFormData.notes || undefined,
      });
      setShowMortalityForm(false);
      setMortalityBatchId(null);
      setMortalityFormData({
        mortality_date: new Date().toISOString().split('T')[0],
        count: 0,
        reason: '',
        notes: '',
      });
      fetchBatches(); // Refresh batches to show updated counts
      // Refresh chart if it's open for this batch
      if (showMortalityChart && selectedBatchId === mortalityBatchId) {
        // Chart will auto-refresh when batchId changes
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to record mortality');
    }
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

      {(showAddForm || editingBatch) && canAddBatch && (
        <div className="add-batch-form">
          <h2>{editingBatch ? 'Edit Hen Batch' : 'Add New Hen Batch'}</h2>
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
                  onChange={(e) => {
                    const newDate = e.target.value;
                    // Auto-calculate age when date changes (only for new batches, not editing)
                    // For new batches, we start with 0 weeks/days, so calculate elapsed time only
                    if (!editingBatch && newDate) {
                      const age = calculateAgeFromDate(newDate, timezone, 0, 0);
                      setFormData({ 
                        ...formData, 
                        date_added: newDate,
                        age_weeks: age.weeks,
                        age_days: age.days
                      });
                    } else {
                      setFormData({ ...formData, date_added: newDate });
                    }
                  }}
                  required
                  disabled={!!editingBatch}
                />
              </div>
            </div>
            <div className="form-row">
              {!editingBatch && (
                <div className="form-group">
                  <label>Initial Head Count</label>
                  <input
                    type="number"
                    value={formData.initial_count ?? ''}
                    onChange={(e) => {
                      const count = e.target.value === '' ? 0 : (parseInt(e.target.value) ?? 0);
                      setFormData({ ...formData, initial_count: count, current_count: count });
                    }}
                    required
                    min="0"
                  />
                </div>
              )}
              <div className="form-group">
                <label>Current Head Count</label>
                <input
                  type="number"
                  value={formData.current_count ?? ''}
                  onChange={(e) => {
                    const count = e.target.value === '' ? 0 : (parseInt(e.target.value) ?? 0);
                    setFormData({ ...formData, current_count: count });
                  }}
                  required
                  min="0"
                />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Age (Weeks) <span className="hint">(Auto-calculated from date, or set manually for existing batches)</span></label>
                <input
                  type="number"
                  value={formData.age_weeks ?? ''}
                  onChange={(e) => {
                    const value = e.target.value === '' ? 0 : (parseInt(e.target.value) ?? 0);
                    setFormData({ ...formData, age_weeks: value });
                  }}
                  min="0"
                  required
                />
              </div>
              <div className="form-group">
                <label>Age (Days) <span className="hint">(Auto-calculated from date, or set manually for existing batches)</span></label>
                <input
                  type="number"
                  value={formData.age_days ?? ''}
                  onChange={(e) => {
                    const value = e.target.value === '' ? 0 : (parseInt(e.target.value) ?? 0);
                    setFormData({ ...formData, age_days: value });
                  }}
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
              <button type="submit" className="submit-btn">
                {editingBatch ? 'Update Batch' : 'Add Batch'}
              </button>
              <button type="button" onClick={handleCancel} className="cancel-btn">
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="error-message">{error}</div>}

      {/* Mortality Form Modal */}
      {showMortalityForm && mortalityBatchId && (
        <div className="modal-overlay" onClick={() => setShowMortalityForm(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Record Mortality</h2>
              <button className="close-btn" onClick={() => setShowMortalityForm(false)}>
                ×
              </button>
            </div>
            <form onSubmit={handleMortalitySubmit}>
              <div className="form-group">
                <label>Mortality Date</label>
                <input
                  type="date"
                  value={mortalityFormData.mortality_date}
                  onChange={(e) => setMortalityFormData({ ...mortalityFormData, mortality_date: e.target.value })}
                  required
                />
              </div>
              <div className="form-group">
                <label>Count</label>
                <input
                  type="number"
                  value={mortalityFormData.count ?? ''}
                  onChange={(e) => {
                    const count = e.target.value === '' ? 0 : (parseInt(e.target.value) ?? 0);
                    setMortalityFormData({ ...mortalityFormData, count });
                  }}
                  min="1"
                  required
                />
              </div>
              <div className="form-group">
                <label>Reason (Optional)</label>
                <input
                  type="text"
                  value={mortalityFormData.reason}
                  onChange={(e) => setMortalityFormData({ ...mortalityFormData, reason: e.target.value })}
                  placeholder="e.g., Disease, Heat stress, etc."
                />
              </div>
              <div className="form-group">
                <label>Notes (Optional)</label>
                <textarea
                  value={mortalityFormData.notes}
                  onChange={(e) => setMortalityFormData({ ...mortalityFormData, notes: e.target.value })}
                  placeholder="Additional notes..."
                  rows={3}
                />
              </div>
              <div className="form-actions">
                <button type="submit" className="btn-primary">
                  Record Mortality
                </button>
                <button type="button" className="btn-secondary" onClick={() => setShowMortalityForm(false)}>
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Mortality Chart Modal */}
      {showMortalityChart && selectedBatchId && (
        <MortalityChart
          batchId={selectedBatchId}
          batchName={selectedBatchName}
          onClose={() => {
            setShowMortalityChart(false);
            setSelectedBatchId(null);
            setSelectedBatchName('');
          }}
        />
      )}

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
                <div className="batch-header-right">
                  <span className="batch-age">{(() => {
                    const age = getCalculatedAge(batch);
                    return formatAge(age.weeks, age.days);
                  })()}</span>
                  {canAddBatch && (
                    <div className="batch-actions">
                      <button
                        onClick={() => handleEdit(batch)}
                        className="edit-btn"
                        title="Edit batch"
                      >
                        ✏️
                      </button>
                      <button
                        onClick={() => handleDelete(batch)}
                        className="delete-btn"
                        title="Delete batch"
                      >
                        🗑️
                      </button>
                    </div>
                  )}
                </div>
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
                    {formatDateForTenant(batch.date_added, timezone, dateFormat)}
                  </span>
                </div>
                {canAddBatch && (
                  <div className="batch-actions-bottom">
                    <button
                      onClick={() => handleAddMortality(batch.id)}
                      className="mortality-btn"
                      title="Record Mortality"
                    >
                      📉 Record Mortality
                    </button>
                    <button
                      onClick={() => {
                        setSelectedBatchId(batch.id);
                        setSelectedBatchName(batch.batch_name);
                        setShowMortalityChart(true);
                      }}
                      className="chart-btn"
                      title="View Mortality Chart"
                    >
                      📊 View Chart
                    </button>
                  </div>
                )}
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



