import {PHASES} from '@/lib/auction';
import type {AdminMatch} from '@/lib/services/admin';
import {AUCTION_PHASE} from '@/lib/services/auction';
import {MATCH_STATUS} from '@/lib/services/match';

/** 对局列表每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 对局状态的名称和徽标样式 */
export const STATUS_LABELS: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' }> = {
  [MATCH_STATUS.playing]: {label: '进行中', variant: 'default'},
  [MATCH_STATUS.finished]: {label: '已结束', variant: 'secondary'},
  [MATCH_STATUS.cancelled]: {label: '已取消', variant: 'outline'},
};

/** 对局所处的阶段, 出价时带上轮次, 如 "出价 第 2 轮"; 已结束或已取消时为结束前的阶段 */
export function phaseLabel(m: Pick<AdminMatch, 'phase' | 'round'>) {
  const name = PHASES[m.phase] ?? (m.phase || '-');
  return m.phase === AUCTION_PHASE.bidding && m.round > 0 ? `${name} 第 ${m.round} 轮` : name;
}
