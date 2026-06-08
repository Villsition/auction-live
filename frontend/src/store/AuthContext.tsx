import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';
import type { User } from '../types';
import { auth as authApi } from '../api';

interface AuthState {
  user: User | null;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string, nickname: string, role?: number) => Promise<void>;
  logout: () => void;
  isSeller: boolean;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const saved = localStorage.getItem('user');
    return saved ? JSON.parse(saved) : null;
  });
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'));

  const saveAuth = useCallback((u: User, t: string) => {
    setUser(u);
    setToken(t);
    localStorage.setItem('user', JSON.stringify(u));
    localStorage.setItem('token', t);
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const data = await authApi.login({ username, password });
    saveAuth(data.user, data.token);
  }, [saveAuth]);

  const register = useCallback(async (username: string, password: string, nickname: string, role = 0) => {
    const data = await authApi.register({ username, password, nickname, role });
    saveAuth(data.user, data.token);
  }, [saveAuth]);

  const logout = useCallback(() => {
    setUser(null);
    setToken(null);
    localStorage.removeItem('user');
    localStorage.removeItem('token');
  }, []);

  return (
    <AuthContext.Provider value={{
      user, token, login, register, logout,
      isSeller: (user?.role ?? 0) >= 1,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}
