import {BaseService, captchaConfig} from '../core';
import type {ChangePasswordRequest, CredentialsRequest, RegisterRequest, UpdateUserInfoRequest, UserInfo} from './types';

export class AuthService extends BaseService {
  protected static readonly basePath = '/auth';

  /** captcha 为人机验证的令牌, 登录开启人机验证时必填 */
  static login(body: CredentialsRequest, captcha?: string) {
    return this.post<UserInfo>('/login', body, captchaConfig(captcha));
  }

  /** captcha 为人机验证的令牌, 注册开启人机验证时必填 */
  static register(body: RegisterRequest, captcha?: string) {
    return this.post<UserInfo>('/register', body, captchaConfig(captcha));
  }

  /** 向注册用的邮箱发送验证码, 只在开启注册时验证邮箱后可用; 注册开启人机验证时也要带令牌 */
  static sendRegisterCode(email: string, captcha?: string) {
    return this.post<null>('/register/code', {email}, captchaConfig(captcha));
  }

  static logout() {
    return this.post<null>('/logout');
  }

  static getUserInfo() {
    return this.get<UserInfo>('/user-info');
  }

  static updateUserInfo(body: UpdateUserInfoRequest) {
    return this.patch<UserInfo>('/user-info', body);
  }

  /** 修改密码, 成功后其他设备会下线 */
  static changePassword(body: ChangePasswordRequest) {
    return this.put<null>('/password', body);
  }

  static sendEmailCode(email: string) {
    return this.post<null>('/email/code', {email});
  }

  static verifyEmail(email: string, code: string) {
    return this.post<null>('/email/verify', {email, code});
  }
}
