import React, { createContext, useContext, useState, ReactNode } from 'react';
import { authAPI, LoginResponse, TenantInfo } from '../services/api';

interface AuthContextType {
  user: LoginResponse | null;
  currentTenant: TenantInfo | null;
  login: (email: string, password: string) => Promise<void>;
  loginWithOTP: (phone: string, code: string) => Promise<void>;
  logout: () => void;
  switchTenant: (tenantId: string) => void;
  updateUser: (patch: Partial<LoginResponse>) => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function loadStoredUser(): LoginResponse | null {
  try {
    const stored = localStorage.getItem('user');
    const token = localStorage.getItem('token');
    if (stored && token) return JSON.parse(stored);
  } catch {
    localStorage.removeItem('user');
    localStorage.removeItem('token');
    localStorage.removeItem('currentTenant');
  }
  return null;
}

function loadStoredTenant(userData: LoginResponse | null): TenantInfo | null {
  if (!userData?.tenants) return null;
  const storedTenant = localStorage.getItem('currentTenant');
  if (storedTenant) {
    return userData.tenants.find(t => t.tenant_id === storedTenant) || userData.tenants[0] || null;
  }
  return userData.tenants[0] || null;
}

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<LoginResponse | null>(() => loadStoredUser());
  const [currentTenant, setCurrentTenant] = useState<TenantInfo | null>(() => loadStoredTenant(loadStoredUser()));

  const handleLoginResponse = (response: LoginResponse) => {
    localStorage.setItem('token', response.token);
    localStorage.setItem('user', JSON.stringify(response));
    
    if (response.tenants && response.tenants.length > 0) {
      setCurrentTenant(response.tenants[0]);
      localStorage.setItem('currentTenant', response.tenants[0].tenant_id);
    }
    
    setUser(response);
  };

  const login = async (email: string, password: string) => {
    const response = await authAPI.login({ email, password });
    handleLoginResponse(response);
  };

  const loginWithOTP = async (phone: string, code: string) => {
    const response = await authAPI.verifyOTP({ phone, code });
    handleLoginResponse(response);
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('currentTenant');
    setUser(null);
    setCurrentTenant(null);
  };

  const switchTenant = (tenantId: string) => {
    if (user?.tenants) {
      const tenant = user.tenants.find(t => t.tenant_id === tenantId);
      if (tenant) {
        setCurrentTenant(tenant);
        localStorage.setItem('currentTenant', tenantId);
      }
    }
  };

  const updateUser = (patch: Partial<LoginResponse>) => {
    setUser(prev => {
      if (!prev) return prev;
      const updated = { ...prev, ...patch };
      localStorage.setItem('user', JSON.stringify(updated));
      return updated;
    });
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        currentTenant,
        login,
        loginWithOTP,
        logout,
        switchTenant,
        updateUser,
        isAuthenticated: !!user && !!currentTenant,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};


