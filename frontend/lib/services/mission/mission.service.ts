import {BaseService} from '../core';
import type {ClaimMissionResult, MissionsData} from './types';

export class MissionService extends BaseService {
  protected static readonly basePath = '/missions';

  /** 启用的任务, 按周期分组, 带当前周期的进度和领取情况 */
  static list() {
    return this.get<MissionsData>('');
  }

  /** 领取当前周期的奖励 */
  static claim(code: string) {
    return this.post<ClaimMissionResult>(`/${encodeURIComponent(code)}/claim`);
  }
}
