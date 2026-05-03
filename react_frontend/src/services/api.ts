import axios from 'axios';
import { getOrCreateDeviceId } from '../utils/deviceId';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

function attachDeviceId(config: { headers: import('axios').AxiosRequestHeaders }) {
  config.headers.set('X-Device-Id', getOrCreateDeviceId());
}

const api = axios.create({
  baseURL: API_URL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

const refreshClient = axios.create({
  baseURL: API_URL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add token to requests
api.interceptors.request.use((config) => {
  attachDeviceId(config);
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

refreshClient.interceptors.request.use((config) => {
  attachDeviceId(config);
  return config;
});

let isRefreshing = false;
let pendingRefreshResolvers: Array<(token: string | null) => void> = [];

function notifyRefreshSubscribers(token: string | null) {
  pendingRefreshResolvers.forEach((resolver) => resolver(token));
  pendingRefreshResolvers = [];
}

// Handle token expiration with refresh flow
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest: any = error.config || {};
    const isLoginRoute = originalRequest?.url?.includes('/auth/login') || originalRequest?.url?.includes('/auth/verify-otp');
    const isRefreshRoute = originalRequest?.url?.includes('/auth/refresh');

    if (error.response?.status === 401 && !originalRequest?._retry && !isLoginRoute && !isRefreshRoute) {
      originalRequest._retry = true;

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          pendingRefreshResolvers.push((newToken) => {
            if (!newToken) {
              reject(error);
              return;
            }
            originalRequest.headers = originalRequest.headers || {};
            originalRequest.headers.Authorization = `Bearer ${newToken}`;
            resolve(api(originalRequest));
          });
        });
      }

      isRefreshing = true;
      try {
        const refreshResponse = await refreshClient.post<{ success: boolean; data: { token: string } }>('/auth/refresh');
        const newToken = refreshResponse.data?.data?.token;
        if (!newToken) {
          throw new Error('No token in refresh response');
        }
        localStorage.setItem('token', newToken);
        notifyRefreshSubscribers(newToken);
        originalRequest.headers = originalRequest.headers || {};
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } catch (refreshError) {
        notifyRefreshSubscribers(null);
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('currentTenant');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }
    return Promise.reject(error);
  }
);

// Types
export interface User {
  user_id: number;
  email: string;
  full_name: string;
  tenants: TenantInfo[];
}

export interface TenantInfo {
  tenant_id: string;
  name: string;
  role: string;
  is_owner: boolean;
}

export interface Tenant {
  id: string;
  parent_id?: string;
  name: string;
  location?: string;
  country_code: string;
  currency: string;
  number_format: string;
  date_format: string;
  timezone: string;
  capacity?: number;
  age_category_chick_max_weeks?: number;
  age_category_grower_max_weeks?: number;
  age_category_prelayer_max_weeks?: number;
  refresh_ttl_without_remember_hours?: number;
  refresh_ttl_with_remember_days?: number;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
  remember_me?: boolean;
}

export interface LoginResponse {
  token: string;
  user_id: number;
  email: string;
  full_name: string;
  tenants: TenantInfo[];
}

export interface Transaction {
  id: number;
  tenant_id: string;
  transaction_date: string;
  transaction_type: string;
  category: string;
  item_name?: string;
  quantity?: number;
  unit?: string;
  rate?: number;
  amount: number;
  notes?: string;
  payment_date?: string;     // Date when payment was made (defaults to transaction_date)
  period_month?: string;      // Month the payment is for (first day of month)
  period_week?: number;       // Week number within the payment period (optional)
  period_days?: number;       // Number of days the payment covers (optional)
  status: string;
  submitted_by_user_id?: number;
  approved_by_user_id?: number;
  approved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface HenBatch {
  id: number;
  tenant_id: string;
  batch_name: string;
  initial_count: number;
  current_count: number;
  age_weeks: number;
  age_days: number;
  date_added: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface HenBatchSale {
  id: number;
  batch_id: number;
  sale_date: string;
  count: number;
  price_per_hen: number;
  total_amount: number;
  notes?: string;
  recorded_by_user_id?: number;
  created_at: string;
}

export interface Employee {
  id: number;
  tenant_id: string;
  full_name: string;
  phone?: string;
  email?: string;
  address?: string;
  designation?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface EnhancedMonthlySummary {
  year: number;
  month: number;
  total_eggs_sold: number;
  egg_breakdown: Array<{ type: string; quantity: number; amount?: number }>;
  total_egg_price: number;
  feed_purchased_tonne: number;
  feed_breakdown: Array<{ type: string; quantity: number; amount?: number }>;
  total_feed_price: number;
  total_medicines: number;
  medicine_breakdown?: Array<{ type: string; quantity: number; amount: number }>;
  other_expenses?: number;
  other_income?: number;
  hen_sale_income?: number;
  total_discounts?: number;
  total_tds?: number;
  total_payments?: number;
  payment_breakdown?: Array<{ type: string; amount: number }>;
  chick_stage_expense?: {
    enabled: boolean;
    batch_id?: number;
    batch_name?: string;
    monthly_total: number;
    monthly_chick_stage: number;
    monthly_grower_stage: number;
    monthly_prelayer_stage: number;
    total_till_25_weeks: number;
    total_till_25_weeks_chick: number;
    total_till_25_weeks_grower: number;
    total_till_25_weeks_prelayer: number;
  };
  net_profit: number;
  estimated_hens: number;
  actual_head_count?: number;
  using_actual_count?: boolean;
  egg_percentage: number;
}

export interface YearlySummary {
  year: number;
  total_sales: number;
  total_expense: number;
  net_profit: number;
}

export interface SendOTPRequest {
  phone: string;
  country_code?: string;
  tenant_id?: string;
}

export interface SendOTPResponse {
  success: boolean;
  message: string;
  phone: string;
  expires_in: number;
}

export interface VerifyOTPRequest {
  phone: string;
  code: string;
  remember_me?: boolean;
}

export interface CountryCodeInfo {
  country_code: string;
  country_name: string;
}

// API Functions
export const authAPI = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const response = await api.post<{ success: boolean; data: LoginResponse }>('/auth/login', data);
    return response.data.data || response.data as any;
  },
  sendOTP: async (data: SendOTPRequest): Promise<SendOTPResponse> => {
    const response = await api.post<SendOTPResponse>('/auth/send-otp', data);
    return response.data;
  },
  verifyOTP: async (data: VerifyOTPRequest): Promise<LoginResponse> => {
    const response = await api.post<{ success: boolean; data: LoginResponse }>('/auth/verify-otp', data);
    return response.data.data || response.data as any;
  },
  refreshToken: async (): Promise<string> => {
    const response = await refreshClient.post<{ success: boolean; data: { token: string } }>('/auth/refresh');
    return response.data?.data?.token;
  },
  logout: async (): Promise<void> => {
    await refreshClient.post('/auth/logout');
  },
  getCountryCodes: async (): Promise<CountryCodeInfo[]> => {
    const response = await api.get<{ success: boolean; data: CountryCodeInfo[] }>('/auth/country-codes');
    return response.data.data || [];
  },
};

export const transactionsAPI = {
  getTransactions: async (params?: {
    start_date?: string;
    end_date?: string;
    category?: string;
    status?: string;
    transaction_type?: string;
  }) => {
    const response = await api.get<{ success: boolean; data: Transaction[] }>('/transactions', { params });
    return response.data.data;
  },
  getTransaction: async (id: number) => {
    const response = await api.get<{ success: boolean; data: Transaction }>(`/transactions/${id}`);
    return response.data.data;
  },
  createTransaction: async (data: Partial<Transaction>) => {
    const response = await api.post<{ success: boolean; data: Transaction }>('/transactions', data);
    return response.data.data;
  },
  updateTransaction: async (id: number, data: Partial<Transaction>) => {
    const response = await api.put<{ success: boolean; data: Transaction }>(`/transactions/${id}`, data);
    return response.data.data;
  },
  deleteTransaction: async (id: number) => {
    await api.delete(`/transactions/${id}`);
  },
  submitTransaction: async (id: number) => {
    const response = await api.post<{ success: boolean; data: Transaction }>(`/transactions/${id}/submit`);
    return response.data.data;
  },
  approveTransaction: async (id: number) => {
    const response = await api.post<{ success: boolean; data: Transaction }>(`/transactions/${id}/approve`);
    return response.data.data;
  },
  rejectTransaction: async (id: number) => {
    const response = await api.post<{ success: boolean; data: Transaction }>(`/transactions/${id}/reject`);
    return response.data.data;
  },
};

export const henBatchesAPI = {
  getHenBatches: async () => {
    const response = await api.get<{ success: boolean; data: HenBatch[] | null }>('/hen-batches');
    // Ensure we always return an array, never null or undefined
    return response.data.data || [];
  },
  getHenBatch: async (id: number) => {
    const response = await api.get<{ success: boolean; data: HenBatch }>(`/hen-batches/${id}`);
    return response.data.data;
  },
  createHenBatch: async (data: Partial<HenBatch>) => {
    const response = await api.post<{ success: boolean; data: HenBatch }>('/hen-batches', data);
    return response.data.data;
  },
  updateHenBatch: async (id: number, data: Partial<HenBatch>) => {
    const response = await api.put<{ success: boolean; data: HenBatch }>(`/hen-batches/${id}`, data);
    return response.data.data;
  },
  deleteHenBatch: async (id: number) => {
    const response = await api.delete<{ success: boolean; message: string }>(`/hen-batches/${id}`);
    return response.data;
  },
  createMortality: async (data: {
    batch_id: number;
    mortality_date: string;
    count: number;
    reason?: string;
    notes?: string;
  }) => {
    const response = await api.post<{ success: boolean; message: string }>('/hen-batches/mortality', data);
    return response.data;
  },
  getMortalityHistory: async (batchId: number) => {
    const response = await api.get<{ success: boolean; data: MortalityRecord[] }>(`/hen-batches/${batchId}/mortality`);
    return response.data.data;
  },
  createSale: async (data: {
    batch_id: number;
    sale_date: string;
    count: number;
    price_per_hen: number;
    notes?: string;
  }) => {
    const response = await api.post<{ success: boolean; message: string; data: HenBatchSale }>('/hen-batches/sales', data);
    return response.data;
  },
  getSalesHistory: async (batchId: number) => {
    const response = await api.get<{ success: boolean; data: HenBatchSale[] }>(`/hen-batches/${batchId}/sales`);
    return response.data.data || [];
  },
};

export interface MortalityRecord {
  id: number;
  batch_id: number;
  mortality_date: string;
  count: number;
  reason?: string;
  notes?: string;
  recorded_by_user_id?: number;
  created_at: string;
}

export const employeesAPI = {
  getEmployees: async (params?: { is_active?: boolean }) => {
    const response = await api.get<{ success: boolean; data: Employee[] }>('/employees', { params });
    return response.data.data;
  },
  getEmployee: async (id: number) => {
    const response = await api.get<{ success: boolean; data: Employee }>(`/employees/${id}`);
    return response.data.data;
  },
  createEmployee: async (data: Partial<Employee>) => {
    const response = await api.post<{ success: boolean; data: Employee }>('/employees', data);
    return response.data.data;
  },
  updateEmployee: async (id: number, data: Partial<Employee>) => {
    const response = await api.put<{ success: boolean; data: Employee }>(`/employees/${id}`, data);
    return response.data.data;
  },
};

export interface MonthlyBreakdown {
  category: string;
  year: number;
  month: number;
  transactions: Transaction[];
  grouped_by_date: Array<{
    date: string;
    transactions: Transaction[];
    total_amount: number;
  }>;
  average_price: number;
  total_count: number;
}

export const analyticsAPI = {
  getEnhancedMonthlySummary: async (year: number, month: number) => {
    const response = await api.get<{ success: boolean; data: EnhancedMonthlySummary[] }>(
      '/analytics/enhanced-monthly-summary',
      { params: { year, month } }
    );
    return response.data.data[0];
  },
  getAllYearsSummary: async () => {
    const response = await api.get<{ success: boolean; data: YearlySummary[] }>('/analytics/all-years-summary');
    return response.data.data;
  },
  getMonthlyBreakdown: async (year: number, month: number, category: string) => {
    const response = await api.get<{ success: boolean; data: MonthlyBreakdown }>(
      '/analytics/monthly-breakdown',
      { params: { year, month, category } }
    );
    return response.data.data;
  },
};

export interface TenantItem {
  id: number;
  tenant_id: string;
  category: string;
  item_name: string;
  display_order: number;
  is_active: boolean;
}

export const tenantItemsAPI = {
  getTenantItems: async (category?: string) => {
    const params: any = {};
    if (category) {
      params.category = category;
    }
    const response = await api.get<{ success: boolean; data: TenantItem[] }>('/tenants/items', { params });
    return response.data.data;
  },
};

// User management types
export interface TenantUser {
  id: number;
  email: string;
  phone: string;
  full_name: string;
  is_active: boolean;
  role: string;
  is_owner: boolean;
  created_at: string;
  updated_at: string;
}

export interface InvitationInfo {
  id: number;
  tenant_id: string;
  invited_by_user_id: number;
  email: { String: string; Valid: boolean } | null;
  phone: { String: string; Valid: boolean } | null;
  role: string;
  token: string;
  expires_at: string;
  accepted_at?: string | null;
  created_at: string;
}

export interface InviteRequest {
  email?: string;
  phone?: string;
  role: string;
}

export const usersAPI = {
  getUsers: async (): Promise<TenantUser[]> => {
    const response = await api.get<{ success: boolean; data: TenantUser[] }>('/users');
    return response.data.data || [];
  },
  inviteUser: async (data: InviteRequest) => {
    const response = await api.post<{ success: boolean; message: string; data: any }>('/users/invite', data);
    return response.data;
  },
  getInvitations: async (): Promise<InvitationInfo[]> => {
    const response = await api.get<{ success: boolean; data: InvitationInfo[] }>('/users/invitations');
    return response.data.data || [];
  },
  updateProfile: async (data: { full_name: string }) => {
    const response = await api.put<{ success: boolean; message: string; data: any }>('/users/profile', data);
    return response.data;
  },
  changePassword: async (data: { current_password?: string; new_password: string; logout_other_devices?: boolean }) => {
    const response = await api.post<{ success: boolean; message: string; data: { revoked_sessions: number } }>('/users/change-password', data);
    return response.data;
  },
  logoutOtherDevices: async () => {
    const response = await api.post<{ success: boolean; message: string; data: { revoked_sessions: number } }>('/users/logout-other-devices');
    return response.data;
  },
};

export const tenantsAPI = {
  getTenant: async (id: string) => {
    const response = await api.get<{ success: boolean; data: Tenant }>(`/tenants/${id}`);
    return response.data.data;
  },
  getTenants: async (tenantId?: string) => {
    const params: any = {};
    if (tenantId) {
      params.tenant_id = tenantId;
    }
    const response = await api.get<{ success: boolean; data: Tenant[] }>('/tenants', { params });
    return response.data.data;
  },
  updateTenant: async (id: string, data: Partial<Tenant>) => {
    const response = await api.put<{ success: boolean; data: Tenant }>(`/tenants/${id}`, data);
    return response.data.data;
  },
};

export default api;


