import {BaseService} from '../core';
import type {
  ListSettingRevisionsData,
  ListSettingRevisionsParams,
  ListSettingsData,
  SettingGroup,
  SettingRevisionDetail,
  UpdateSettingRequest,
} from './types';

export class SettingsAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listSettings(category?: string) {
    return this.get<ListSettingsData>('/settings', {params: {category}});
  }

  static getSetting(key: string) {
    return this.get<SettingGroup>(`/settings/${key}`);
  }

  /** 保存一个设置组, 立即生效, 返回保存后的设置组; 已被别人修改时返回 409 */
  static updateSetting(key: string, body: UpdateSettingRequest) {
    return this.put<SettingGroup>(`/settings/${key}`, body);
  }

  /** 设置组的历史版本, 从新到旧 */
  static listSettingRevisions(key: string, params: ListSettingRevisionsParams) {
    return this.get<ListSettingRevisionsData>(`/settings/${key}/revisions`, {params});
  }

  static getSettingRevision(key: string, version: number) {
    return this.get<SettingRevisionDetail>(`/settings/${key}/revisions/${version}`);
  }

  /** 回滚到 target 版本, 生成一个新版本 */
  static rollbackSetting(key: string, body: { version: number; target: number }) {
    return this.post<SettingGroup>(`/settings/${key}/rollback`, body);
  }

  /** 恢复默认值, 密钥字段会被清空 */
  static resetSetting(key: string, body: { version: number }) {
    return this.post<SettingGroup>(`/settings/${key}/reset`, body);
  }

  /** 用已保存的邮件设置发送测试邮件 */
  static testMail(to: string) {
    return this.post<null>('/settings/mail/test', {to});
  }
}
