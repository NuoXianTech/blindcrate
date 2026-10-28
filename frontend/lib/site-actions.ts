'use server';

import {cookies} from 'next/headers';
import {updateTag} from 'next/cache';
import {PUBLIC_SETTINGS_TAG, QUALITIES_TAG} from './site';

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? 'http://localhost:8000';

/** 当前请求的用户是否为管理员 */
async function isAdmin(): Promise<boolean> {
  const res = await fetch(`${BACKEND_ORIGIN}/api/v1/auth/user-info`, {
    headers: {cookie: (await cookies()).toString()},
    cache: 'no-store',
    signal: AbortSignal.timeout(3000),
  });
  if (!res.ok) return false;
  const {data} = (await res.json()) as { data: { is_admin?: boolean } | null };
  return !!data?.is_admin;
}

/** 管理员保存系统设置后调用, 让站点名称、公告等立即生效; 只有管理员能触发 */
export async function refreshPublicSettings() {
  if (await isAdmin()) updateTag(PUBLIC_SETTINGS_TAG);
}

/** 管理员修改品质后调用, 让品质的名称和颜色立即生效; 只有管理员能触发 */
export async function refreshQualities() {
  if (await isAdmin()) updateTag(QUALITIES_TAG);
}
