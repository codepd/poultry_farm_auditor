import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { authAPI, LoginResponse, TenantInfo } from '../services/api';

interface AuthContextType {
  user: LoginResponse | null;
  currentTenant: TenantInfo | null;
  login: (email: string, password: string, rememberMe?: boolean) => Promise<void>;
  loginWithOTP: (phone: string, code: string, rememberMe?: boolean) => Promise<void>;
  logout: () => void;
  switchTenant: (tenantId: string) => void;
  updateUser: (patch: Partial<LoginResponse>) => void;
  isAuthenticated: boolean;
  isAuthLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function loadStoredUser(): LoginResponse | null {
  try {
    const stored = localStorage.getItem('user');
    if (stored) return JSON.parse(stored);
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
  const [isAuthLoading, setIsAuthLoading] = useState(true);

  useEffect(() => {
    const bootstrapAuth = async () => {
      const token = localStorage.getItem('token');
      const storedUser = loadStoredUser();

      // Existing token or no stored user: nothing to refresh on startup.
      if (token || !storedUser) {
        setIsAuthLoading(false);
        return;
      }

      try {
        const newToken = await authAPI.refreshToken();
        if (newToken) {
          localStorage.setItem('token', newToken);
          setUser(storedUser);
          setCurrentTenant(loadStoredTenant(storedUser));
        } else {
          throw new Error('No token from refresh');
        }
      } catch {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('currentTenant');
        setUser(null);
        setCurrentTenant(null);
      } finally {
        setIsAuthLoading(false);
      }
    };

    bootstrapAuth();
  }, []);

  const handleLoginResponse = (response: LoginResponse) => {
    localStorage.setItem('token', response.token);
    localStorage.setItem('user', JSON.stringify(response));
    
    if (response.tenants && response.tenants.length > 0) {
      setCurrentTenant(response.tenants[0]);
      localStorage.setItem('currentTenant', response.tenants[0].tenant_id);
    }
    
    setUser(response);
  };

  const login = async (email: string, password: string, rememberMe = false) => {
    const response = await authAPI.login({ email, password, remember_me: rememberMe });
    handleLoginResponse(response);
  };

  const loginWithOTP = async (phone: string, code: string, rememberMe = false) => {
    const response = await authAPI.verifyOTP({ phone, code, remember_me: rememberMe });
    handleLoginResponse(response);
  };

  const logout = () => {
    authAPI.logout().catch(() => {});
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
        isAuthLoading,
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


