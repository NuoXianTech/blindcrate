"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { AuthService, type UserInfo } from "@/lib/services/auth";
import { ApiError, UNAUTHORIZED_EVENT } from "@/lib/services/core";

interface AuthContextValue {
  /** 当前用户, 未登录为 null */
  user: UserInfo | null;
  /** 首次获取用户信息中 */
  loading: boolean;
  /** 用户主动退出登录, 与会话过期区分 */
  loggedOut: boolean;
  /** 登录、注册成功后写入用户 */
  setUser: (user: UserInfo | null) => void;
  /** 重新获取用户信息 */
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/** 全局登录状态: 启动时获取一次当前用户, 任意接口返回 401 时清空 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [loggedOut, setLoggedOut] = useState(false);

  const refresh = useCallback(async () => {
    try {
      setUser(await AuthService.getUserInfo());
    } catch (err) {
      // 网络等错误不当作登出, 保留现有状态
      if (err instanceof ApiError && err.status === 401) setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    await AuthService.logout();
    setLoggedOut(true);
    setUser(null);
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 挂载时拉取一次用户信息
    void refresh();
    const onUnauthorized = () => setUser(null);
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [refresh]);

  const login = useCallback((info: UserInfo | null) => {
    setLoggedOut(false);
    setUser(info);
  }, []);

  const value = useMemo(
    () => ({ user, loading, loggedOut, setUser: login, refresh, logout }),
    [user, loading, loggedOut, login, refresh, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
