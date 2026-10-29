import {BaseService} from '../core';
import type {RatingView} from './types';

export class RankedService extends BaseService {
  protected static readonly basePath = '/ranked';

  /** 自己在当前赛季的段位分 */
  static getRating() {
    return this.get<RatingView>('/rating');
  }
}
