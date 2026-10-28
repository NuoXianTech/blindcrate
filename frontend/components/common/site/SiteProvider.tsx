'use client';

import {createContext, useContext, useMemo} from 'react';
import {UNKNOWN_QUALITY, qualityStyle, qualityStyleSheet, type Quality, type QualityStyle} from '@/lib/quality';
import type {PublicSettings} from '@/lib/site';

const SiteContext = createContext<{ settings: PublicSettings; qualities: Quality[] } | null>(null);

/**
 * 把服务端读取的公开设置和品质交给客户端组件, 见 app/layout.tsx; 纯色品质的类名在这里生成样式表
 */
export function SiteProvider({settings, qualities, children}: {
  settings: PublicSettings;
  qualities: Quality[];
  children: React.ReactNode;
}) {
  const value = useMemo(() => ({settings, qualities}), [settings, qualities]);
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
