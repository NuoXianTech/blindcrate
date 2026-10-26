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
