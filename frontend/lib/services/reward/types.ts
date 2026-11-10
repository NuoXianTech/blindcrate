/** 一组货币和物品, 对应后端 reward.Bundle */
export interface Bundle {
  currencies?: { currency: string; amount: number }[];
  items?: { item_id: number; quantity: number }[];
}

/** 奖励、价格或附件中的一种货币或物品, 带名称便于展示, 对应后端 reward.Line */
export interface RewardLine {
  /** 货币代码, 与 item_id 二选一 */
  currency?: string;
  item_id?: number;
  name: string;
  image_url: string;
  /** 物品的品质等级, 货币为 0 */
  quality: number;
  /** 货币的展示用小数位, 物品为 0 */
  decimals: number;
  /** 货币金额或物品数量 */
  amount: number;
}
