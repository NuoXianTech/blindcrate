import type { PageParams } from "../core";

/** 管理端看到的完整用户, 对应后端 auth.User */
export interface AdminUser {
  id: number;
  username: string;
  email: string;
  email_verified_at: string | null;
  nickname: string;
  avatar_url: string;
  is_disabled: boolean;
  is_admin: boolean;
  banned_until: string | null;
  ban_reason: string;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListUsersParams extends PageParams {
  /** 按用户名或邮箱前缀搜索 */
  keyword?: string;
  is_admin?: boolean;
  is_disabled?: boolean;
  is_banned?: boolean;
}

export interface ListUsersData {
  total: number;
  users: AdminUser[];
}

export interface UpdateUserRequest {
  username?: string;
  nickname?: string;
  avatar_url?: string;
}

export type AuditScope = "user" | "admin";

export interface AuditLog {
  id: number;
  actor_id: number;
  /** 未登录或用户已不存在时为空 */
  actor_username: string;
  scope: AuditScope;
  action: string;
  target_type: string;
  target_id: number;
  detail: Record<string, unknown> | null;
  ip: string;
  user_agent: string;
  created_at: string;
}

export interface ListAuditLogsParams extends PageParams {
  actor_id?: number;
  scope?: AuditScope;
  action?: string;
  target_type?: string;
  target_id?: number;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListAuditLogsData {
  total: number;
  logs: AuditLog[];
}

/** 货币定义, 对应后端 wallet.Currency */
export interface Currency {
  code: string;
  name: string;
  /** 展示用小数位, 金额统一按最小单位的整数存储 */
  decimals: number;
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 物品定义, 对应后端 item.Item */
export interface Item {
  id: number;
  name: string;
  description: string;
  /** 1-7, 见 lib/quality.ts */
  quality: number;
  image_url: string;
  sell_currency: string;
  /** 官方回收价, 最小单位; 0 表示不可卖出 */
  sell_price: number;
  /** 掉落权重, 0 表示不会掉落 */
  weight: number;
  /** 在保险柜中占的格数 */
  width: number;
  height: number;
  /** 下架后不会掉落, 也不能再发放, 已有的保留 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 管理端物品列表项, 附带持有人数 */
export interface AdminItem extends Item {
  holder_count: number;
}

export interface ListItemsParams extends PageParams {
  /** 按名称搜索, 不区分大小写 */
  keyword?: string;
  quality?: number;
  is_enabled?: boolean;
}

export interface ListItemsData {
  total: number;
  items: AdminItem[];
}

/** 新建或修改物品, 修改时也需提交全部字段 */
export type ItemRequest = Pick<
  Item,
  "name" | "description" | "quality" | "image_url" | "sell_currency" | "sell_price" | "weight" | "width" | "height" | "is_enabled"
>;
