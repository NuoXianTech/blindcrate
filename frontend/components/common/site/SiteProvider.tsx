'use client';

import {createContext, useContext} from 'react';
import type {PublicSettings} from '@/lib/site';

const SiteContext = createContext<PublicSettings | null>(null);

/**
 * 把服务端读取的公开设置交给客户端组件, 见 app/layout.tsx
 */
export function SiteProvider({settings, children}: { settings: PublicSettings; children: React.ReactNode }) {
  return <SiteContext.Provider value={settings}>{children}</SiteContext.Provider>;
}

/**
 * 读取站点公开设置
 */
export function useSite(): PublicSettings {
  const settings = useContext(SiteContext);
  if (!settings) throw new Error('useSite must be used within SiteProvider');
  return settings;
}
