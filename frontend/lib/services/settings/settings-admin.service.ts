import {BaseService} from '../core';
import type {ListSettingsData, SettingGroup, UpdateSettingRequest} from './types';

export class SettingsAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listSettings(category?: string) {
    return this.get<ListSettingsData>('/settings', {params: {category}});
  }

  static getSetting(key: string) {
    return this.get<SettingGroup>(`/settings/${key}`);
  }

  /** 保存一个设置组, 立即生效, 返回保存后的设置组 */
  static updateSetting(key: string, body: UpdateSettingRequest) {
    return this.put<SettingGroup>(`/settings/${key}`, body);
  }

  /** 恢复默认值, 密钥字段会被清空, 返回保存后的设置组 */
  static resetSetting(key: string) {
    return this.post<SettingGroup>(`/settings/${key}/reset`);
  }

  /** 用已保存的邮件设置发送测试邮件 */
  static testMail(to: string) {
    return this.post<null>('/settings/mail/test', {to});
  }
}
