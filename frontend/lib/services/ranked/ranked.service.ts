import {BaseService} from '../core';
import type {ListTiersData, RatingView} from './types';

export class RankedService extends BaseService {
  protected static readonly basePath = '/ranked';

  /** 自己在当前赛季的段位分 */
  static getRating() {
    return this.get<RatingView>('/rating');
  }

  /** 全部段位, 按分数线从低到高 */
  static listTiers() {
    return this.get<ListTiersData>('/tiers');
  }
}
