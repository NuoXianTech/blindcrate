import {BaseService} from '../core';
import type {UserStats} from './types';

export class StatsService extends BaseService {
  protected static readonly basePath = '/stats';

  /** 当前用户在一个统计范围的累计数据和最近 7 天的数据, 范围如 all、mode:pve、venue:dock */
  static getStats(scope: string) {
    return this.get<UserStats>('', {params: {scope}});
  }
}
