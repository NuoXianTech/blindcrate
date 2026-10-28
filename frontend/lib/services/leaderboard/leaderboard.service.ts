import {BaseService} from '../core';
import type {Leaderboard, LeaderboardDetail} from './types';

export class LeaderboardService extends BaseService {
  protected static readonly basePath = '/leaderboards';

  /** 启用的榜单, 按排序值排列 */
  static listBoards() {
    return this.get<{ boards: Leaderboard[] }>('');
  }

  /** 榜单的前 N 名和自己的名次 */
  static getBoard(code: string) {
    return this.get<LeaderboardDetail>(`/${encodeURIComponent(code)}`);
  }
}
