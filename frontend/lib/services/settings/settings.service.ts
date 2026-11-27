import type {PublicSettings} from '@/lib/site';
import {BaseService} from '../core';
import type {CaptchaStatus} from './types';

export class SettingsService extends BaseService {
  protected static readonly basePath = '/settings';

  /** 公开设置的最新值, 不经过服务端缓存; 页面上的公开设置可能是旧的, 需要和后端一致的地方(如出售手续费率)使用 */
  static getPublic() {
    return this.get<Partial<PublicSettings>>('/public');
  }

  /** 当前用户这次操作要不要人机验证: 场景已开启且不在验证冷却内时需要; 场景与后端 captcha 的场景一致 */
  static captchaStatus(scene: string) {
    return this.get<CaptchaStatus>(`/captcha/${encodeURIComponent(scene)}`);
  }
}
