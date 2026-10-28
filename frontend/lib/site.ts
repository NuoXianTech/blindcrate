import {cacheLife, cacheTag} from 'next/cache';
import type {Quality} from './quality';

/**
 * 站点公开设置, 对应后端 GET /api/v1/settings/public, 由管理员在系统设置中修改
 * 只在服务端读取并缓存, 客户端组件通过 SiteProvider 拿到
 */
export interface PublicSettings {
  site: {
    name: string;
    description: string;
    logo_url: string;
    /** 空表示不展示 */
    announcement: string;
    icp: string;
    contact_email: string;
  };
  auth: {
    register_enabled: boolean;
    email_login_enabled: boolean;
  };
  economy: {
    /** 基础货币的代码: 新手资金和已删除物品的回收价使用, 也是新建物品和地图时默认选中的货币 */
    base_currency: string;
    /** 出售手续费率, 单位为基点 */
    sell_fee_rate: number;
    /** 新用户注册时发放的基础货币, 0 表示不发放 */
    signup_grant: number;
    /** 人机每日收益上限, 0 表示不限制 */
    pve_daily_profit_cap: number;
  };
}

/** 缓存标签, 管理员保存设置后按标签刷新, 见 lib/site-actions.ts */
export const PUBLIC_SETTINGS_TAG = 'public-settings';

/** 与后端设置的默认值一致, 后端不可用时使用 */
const DEFAULT_SETTINGS: PublicSettings = {
  site: {name: 'BlindCrate', description: '', logo_url: '', announcement: '', icp: '', contact_email: ''},
  auth: {register_enabled: true, email_login_enabled: true},
  economy: {base_currency: 'chip', sell_fee_rate: 500, signup_grant: 5000, pve_daily_profit_cap: 5000},
};

/** 站点简介为空时使用 */
export const DEFAULT_DESCRIPTION = '盲盒开箱 · 保险柜竞拍对战平台';

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? 'http://localhost:8000';

async function fetchPublicSettings(): Promise<PublicSettings | null> {
  try {
    const res = await fetch(`${BACKEND_ORIGIN}/api/v1/settings/public`, {signal: AbortSignal.timeout(3000)});
    if (!res.ok) return null;
    const {data} = (await res.json()) as { data: Partial<PublicSettings> | null };
    return {
      site: {...DEFAULT_SETTINGS.site, ...data?.site},
      auth: {...DEFAULT_SETTINGS.auth, ...data?.auth},
      economy: {...DEFAULT_SETTINGS.economy, ...data?.economy},
    };
  } catch {
    return null;
  }
}

/** 读取公开设置, 每分钟在后台刷新一次; 后端不可用时用默认值, 并在 10 秒后重试 */
export async function getPublicSettings(): Promise<PublicSettings> {
  'use cache';
  cacheTag(PUBLIC_SETTINGS_TAG);
  const settings = await fetchPublicSettings();
  if (settings) cacheLife('minutes');
  else cacheLife({stale: 60, revalidate: 10, expire: 300});
  return settings ?? DEFAULT_SETTINGS;
}

/** 品质的缓存标签, 管理员修改品质后按标签刷新, 见 lib/site-actions.ts */
export const QUALITIES_TAG = 'qualities';

/** 与后端品质的初始数据一致, 后端不可用时使用 */
const DEFAULT_QUALITIES: Quality[] = [
  {level: 1, code: 'gray', name: '灰色', color: '#a1a5ab', effect: '', scan_ms: 400, sort_order: 10},
  {level: 2, code: 'green', name: '绿色', color: '#4bcb71', effect: '', scan_ms: 500, sort_order: 20},
  {level: 3, code: 'blue', name: '蓝色', color: '#3ba6f5', effect: '', scan_ms: 700, sort_order: 30},
  {level: 4, code: 'purple', name: '紫色', color: '#ad74ff', effect: '', scan_ms: 1000, sort_order: 40},
  {level: 5, code: 'gold', name: '金色', color: '#fab72a', effect: '', scan_ms: 1500, sort_order: 50},
  {level: 6, code: 'red', name: '红色', color: '#f94144', effect: '', scan_ms: 2000, sort_order: 60},
  {level: 7, code: 'rainbow', name: '炫彩', color: '#ff6ad5', effect: 'rainbow', scan_ms: 3000, sort_order: 70},
];

async function fetchQualities(): Promise<Quality[] | null> {
  try {
    const res = await fetch(`${BACKEND_ORIGIN}/api/v1/qualities`, {signal: AbortSignal.timeout(3000)});
    if (!res.ok) return null;
    const {data} = (await res.json()) as { data: { qualities: Quality[] } | null };
    return data?.qualities ?? null;
  } catch {
    return null;
  }
}

/** 读取品质, 每分钟在后台刷新一次; 后端不可用时用初始数据, 并在 10 秒后重试 */
export async function getQualities(): Promise<Quality[]> {
  'use cache';
  cacheTag(QUALITIES_TAG);
  const qualities = await fetchQualities();
  if (qualities) cacheLife('minutes');
  else cacheLife({stale: 60, revalidate: 10, expire: 300});
  return qualities ?? DEFAULT_QUALITIES;
}
