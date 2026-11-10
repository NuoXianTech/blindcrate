import {BaseService} from '../core';
import type {StatMetric} from '../stats';
import type {AdminLeaderboard, LeaderboardRequest, ListLeaderboardsData, ListLeaderboardsParams} from './types';

export class LeaderboardAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listLeaderboards(params: ListLeaderboardsParams) {
    return this.get<ListLeaderboardsData>('/leaderboards', {params});
  }

  /** 可以排名的指标: 计数指标和每种货币的净收益 */
  static listLeaderboardMetrics() {
    return this.get<{ metrics: StatMetric[] }>('/leaderboards/metrics');
  }

  static createLeaderboard(body: LeaderboardRequest) {
    return this.post<AdminLeaderboard>('/leaderboards', body);
  }

  /** 修改后缓存的前 N 名随即失效 */
  static updateLeaderboard(id: number, body: LeaderboardRequest) {
    return this.put<AdminLeaderboard>(`/leaderboards/${id}`, body);
  }

  /** 只删除榜单配置, 统计数据不受影响 */
  static deleteLeaderboard(id: number) {
    return this.delete<null>(`/leaderboards/${id}`);
  }
}
