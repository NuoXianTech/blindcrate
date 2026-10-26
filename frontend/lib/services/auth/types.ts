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
