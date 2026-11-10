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
export const MAX_SWINGS = 8;
/** 拍场, 与后端 venue.LotRule 的校验一致 */
export const MAX_LOTS = 8;
export const MAX_TIERS = 6;
export const MAX_GAP_SECONDS = 30;
/** 每个模式出场的人机风格数量上限 */
export const MAX_BOT_PERSONAS = 10;
export const MAX_USES = 10;
export const MAX_ALLOWED_ITEMS = 50;
/** 主持人, 与后端 venue.HostRule 的校验一致: 撒谎率上限为基点 */
export const MAX_LIE_RATE = 2500;
/** 对局中使用的效果场景, 与后端 effect.SceneMatch 对应 */
export const SCENE_MATCH = 'match';

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
    // 档次的金额由后端推导, 不提交
    lots: {
      min: venue.lots?.min ?? 0,
      max: venue.lots?.max ?? 0,
      tiers: (venue.lots?.tiers ?? []).map(({code, name, weight, warehouse}) => ({code, name, weight, warehouse})),
      gap_seconds: venue.lots?.gap_seconds ?? 0,
      coverage: venue.lots?.coverage ?? 0,
    },
    auction: venue.auction,
    bots: venue.bots,
    pve: venue.pve,
    consumables: venue.consumables,
    host: venue.host,
  };
}
