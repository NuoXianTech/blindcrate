import type {TransactionActors, TransactionFilters} from '../ledger';
import type {PageParams} from '../core';
import type {RewardLine} from '../reward';

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
  /** 在货仓中占的格数 */
  width: number;
  height: number;
  /** 使用效果代码, 空表示不能使用 */
  effect: string;
  /** 可以在背包中直接使用, 见 ItemUseView */
  usable: boolean;
  /** 下架后不会再掉落或发放, 已有的仍在背包中 */
  is_enabled: boolean;
  /** 物品已被删除, 按占位物品展示 */
  is_placeholder: boolean;
  quantity: number;
  /** 首次获得时间 */
  created_at: string;
  updated_at: string;
}

/** 在背包中使用物品的方式, 对应后端 effect.Mode*; 按方式展示, 后端新增效果时不用改前端 */
export const USE_MODE = {
  /** 得到全部内容, 如礼包 */
  grant: 'grant',
  /** 按概率得到其中一项, 如箱子 */
  random: 'random',
  /** 自选其中一项 */
  choose: 'choose',
  /** 不发放内容, 只执行动作 */
  action: 'action',
} as const;

/** 使用后可以得到的一组内容, 对应后端 effect.OptionView */
export interface UseOption {
  /** amount 为数量; 按概率得到且数量不固定时为最小数量 */
  lines: RewardLine[];
  /** 只有一行时这一行数量的上限, 不大于 amount 时数量固定 */
  max_amount: number;
  /** 按概率得到时抽中这一项的概率, 其他方式为 0 */
  probability: number;
}

/** 一个可以在背包中使用的物品, 对应后端 effect.UseView */
export interface ItemUseView {
  item_id: number;
  name: string;
  description: string;
  quality: number;
  image_url: string;
  /** 背包中的数量 */
  quantity: number;
  effect: string;
  /** 效果的名称, 如 "开启箱子" */
  label: string;
  /** 见 USE_MODE */
  mode: string;
  /** 一句话说明使用后会发生什么 */
  summary: string;
  /** 使用时需要填写的文本, 不需要时为 null */
  input: { label: string; placeholder: string; max_length: number } | null;
  /** 可以得到的内容; random 时按品质从高到低, 概率之和为 1 */
  options: UseOption[];
}

export interface UseItemRequest {
  /** 每次使用生成一个, 重复提交只使用一次并返回原来的结果 */
  idempotency_key: string;
  /** 自选时选第几项, 从 0 开始 */
  choice?: number;
  /** 需要填写文本时的内容 */
  input?: string;
}

/** 使用一次的结果, 对应后端 effect.UseResult */
export interface ItemUseResult {
  /** 使用记录 ID */
  id: number;
  /** 得到并已经发放的货币和物品, 没有时为空 */
  received: RewardLine[];
  /** 效果给出的一句结果, 没有时为空 */
  message: string;
  /** 使用后背包中剩余的数量 */
  quantity: number;
}

export interface ListUserItemsParams extends PageParams {
  quality?: number;
}

export interface ListUserItemsData {
  total: number;
  items: UserItem[];
}

/** 背包流水的一行, 对应后端 inventory.Transaction, 后台查看和调整背包时使用; 玩家看到的背包记录见 InventoryRecord */
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

/** 用户自己的一条背包记录, 同一次变动的多种物品合并为一条, 对应后端 inventory.Record */
export interface InventoryRecord {
  /** 这次变动第一行流水的 ID */
  id: number;
  biz_type: string;
  biz_id: string;
  remark: string;
  created_at: string;
  /** 每种物品一行; 按物品筛选时只含这种物品 */
  items: InventoryRecordLine[];
}

/** 背包记录中一种物品的变动, 对应后端 inventory.RecordLine */
export interface InventoryRecordLine {
  item_id: number;
  /** 变动时的物品名称 */
  item_name: string;
  /** 正数获得, 负数消耗 */
  quantity: number;
  quantity_after: number;
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
  /** 记录的条数, 同一次变动的多种物品算一条 */
  total: number;
  records: InventoryRecord[];
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

export interface AdjustInventoryRequest {
  item_id: number;
  /** 正数发放, 负数扣除, 扣除后数量不能为负 */
  quantity: number;
  remark: string;
  idempotency_key: string;
}

/** 全部用户的背包流水, 对应后端 admin.InventoryTransactionItem */
export type AdminInventoryTransaction = InventoryTransaction & TransactionActors;

export interface ListAdminInventoryTransactionsParams extends TransactionFilters {
  item_id?: number;
}

export interface ListAdminInventoryTransactionsData {
  total: number;
  transactions: AdminInventoryTransaction[];
}
