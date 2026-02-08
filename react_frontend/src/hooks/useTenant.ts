import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { tenantsAPI, Tenant } from '../services/api';

/**
 * Hook to fetch and access tenant details including timezone
 */
export function useTenant() {
  const { currentTenant } = useAuth();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!currentTenant?.tenant_id) {
      setLoading(false);
      return;
    }

    const fetchTenant = async () => {
      try {
        setLoading(true);
        setError(null);
        const tenantData = await tenantsAPI.getTenant(currentTenant.tenant_id);
        setTenant(tenantData);
      } catch (err: any) {
        console.error('Error fetching tenant:', err);
        setError(err.response?.data?.error || 'Failed to fetch tenant details');
        // Set default timezone if fetch fails
        setTenant({
          id: currentTenant.tenant_id,
          name: currentTenant.name,
          country_code: 'IND',
          currency: 'INR',
          number_format: 'lakhs',
          date_format: 'DD-MM-YYYY',
          timezone: 'Asia/Kolkata', // Default fallback
          created_at: '',
          updated_at: '',
        });
      } finally {
        setLoading(false);
      }
    };

    fetchTenant();
  }, [currentTenant?.tenant_id, currentTenant?.name]);

  return {
    tenant,
    loading,
    error,
    timezone: tenant?.timezone || 'Asia/Kolkata',
    dateFormat: tenant?.date_format || 'DD-MM-YYYY',
  };
}


