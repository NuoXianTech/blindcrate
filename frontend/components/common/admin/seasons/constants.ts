import {SEASON_STATUS, type SeasonStatus} from '@/lib/services/season';

/** 赛季列表每页条数 */
export const PAGE_SIZE = 20;

/** 与后端 season.Validate 一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,24}$/;

/** 新建赛季时默认的时长 */
export const DEFAULT_DAYS = 30;

/** 新建赛季时默认的段位分继承比例, 百分比 */
export const DEFAULT_RATING_CARRY = 50;

/** 赛季状态的名称 */
export const STATUS_LABELS: Record<SeasonStatus, string> = {
  [SEASON_STATUS.upcoming]: '未开始',
  [SEASON_STATUS.active]: '进行中',
  [SEASON_STATUS.ended]: '已结束',
  [SEASON_STATUS.settled]: '已结算',
};

/** 状态筛选 */
export type StatusFilter = 'all' | SeasonStatus;

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  ...Object.values(SEASON_STATUS).map((value) => ({value, label: STATUS_LABELS[value]})),
];

/** 时间转为 datetime-local 输入框的值, 按本地时区, 精确到分钟 */
export function toLocalInput(value: string | Date): string {
  const d = new Date(value);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** datetime-local 输入框的值转为 RFC3339, 格式不对时返回 null */
export function fromLocalInput(value: string): string | null {
  const d = new Date(value);
  return value && !Number.isNaN(d.getTime()) ? d.toISOString() : null;
}
