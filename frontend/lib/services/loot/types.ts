import type {PageParams} from '../core';

/** 掉落池, 对应后端 loot.Pool */
export interface LootPool {
  id: number;
  /** 玩法中引用的代码, 小写字母、数字和下划线 */
  code: string;
  name: string;
  description: string;
  /** 停用后不能再抽取 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 掉落池列表项, 附带条目数 */
export interface LootPoolListItem extends LootPool {
  entry_count: number;
}

export interface ListLootPoolsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListLootPoolsData {
  total: number;
  pools: LootPoolListItem[];
}

export type LootPoolRequest = Pick<LootPool, 'code' | 'name' | 'description' | 'is_enabled'>;

/** 掉落池条目, 对应后端 loot.EntryDetail */
export interface LootEntry {
  id: number;
  pool_id: number;
  /** 掉落的物品, 与 currency 二选一 */
  item_id: number;
  currency: string;
  /** 抽中的概率为权重 / 池中可抽取条目的权重之和 */
  weight: number;
  /** 每次掉落的数量范围, 货币为金额 */
  min_quantity: number;
  max_quantity: number;
  created_at: string;
  /** 物品或货币的名称, 已删除的物品为占位名称 */
  name: string;
  /** 物品品质, 货币为 0 */
  quality: number;
  image_url: string;
  /** 物品已上架或货币已启用, 否则不参与抽取 */
  is_available: boolean;
  /** 当前每次抽中的概率, 不参与抽取时为 0 */
  probability: number;
}

export type LootEntryRequest = Pick<LootEntry, 'item_id' | 'currency' | 'weight' | 'min_quantity' | 'max_quantity'>;

export interface LootPoolEntriesData {
  pool: LootPool;
  /** 按条目 ID 排序, 包含不参与抽取的条目 */
  entries: LootEntry[];
}
