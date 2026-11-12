import type {PageParams} from '../core';

/** 当前用户基本信息, 对应后端 BasicUserInfo */
export interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  email: string;
  avatar_url: string;
  is_admin: boolean;
}

export interface CredentialsRequest {
  /** 登录时可填用户名或已验证的邮箱 */
  username: string;
  password: string;
}

export interface RegisterRequest extends CredentialsRequest {
  /** 开启注册时验证邮箱后必填 */
  email?: string;
  /** 发到邮箱的验证码 */
  code?: string;
}

export interface UpdateUserInfoRequest {
  /** 不传表示不修改 */
  nickname?: string;
  /** 不传表示不修改, 传空字符串表示清除头像 */
  avatar_url?: string;
}

export interface ChangePasswordRequest {
  /** 未设置过密码(仅第三方登录)的用户可不填 */
  old_password?: string;
  new_password: string;
}

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
  nickname?: string;
  avatar_url?: string;
}
