'use server';

import {cookies} from 'next/headers';
import {updateTag} from 'next/cache';
import {PUBLIC_SETTINGS_TAG} from './site';

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? 'http://localhost:8000';

/** 管理员保存系统设置后调用, 让站点名称、公告等立即生效; 只有管理员能触发 */
export async function refreshPublicSettings() {
  const res = await fetch(`${BACKEND_ORIGIN}/api/v1/auth/user-info`, {
    headers: {cookie: (await cookies()).toString()},
    cache: 'no-store',
    signal: AbortSignal.timeout(3000),
  });
  if (!res.ok) return;
  const {data} = (await res.json()) as { data: { is_admin?: boolean } | null };
  if (data?.is_admin) updateTag(PUBLIC_SETTINGS_TAG);
}
