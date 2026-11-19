import type {Quality} from '@/lib/quality';
import {BaseService} from '../core';

export class ItemService extends BaseService {
  /** 全部公开品质的最新值, 不经过页面的服务端缓存 */
  static listQualities() {
    return this.get<{ qualities: Quality[] }>('/qualities');
  }
}
