import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { authAPI, LoginResponse, TenantInfo } from '../services/api';

interface AuthContextType {
  user: LoginResponse | null;
  currentTenant: TenantInfo | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  switchTenant: (tenantId: string) => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<LoginResponse | null>(null);
  const [currentTenant, setCurrentTenant] = useState<TenantInfo | null>(null);

  useEffect(() => {
    // Load user from localStorage
    const storedUser = localStorage.getItem('user');
    const storedToken = localStorage.getItem('token');
    const storedTenant = localStorage.getItem('currentTenant');

    if (storedUser && storedToken) {
      try {
        const userData = JSON.parse(storedUser);
        setUser(userData);
        
        if (storedTenant) {
          const tenant = userData.tenants?.find((t: TenantInfo) => t.tenant_id === storedTenant);
          setCurrentTenant(tenant || userData.tenants?.[0] || null);
        } else {
          setCurrentTenant(userData.tenants?.[0] || null);
        }
      } catch (e) {
        localStorage.removeItem('user');
        localStorage.removeItem('token');
        localStorage.removeItem('currentTenant');
      }
    }
  }, []);

  const login = async (email: string, password: string) => {
    const response = await authAPI.login({ email, password });
    localStorage.setItem('token', response.token);
    localStorage.setItem('user', JSON.stringify(response));
    
    if (response.tenants && response.tenants.length > 0) {
      setCurrentTenant(response.tenants[0]);
      localStorage.setItem('currentTenant', response.tenants[0].tenant_id);
    }
    
    setUser(response);
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
        // TODO: Update JWT token with new tenant
      }
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        currentTenant,
        login,
        logout,
        switchTenant,
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


