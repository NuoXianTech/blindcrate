import type {Bundle} from '../auction';
import type {PageParams} from '../core';
import type {RewardLine} from '../reward';

/** 图鉴中的一种藏品, 对应后端 codex.EntryView; 没有收集时只返回品质和图片 */
export interface CodexEntry {
  item_id: number;
  /** 收集后才返回, 没有收集时为空, 显示为 ??? */
  name: string;
  /** 收集后才返回, 没有收集时为空 */
  description: string;
  /** 品质等级, 见 useQualities */
  quality: number;
  /** 没有收集时也返回, 显示为剪影 */
  image_url: string;
  /** 收集后才返回 */
  sell_currency: string;
  /** 物品当前的卖出单价, 收集后才返回; 0 表示不可出售 */
  sell_price: number;
  collected: boolean;
  /** 第一次获得的时间, 没有收集时为 null */
  first_at: string | null;
  /** 背包中现有的数量, 卖出或用掉后为 0, 仍算已收集 */
  quantity: number;
}

/** 玩家看到的收集组, 对应后端 codex.SetView */
export interface CodexSet {
  code: string;
  name: string;
  description: string;
  /** 组内启用的藏品, 按图鉴中的顺序排列 */
  item_ids: number[];
  /** 其中已收集的数量 */
  collected: number;
  reward: RewardLine[];
  claimed: boolean;
  /** 已集齐且未领取 */
  claimable: boolean;
}

/** 玩家的图鉴, 对应后端 codex.CodexData */
export interface CodexData {
  /** 全部启用的藏品, 按品质从高到低、ID 排列 */
  items: CodexEntry[];
  /** 已收集的藏品数 */
  collected: number;
  /** 启用的收集组, 按排序值和 ID 排列; 组内没有启用的藏品时不返回 */
  sets: CodexSet[];
  /** 可以领取的收集组数 */
  claimable: number;
}

/** 领取的结果 */
export interface ClaimCodexSetResult {
  /** 实际得到的货币和物品, 已下架的物品按回收价折成了货币 */
  received: RewardLine[];
}

/** 后台看到的收集组, 对应后端 codex.Set */
export interface AdminCodexSet {
  id: number;
  /** 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  /** 组内的藏品, 去重后从小到大排列; 已下架的不计入进度 */
  item_ids: number[];
  reward: Bundle;
  /** 从小到大排列 */
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListCodexSetsParams extends PageParams {
  /** 按代码或名称搜索 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListCodexSetsData {
  total: number;
  sets: AdminCodexSet[];
}

/** 新建或修改收集组的请求体, 修改时需提交全部字段 */
export type CodexSetRequest = Omit<AdminCodexSet, 'id' | 'created_at' | 'updated_at'>;
