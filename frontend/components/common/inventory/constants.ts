/** 物品网格每页数量 */
export const ITEM_PAGE_SIZE = 24;

/** 背包记录每页条数 */
export const RECORD_PAGE_SIZE = 20;

/** 出售记录每页条数 */
export const SELL_ORDER_PAGE_SIZE = 10;

/** 一次最多出售多少种物品, 与后端一致 */
export const MAX_SELL_LINES = 100;

/** 一种物品一次最多出售的数量, 与后端一致 */
export const MAX_SELL_QUANTITY = 1_000_000;

/** 记录页只看某个物品时的筛选条件 */
export type ItemFilter = { id: number; name: string };
