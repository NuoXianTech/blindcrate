import type {Mission} from '@/lib/services/mission';
import {formatAmount} from '@/lib/utils';

const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** 进度条的比例, 净收益为负时按 0 算, 超过目标时按 1 算 */
export function progressRatio(mission: Mission) {
  return Math.min(1, Math.max(0, mission.progress / mission.target));
}

/** 进度或目标的展示文字, 净收益按货币的小数位展示 */
export function valueText(mission: Mission, value: number) {
  return mission.currency ? formatAmount(value, mission.decimals) : String(value);
}

/** 距离重置的时间, 如 "2 天 3 小时后重置" */
export function resetText(ms: number) {
  if (ms >= DAY) return `${Math.floor(ms / DAY)} 天 ${Math.floor((ms % DAY) / HOUR)} 小时后重置`;
  if (ms >= HOUR) return `${Math.floor(ms / HOUR)} 小时 ${Math.floor((ms % HOUR) / MINUTE)} 分后重置`;
  return `${Math.floor(ms / MINUTE)} 分 ${Math.floor((ms % MINUTE) / 1000)} 秒后重置`;
}

/** 列表中的顺序: 可领取的在前, 已领取的在后, 其余保持后端的排序 */
export function missionRank(mission: Mission) {
  if (mission.claimable) return 0;
  return mission.claimed ? 2 : 1;
}

/** 一组任务中可领取的数量 */
export function claimableCount(missions: Mission[]) {
  return missions.filter((m) => m.claimable).length;
}
