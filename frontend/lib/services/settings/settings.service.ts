import type {PublicSettings} from '@/lib/site';
import {BaseService} from '../core';

export class SettingsService extends BaseService {
  protected static readonly basePath = '/settings';

  /** 公开设置的最新值, 不经过服务端缓存; 页面上的公开设置可能是旧的, 需要和后端一致的地方(如出售手续费率)使用 */
  static getPublic() {
    return this.get<Partial<PublicSettings>>('/public');
  }
}
