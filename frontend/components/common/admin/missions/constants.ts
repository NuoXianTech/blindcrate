import type {Currency} from '@/lib/services/admin';
import type {AdminMission, MissionCycle, MissionRequest} from '@/lib/services/mission';
import {PROFIT_METRIC_PREFIX} from '@/lib/services/stats';

/** 任务列表每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '已启用'},
  {value: 'disabled', label: '已停用'},
];

/** 与后端 mission.Validate 一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const MAX_TARGET = 1_000_000_000_000;
export const MAX_SORT_ORDER = 1_000_000;
export const MAX_BUNDLE_LINES = 8;
export const MAX_REQUIREMENTS = 16;

/** 条件编辑器中统计范围用到的地图数上限 */
export const VENUE_LIMIT = 100;

/** 新建任务时的默认周期 */
export const DEFAULT_CYCLE: MissionCycle = 'daily';

/** 净收益指标的货币, 目标按它的小数位输入; 计数指标为 undefined */
export function metricCurrency(metric: string, currencies: Currency[]): Currency | undefined {
  if (!metric.startsWith(PROFIT_METRIC_PREFIX)) return undefined;
  const code = metric.slice(PROFIT_METRIC_PREFIX.length);
  return currencies.find((c) => c.code === code);
}

/** 任务转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(mission: AdminMission): MissionRequest {
  return {
    code: mission.code,
    name: mission.name,
    description: mission.description,
    cycle: mission.cycle,
    scope: mission.scope,
    metric: mission.metric,
    target: mission.target,
    reward: mission.reward,
    requirements: mission.requirements,
    sort_order: mission.sort_order,
    is_enabled: mission.is_enabled,
  };
}
