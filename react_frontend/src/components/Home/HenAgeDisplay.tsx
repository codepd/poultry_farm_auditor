import React, { useState, useEffect } from 'react';
import { HenBatch } from '../../services/api';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import './HenAgeDisplay.css';

const HenAgeDisplay: React.FC = () => {
  const { currentTenant } = useAuth();
  const [batches, setBatches] = useState<HenBatch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchBatches = async () => {
      try {
        setLoading(true);
        setError(''); // Clear any previous errors
        const response = await api.get<{ success: boolean; data: HenBatch[]; message?: string }>('/hen-batches');
        setBatches(Array.isArray(response.data.data) ? response.data.data : []);
        // Don't show error if message says no batches found - that's normal
        if (response.data.message && !response.data.message.includes('No hen batches found')) {
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
          setBatches([]);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchBatches();
  }, []);

  const formatAge = (weeks: number, days: number) => {
    if (weeks === 0 && days === 0) return 'New';
    if (days === 0) return `${weeks}W`;
    return `${weeks}W ${days}D`;
  };

  const totalCount = batches.reduce((sum, batch) => sum + batch.current_count, 0);

  if (loading) {
    return (
      <div className="hen-age-display">
        <div className="loading">Loading hen information...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="hen-age-display">
        <div className="error">{error}</div>
      </div>
    );
  }

  return (
    <div className="hen-age-display">
      <div className="hen-age-header">
        <h2>Hen Information</h2>
        <div className="total-count">
          Total Head Count: <strong>{totalCount.toLocaleString()}</strong>
        </div>
      </div>
      {batches.length === 0 ? (
        <div className="no-batches">
          <p>No hen batches found. Add a batch to get started.</p>
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
                  <span className="label">Date Added:</span>
                  <span className="value">
                    {new Date(batch.date_added).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default HenAgeDisplay;

