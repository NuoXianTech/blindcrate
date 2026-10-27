import {BaseService} from '../core';
import type {ChangePasswordRequest, CredentialsRequest, UpdateUserInfoRequest, UserInfo} from './types';

export class AuthService extends BaseService {
  protected static readonly basePath = '/auth';

  static login(body: CredentialsRequest) {
    return this.post<UserInfo>('/login', body);
  }

  static register(body: CredentialsRequest) {
    return this.post<UserInfo>('/register', body);
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
