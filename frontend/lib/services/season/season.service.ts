import {BaseService} from '../core';
import type {CurrentSeasonData} from './types';

export class SeasonService extends BaseService {
  protected static readonly basePath = '/seasons';

  /** 进行中的赛季和下一个未开始的赛季 */
  static getCurrent() {
    return this.get<CurrentSeasonData>('/current');
  }
}
