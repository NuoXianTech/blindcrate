import type {AdminLeaderboard, LeaderboardRequest} from '@/lib/services/admin';
import {LEADERBOARD_PERIOD, type LeaderboardPeriod} from '@/lib/services/leaderboard';

/** 榜单列表每页条数 */
export const PAGE_SIZE = 20;

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '启用中'},
  {value: 'disabled', label: '已停用'},
];

/** 统计周期 */
export const PERIODS: { value: LeaderboardPeriod; label: string }[] = [
  {value: LEADERBOARD_PERIOD.total, label: '累计'},
  {value: LEADERBOARD_PERIOD.day, label: '当天'},
];

/** 与后端 leaderboard 的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,64}$/;
export const MIN_SIZE = 10;
export const MAX_SIZE = 500;
export const DEFAULT_SIZE = 100;

/** 榜单转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(board: AdminLeaderboard): LeaderboardRequest {
  return {
    code: board.code,
    name: board.name,
    description: board.description,
    metric: board.metric,
    scope: board.scope,
    period: board.period,
    size: board.size,
    min_matches: board.min_matches,
    sort_order: board.sort_order,
    is_enabled: board.is_enabled,
  };
}
