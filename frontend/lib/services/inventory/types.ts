import type { PageParams } from "../core";

/** 背包中的一种物品, 对应后端 inventory.UserItemDetail */
export interface UserItem {
  item_id: number;
  name: string;
  description: string;
  /** 品质 1-7, 见 lib/quality.ts */
  quality: number;
  image_url: string;
  sell_currency: string;
  /** 0 表示不可卖出 */
  sell_price: number;
  /** 在保险柜中占的格数 */
  width: number;
  height: number;
  /** 下架后不会再掉落或发放, 已有的仍在背包中 */
  is_enabled: boolean;
  /** 物品已被删除, 按占位物品展示 */
  is_placeholder: boolean;
  quantity: number;
  /** 首次获得时间 */
  created_at: string;
  updated_at: string;
}

export interface ListUserItemsParams extends PageParams {
  quality?: number;
}

export interface ListUserItemsData {
  total: number;
  items: UserItem[];
}

/** 用户自己的背包流水, 对应后端 inventory.TransactionItem */
export interface InventoryTransaction {
  id: number;
  item_id: number;
  /** 变动时的物品名称 */
  item_name: string;
  /** 正数获得, 负数消耗 */
  quantity: number;
  quantity_after: number;
  biz_type: string;
  biz_id: string;
  remark: string;
  created_at: string;
}

export interface ListInventoryTransactionsParams extends PageParams {
  item_id?: number;
  biz_type?: string;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListInventoryTransactionsData {
  total: number;
  transactions: InventoryTransaction[];
}
