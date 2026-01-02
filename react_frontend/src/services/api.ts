import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle token expiration
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
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

export interface LoginRequest {
  email: string;
  password: string;
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
  total_payments?: number;
  payment_breakdown?: Array<{ type: string; amount: number }>;
  net_profit: number;
  estimated_hens: number;
  egg_percentage: number;
}

export interface YearlySummary {
  year: number;
  total_sales: number;
  total_expense: number;
  net_profit: number;
}

// API Functions
export const authAPI = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const response = await api.post<{ success: boolean; data: LoginResponse }>('/auth/login', data);
    return response.data.data || response.data as any;
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
};

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

export default api;


