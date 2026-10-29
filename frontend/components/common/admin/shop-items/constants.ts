import type {AdminShopItem, ShopItemRequest} from '@/lib/services/admin';

/** 商品列表每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 上架状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '已上架'},
  {value: 'disabled', label: '已下架'},
];

/** 与后端 shop.Validate 一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const MAX_BUNDLE_LINES = 8;
export const MAX_REQUIREMENTS = 16;
export const MAX_TAB_LENGTH = 16;
/** 限购和库存的上限 */
export const MAX_LIMIT = 1_000_000;

/** 条件编辑器中统计范围用到的地图数上限 */
export const VENUE_LIMIT = 100;

/** 商品转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(item: AdminShopItem): ShopItemRequest {
  return {
    code: item.code,
    name: item.name,
    description: item.description,
    image_url: item.image_url,
    tab: item.tab,
    sort_order: item.sort_order,
    goods: item.goods,
    price: item.price,
    original_price: item.original_price,
    requirements: item.requirements,
    limits: item.limits,
    stock: item.stock,
    starts_at: item.starts_at,
    ends_at: item.ends_at,
    is_enabled: item.is_enabled,
  };
}

/** 商品在 now 时的销售状态 */
export type SaleStatus = 'disabled' | 'upcoming' | 'selling' | 'expired';

export const SALE_STATUS_LABELS: Record<SaleStatus, string> = {
  disabled: '已下架',
  upcoming: '未开售',
  selling: '销售中',
  expired: '已过期',
};

export function saleStatus(item: AdminShopItem, now: number): SaleStatus {
  if (!item.is_enabled) return 'disabled';
  if (item.starts_at && new Date(item.starts_at).getTime() > now) return 'upcoming';
  if (item.ends_at && new Date(item.ends_at).getTime() <= now) return 'expired';
  return 'selling';
}
