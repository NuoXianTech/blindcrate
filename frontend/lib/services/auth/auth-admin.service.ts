import {BaseService} from '../core';
import type {AdminUser, ListUsersData, ListUsersParams, UpdateUserRequest} from './types';

export class AuthAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listUsers(params: ListUsersParams) {
    return this.get<ListUsersData>('/users', {params});
  }

  static updateUser(id: number, body: UpdateUserRequest) {
    return this.patch<Pick<AdminUser, 'id' | 'username' | 'nickname' | 'email' | 'avatar_url' | 'is_admin'>>(
        `/users/${id}`,
        body,
    );
  }

  /** 封禁到指定时间(RFC3339), 并让其所有设备下线 */
  static banUser(id: number, bannedUntil: string, reason: string) {
    return this.put<null>(`/users/${id}/ban`, {banned_until: bannedUntil, reason});
  }

  static unbanUser(id: number) {
    return this.delete<null>(`/users/${id}/ban`);
  }

  /** 停用或启用, 停用时让其所有设备下线 */
  static setUserDisabled(id: number, disabled: boolean) {
    return this.put<null>(`/users/${id}/disabled`, {disabled});
  }

  /** 修改其他用户的密码, 不需要原密码, 并让其所有设备下线; 自己的密码用 auth.changePassword 修改 */
  static setUserPassword(id: number, password: string) {
    return this.put<null>(`/users/${id}/password`, {password});
  }
}
