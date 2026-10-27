import { cacheLife, cacheTag } from "next/cache";

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
}

/** 缓存标签, 管理员保存设置后按标签刷新, 见 lib/site-actions.ts */
export const PUBLIC_SETTINGS_TAG = "public-settings";

/** 与后端设置的默认值一致, 后端不可用时使用 */
const DEFAULT_SETTINGS: PublicSettings = {
  site: { name: "BlindCrate", description: "", logo_url: "", announcement: "", icp: "", contact_email: "" },
  auth: { register_enabled: true, email_login_enabled: true },
};

/** 站点简介为空时使用 */
export const DEFAULT_DESCRIPTION = "盲盒开箱 · 保险柜竞拍对战平台";

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? "http://localhost:8000";

async function fetchPublicSettings(): Promise<PublicSettings | null> {
  try {
    const res = await fetch(`${BACKEND_ORIGIN}/api/v1/settings/public`, { signal: AbortSignal.timeout(3000) });
    if (!res.ok) return null;
    const { data } = (await res.json()) as { data: Partial<PublicSettings> | null };
    return {
      site: { ...DEFAULT_SETTINGS.site, ...data?.site },
      auth: { ...DEFAULT_SETTINGS.auth, ...data?.auth },
    };
  } catch {
    return null;
  }
}

/** 读取公开设置, 每分钟在后台刷新一次; 后端不可用时用默认值, 并在 10 秒后重试 */
export async function getPublicSettings(): Promise<PublicSettings> {
  "use cache";
  cacheTag(PUBLIC_SETTINGS_TAG);
  const settings = await fetchPublicSettings();
  if (settings) cacheLife("minutes");
  else cacheLife({ stale: 60, revalidate: 10, expire: 300 });
  return settings ?? DEFAULT_SETTINGS;
}
