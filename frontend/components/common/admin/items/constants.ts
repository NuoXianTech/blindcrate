import type {AdminItem, ItemRequest} from '@/lib/services/admin';

/** 物品列表每页条数 */
export const PAGE_SIZE = 20;

/** 上下架状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '上架中'},
  {value: 'disabled', label: '已下架'},
];

/** 与后端 ItemRequest 的校验一致 */
export const MAX_SELL_PRICE = 1_000_000_000_000;
export const MAX_SIZE = 10;

/** 物品转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(item: AdminItem): ItemRequest {
  return {
    name: item.name,
    description: item.description,
    quality: item.quality,
    category: item.category,
    image_url: item.image_url,
    sell_currency: item.sell_currency,
    sell_price: item.sell_price,
    width: item.width,
    height: item.height,
    is_enabled: item.is_enabled,
  };
}
