import type {AdminMail} from '@/lib/services/mailbox';

/** 列表每页条数 */
export const PAGE_SIZE = 20;

/** 与后端 mailbox 的限制一致 */
export const MAX_SENDER = 32;
export const MAX_TITLE = 64;
export const MAX_CONTENT = 2000;
export const MAX_EXPIRE_DAYS = 365;
export const MAX_RECIPIENTS = 1000;
export const MAX_ATTACHMENT_LINES = 8;
export const MAX_REQUIREMENTS = 16;

/** 条件编辑器中统计范围用到的地图数上限 */
export const VENUE_LIMIT = 100;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 输入框中的用户 ID 列表, 用逗号、空格或换行分隔; 有不是正整数的项时返回 null */
export function parseUserIDs(text: string): number[] | null {
  const parts = text.split(/[\s,，]+/).filter(Boolean);
  const ids: number[] = [];
  for (const part of parts) {
    if (!/^\d{1,19}$/.test(part)) return null;
    const id = Number(part);
    if (!Number.isSafeInteger(id) || id < 1) return null;
    if (!ids.includes(id)) ids.push(id);
  }
  return ids;
}

/** 邮件在信箱中的状态 */
export function mailStatus(mail: AdminMail, now: number): string {
  if (mail.removed_at) return '已删除';
  if (new Date(mail.expires_at).getTime() <= now) return '已过期';
  if (mail.claimed_at) return '已领取';
  return mail.read_at ? '已读' : '未读';
}
