import React, { useState, useEffect } from 'react';
import { HenBatch } from '../../services/api';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { useTenant } from '../../hooks/useTenant';
import { formatDateForTenant, calculateAgeFromDate } from '../../utils/dateUtils';
import './HenAgeDisplay.css';

const HenAgeDisplay: React.FC = () => {
  const { currentTenant } = useAuth();
  const tenantHook = useTenant();
  const tenant = tenantHook.tenant;
  const timezone = tenantHook.timezone;
  const dateFormat = tenantHook.dateFormat;
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

  const getCalculatedAge = (batch: HenBatch) => {
    // Use the initial age stored in the batch (age_weeks, age_days) when it was added
    // and add the elapsed time since date_added
    return calculateAgeFromDate(batch.date_added, timezone, batch.age_weeks, batch.age_days);
  };

  // Categorize hens based on age (in weeks)
  // Standard industry ranges: Chick: 0-6w, Grower: 6-18w, Pre-layer: 18-22w, Layer: 22+w
  // Use tenant configuration if available, otherwise use industry standard
  const getAgeCategory = (weeks: number): 'chick' | 'grower' | 'pre-layer' | 'layer' => {
    // Get age category thresholds from tenant config or use defaults
    const tenantConfig = tenant || null;
    const chickMax = tenantConfig?.age_category_chick_max_weeks ?? 6;
    const growerMax = tenantConfig?.age_category_grower_max_weeks ?? 18;
    const preLayerMax = tenantConfig?.age_category_prelayer_max_weeks ?? 22;
    
    if (weeks < chickMax) return 'chick';
    if (weeks < growerMax) return 'grower';
    if (weeks < preLayerMax) return 'pre-layer';
    return 'layer';
  };

  // Group batches by age category and calculate head counts
  const categoryData = batches.reduce((acc, batch) => {
    const age = getCalculatedAge(batch);
    const category = getAgeCategory(age.weeks);
    const count = batch.current_count || 0;
    
    if (!acc[category]) {
      acc[category] = { count: 0, batches: [] };
    }
    acc[category].count += count;
    acc[category].batches.push(batch);
    
    return acc;
  }, {} as Record<'chick' | 'grower' | 'pre-layer' | 'layer', { count: number; batches: HenBatch[] }>);

  const totalCount = batches.reduce((sum, batch) => sum + (batch.current_count || 0), 0);

  // Get category labels with tenant-specific ranges
  const getCategoryLabel = (category: 'chick' | 'grower' | 'pre-layer' | 'layer'): string => {
    // Get age category thresholds from tenant config or use defaults
    const tenantConfig = tenant || null;
    const chickMax = tenantConfig?.age_category_chick_max_weeks ?? 6;
    const growerMax = tenantConfig?.age_category_grower_max_weeks ?? 18;
    const preLayerMax = tenantConfig?.age_category_prelayer_max_weeks ?? 22;
    
    switch (category) {
      case 'chick':
        return `Chick (0-${chickMax} weeks)`;
      case 'grower':
        return `Grower (${chickMax}-${growerMax} weeks)`;
      case 'pre-layer':
        return `Pre-Layer (${growerMax}-${preLayerMax} weeks)`;
      case 'layer':
        return `Layer (${preLayerMax}+ weeks)`;
    }
  };

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
        <>
          {/* Age Category Summary */}
          <div className="age-categories">
            <h3>Hens by Age Category</h3>
            <div className="category-cards">
              {(['chick', 'grower', 'pre-layer', 'layer'] as const).map((category) => (
                <div key={category} className={`category-card ${category}`}>
                  <div className="category-label">{getCategoryLabel(category)}</div>
                  <div className="category-count">
                    {(categoryData[category]?.count || 0).toLocaleString()}
                  </div>
                  <div className="category-percentage">
                    {totalCount > 0 
                      ? ((categoryData[category]?.count || 0) / totalCount * 100).toFixed(1)
                      : '0'}%
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Individual Batches */}
          <div className="batches-section">
            <h3>Batch Details</h3>
            <div className="batches-grid">
              {batches.map((batch) => {
                const age = getCalculatedAge(batch);
                const category = getAgeCategory(age.weeks);
                return (
                  <div key={batch.id} className={`batch-card ${category}`}>
                    <div className="batch-header">
                      <h3>{batch.batch_name}</h3>
                      <div className="batch-header-right">
                        <span className={`category-badge ${category}`}>
                          {getCategoryLabel(category).split(' ')[0]}
                        </span>
                        <span className="batch-age">{formatAge(age.weeks, age.days)}</span>
                      </div>
                    </div>
                    <div className="batch-details">
                      <div className="detail-item">
                        <span className="label">Current Count:</span>
                        <span className="value">{(batch.current_count ?? 0).toLocaleString()}</span>
                      </div>
                      <div className="detail-item">
                        <span className="label">Initial Count:</span>
                        <span className="value">{(batch.initial_count ?? 0).toLocaleString()}</span>
                      </div>
                      <div className="detail-item">
                        <span className="label">Date Added:</span>
                        <span className="value">
                          {formatDateForTenant(batch.date_added, timezone, dateFormat)}
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default HenAgeDisplay;

