'use client';

import {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react';
import {usePathname, useRouter} from 'next/navigation';
import services from '@/lib/services';
import type {UserInfo} from '@/lib/services/auth';
import {ApiError, UNAUTHORIZED_EVENT} from '@/lib/services/core';
import {ROUTES, loginHref, requiresAuth} from '@/lib/access';

interface AuthContextValue {
  /** 当前用户, 未登录为 null */
  user: UserInfo | null;
  /** 首次获取用户信息中 */
  loading: boolean;
  /** 登录、注册、修改资料后写入用户 */
  setUser: (user: UserInfo | null) => void;
  /** 重新获取用户信息 */
  refresh: () => Promise<void>;
  /** 退出登录并回到首页 */
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * 全局登录状态: 启动时获取一次当前用户
 * 任意接口返回 401 时清空用户, 当前页面需要登录则跳去登录页(规则见 lib/access.ts)
 */
export function AuthProvider({children}: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      setUser(await services.auth.getUserInfo());
    } catch (err) {
      // 网络等错误不当作登出, 保留现有状态; 401 由下面的事件处理
      if (err instanceof ApiError && err.status === 401) setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    await services.auth.logout();
    setUser(null);
    router.replace(ROUTES.landing);
  }, [router]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 挂载时拉取一次用户信息
    void refresh();
  }, [refresh]);

  useEffect(() => {
    const onUnauthorized = () => {
      setUser(null);
      if (requiresAuth(pathname)) router.replace(loginHref(pathname));
    };
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [pathname, router]);

  const value = useMemo(() => ({user, loading, setUser, refresh, logout}), [user, loading, refresh, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** 当前登录用户和登录状态操作, 需在 AuthProvider 内使用 */
export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
