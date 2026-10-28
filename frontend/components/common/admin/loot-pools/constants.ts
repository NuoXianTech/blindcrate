import type {LootPool, LootPoolRequest} from '@/lib/services/admin';

/** 掉落池列表每页条数 */
export const PAGE_SIZE = 20;

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '启用中'},
  {value: 'disabled', label: '已停用'},
];

/** 与后端 loot 的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,64}$/;
export const MAX_ENTRIES = 200;
export const MAX_WEIGHT = 1_000_000;
/** 物品每次最多掉落的数量 */
export const MAX_ITEM_QUANTITY = 1_000_000;
/** 货币每次最多掉落的金额, 最小单位 */
export const MAX_CURRENCY_AMOUNT = 1_000_000_000_000;

/** 掉落池转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(pool: LootPool): LootPoolRequest {
  return {code: pool.code, name: pool.name, description: pool.description, is_enabled: pool.is_enabled};
}
