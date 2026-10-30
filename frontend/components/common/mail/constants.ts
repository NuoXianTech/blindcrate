import type {Mail} from '@/lib/services/mailbox';

/** 信箱每页封数 */
export const MAIL_PAGE_SIZE = 20;

/** 还有附件没有领取 */
export function claimable(mail: Mail) {
  return mail.attachments.length > 0 && !mail.claimed_at;
}

/** 剩余有效期的描述, 如 "3 天后过期", 不足一天时按小时 */
export function expiryText(expiresAt: string, now: number = Date.now()) {
  const hours = Math.floor((new Date(expiresAt).getTime() - now) / 3_600_000);
  if (hours >= 24) return `${Math.floor(hours / 24)} 天后过期`;
  if (hours >= 1) return `${hours} 小时后过期`;
  return '即将过期';
}
