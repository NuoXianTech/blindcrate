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
