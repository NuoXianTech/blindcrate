import type {PageParams} from '../core';

/** 背包中的一种物品, 对应后端 inventory.UserItemDetail */
export interface UserItem {
  item_id: number;
  name: string;
  description: string;
  /** 品质等级, 见 useQualities */
  quality: number;
  /** 类别代码, 如 collectible 藏品 */
  category: string;
  image_url: string;
  sell_currency: string;
  /** 0 表示不可卖出 */
  sell_price: number;
  /** 在仓库中占的格数 */
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

/** 出售的一种物品, 单价和货币需与背包中看到的一致, 否则后端拒绝出售 */
export interface SellItemLine {
  item_id: number;
  quantity: number;
  sell_currency: string;
  sell_price: number;
}

export interface SellItemsRequest {
  /** 同一物品只能出现一次, 最多 100 种 */
  items: SellItemLine[];
  /** 出售内容相同的重复提交只成交一次 */
  idempotency_key: string;
  /** 确认框中看到的手续费率(基点), 与当前不一致时后端拒绝出售 */
  fee_rate?: number;
}

/** 出售单中的一种物品, 记录出售时的名称和价格 */
export interface SellOrderItem {
  id: number;
  order_id: number;
  item_id: number;
  item_name: string;
  quantity: number;
  sell_currency: string;
  sell_price: number;
  /** 数量 × 单价 */
  amount: number;
}

/** 出售单在一种货币上的入账, 实际到账为 amount - fee */
export interface SellIncome {
  currency: string;
  /** 卖出金额, 未扣手续费 */
  amount: number;
  fee: number;
  /** 扣除手续费后的可用余额 */
  balance_after: number;
}

/** 出售单, 对应后端 inventory.SellOrderDetail */
export interface SellOrder {
  id: number;
  /** 出售时的手续费率, 单位为基点 */
  fee_rate: number;
  /** 按物品 ID 排序 */
  items: SellOrderItem[];
  /** 按货币代码排序 */
  incomes: SellIncome[];
  created_at: string;
}

export interface ListSellOrdersParams extends PageParams {
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListSellOrdersData {
  total: number;
  orders: SellOrder[];
}
