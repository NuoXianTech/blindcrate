import type {Condition} from '../admin';
import type {Bundle} from '../auction';
import type {CodeLabel, PageParams} from '../core';
import type {RewardLine} from '../reward';

/** 邮件相关的 ws 推送 */
export const MAILBOX_MESSAGES = {
  /** 收到新邮件或全服邮件有变化, 需要重新获取未读数, 没有 data */
  new: 'mailbox:new',
} as const;

/** 邮件来源, 对应后端 mailbox.SourceAdmin 等; 其他模块登记的来源也会出现 */
export const MAIL_SOURCE = {
  admin: 'admin',
  broadcast: 'broadcast',
} as const;

/** 信箱中的邮件, 对应后端 mailbox.MailView */
export interface Mail {
  id: number;
  source: string;
  sender: string;
  title: string;
  content: string;
  /** 附件, 为空时没有附件 */
  attachments: RewardLine[];
  read_at: string | null;
  claimed_at: string | null;
  expires_at: string;
  created_at: string;
}

export type ListMailsParams = PageParams;

export interface ListMailsData {
  total: number;
  mails: Mail[];
}

/** 未读和附件未领取的封数 */
export interface MailCounts {
  unread: number;
  claimable: number;
}

export interface ReadAllMailsData {
  /** 本次标记的封数 */
  read: number;
}

/** 领取附件的结果 */
export interface ClaimMailsResult {
  /** 领取了附件的封数 */
  claimed: number;
  /** 附件暂时无法领取而跳过的封数, 只有一键领取会跳过 */
  skipped: number;
  /** 实际得到的货币和物品, 已下架的物品按回收价折成了货币 */
  received: RewardLine[];
}

/** 后台查到的邮件, 对应后端 mailboxadmin.MailItem */
export interface AdminMail {
  id: number;
  user_id: number;
  /** 用户已不存在时为空 */
  username: string;
  source: string;
  source_id: string;
  sender: string;
  title: string;
  content: string;
  attachments: Bundle;
  read_at: string | null;
  claimed_at: string | null;
  expires_at: string;
  created_at: string;
  /** 玩家删除或超出保留封数被移出信箱的时间, 在信箱中时为空 */
  removed_at: string | null;
}

export interface ListAdminMailsParams extends PageParams {
  user_id?: number;
  source?: string;
  /** 按标题搜索 */
  keyword?: string;
}

export interface ListAdminMailsData {
  total: number;
  mails: AdminMail[];
}

export interface ListMailSourcesData {
  sources: CodeLabel[];
}

/** 发给指定用户 */
export interface SendMailsRequest {
  /** 1-1000 人 */
  user_ids: number[];
  /** 为空时为 "系统" */
  sender: string;
  title: string;
  content: string;
  attachments: Bundle;
  /** 有效期天数, 0 表示使用默认有效期 */
  expire_days: number;
  /** 每次打开发送页生成一个, 重复提交只发送一次 */
  idempotency_key: string;
}

export interface SendMailsData {
  /** 实际发送的封数, 重复提交时不计已发送的 */
  sent: number;
}

/** 全服邮件, 对应后端 mailboxadmin.BroadcastItem */
export interface MailBroadcast {
  id: number;
  sender: string;
  title: string;
  content: string;
  attachments: Bundle;
  /** 收件条件, 需要全部满足 */
  requirements: Condition[];
  /** 只发给这个时间之前注册的用户, 为空时之后注册的也能收到 */
  registered_before: string | null;
  expires_at: string;
  created_by: number;
  created_username: string;
  revoked_at: string | null;
  revoked_by: number;
  revoked_username: string;
  created_at: string;
  /** 已放进信箱的人数, 包括之后删除的 */
  expanded: number;
  /** 已领取附件的人数 */
  claimed: number;
}

export interface ListMailBroadcastsParams extends PageParams {
  keyword?: string;
}

export interface ListMailBroadcastsData {
  total: number;
  broadcasts: MailBroadcast[];
}

/** 发送全服邮件 */
export interface MailBroadcastRequest {
  sender: string;
  title: string;
  content: string;
  attachments: Bundle;
  requirements: Condition[];
  /** 发送之后注册的用户也能收到 */
  include_new_users: boolean;
  expire_days: number;
  /** 每次打开发送框生成一个, 重复提交只发送一次 */
  idempotency_key: string;
}

export interface RevokeMailBroadcastData {
  /** 撤回后的全服邮件, 不带统计 */
  broadcast: Omit<MailBroadcast, 'created_username' | 'revoked_username' | 'expanded' | 'claimed'>;
  /** 移出信箱的封数, 已领取附件的邮件保留 */
  removed: number;
}
