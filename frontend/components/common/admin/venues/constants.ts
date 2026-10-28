import type {AdminVenue, VenueRequest} from '@/lib/services/admin';

/** 地图列表每页条数 */
export const PAGE_SIZE = 20;

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '启用中'},
  {value: 'disabled', label: '已停用'},
];

/** 模式, 与后端 venue.ModePve / ModeRanked 对应 */
export const MODES = [
  {value: 'pve', label: '人机'},
  {value: 'ranked', label: '排位'},
] as const;

/** 与后端 venue 的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,24}$/;
export const MAX_ENTRY_COST_LINES = 8;
export const MAX_REQUIREMENTS = 16;
export const MAX_SLOTS = 8;
export const MAX_ROUND_FACTORS = 12;
/** 人机的三个等级 */
export const BOT_LEVELS = ['低', '中', '高'];

/** 地图转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(venue: AdminVenue): VenueRequest {
  return {
    code: venue.code,
    name: venue.name,
    description: venue.description,
    image_url: venue.image_url,
    sort_order: venue.sort_order,
    is_enabled: venue.is_enabled,
    pve_enabled: venue.pve_enabled,
    ranked_enabled: venue.ranked_enabled,
    bid_currency: venue.bid_currency,
    entry_cost: venue.entry_cost,
    requirements: venue.requirements,
    warehouse: venue.warehouse,
    auction: venue.auction,
    bots: venue.bots,
    pve: venue.pve,
  };
}
