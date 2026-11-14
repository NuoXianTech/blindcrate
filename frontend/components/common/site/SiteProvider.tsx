'use client';

import {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react';
import {UNKNOWN_QUALITY, qualityStyle, qualityStyleSheet, type Quality, type QualityStyle} from '@/lib/quality';
import services from '@/lib/services';
import type {PublicSettings} from '@/lib/site';

interface SiteContextValue {
  settings: PublicSettings;
  qualities: Quality[];
  /** 向后端读取最新的公开设置 */
  refresh: () => void;
}

const SiteContext = createContext<SiteContextValue | null>(null);

/**
 * 把服务端读取的公开设置和品质交给客户端组件, 见 app/layout.tsx; 纯色品质的类名在这里生成样式表
 */
export function SiteProvider({settings, qualities, children}: {
  settings: PublicSettings;
  qualities: Quality[];
  children: React.ReactNode;
}) {
  // 服务端读取的设置有缓存, 在站内切换页面时也不会重新读取; 这里是浏览器向后端读到的最新值, 服务端送来新的设置后作废
  const [live, setLive] = useState<{ base: PublicSettings; settings: PublicSettings } | null>(null);
  const refresh = useCallback(() => {
    services.settings.getPublic()
        .then((latest) => setLive({base: settings, settings: {...settings, ...latest}}))
        .catch(() => {
          // 读取失败时继续使用服务端读取的设置
        });
  }, [settings]);
  const current = live?.base === settings ? live.settings : settings;
  const value = useMemo(() => ({settings: current, qualities, refresh}), [current, qualities, refresh]);
  return (
    <SiteContext.Provider value={value}>
      {/* 颜色已按 #rrggbb 过滤 */}
      <style dangerouslySetInnerHTML={{__html: qualityStyleSheet(qualities)}} />
      {children}
    </SiteContext.Provider>
  );
}

function useSiteContext() {
  const value = useContext(SiteContext);
  if (!value) throw new Error('useSite must be used within SiteProvider');
  return value;
}

/**
 * 读取站点公开设置
 */
export function useSite(): PublicSettings {
  return useSiteContext().settings;
}

/**
 * 读取站点公开设置, 并在挂载时向后端读取一次最新值, 读到后全站的公开设置一起更新
 * 用在必须和后端一致的页面(如登录、注册表单): 页面上的设置可能是旧的, 后端已经按新的设置校验
 */
export function useLiveSite(): PublicSettings {
  const {settings, refresh} = useSiteContext();
  useEffect(refresh, [refresh]);
  return settings;
}

/**
 * 读取品质: 按排序值排列的品质, 以及按等级取品质, 等级不存在时为未知品质
 */
export function useQualities(): { qualities: QualityStyle[]; qualityOf: (level: number) => QualityStyle } {
  const {qualities} = useSiteContext();
  return useMemo(() => {
    const styles = qualities.map(qualityStyle);
    const byLevel = new Map(styles.map((q) => [q.level, q]));
    return {qualities: styles, qualityOf: (level: number) => byLevel.get(level) ?? UNKNOWN_QUALITY};
  }, [qualities]);
}
