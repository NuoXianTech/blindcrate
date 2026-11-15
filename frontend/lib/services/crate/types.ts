import type {RewardLine} from '../reward';

/** 开启箱子的使用效果, 与后端 internal/apps/crate/constants.go 的 EffectOpen 一致; 背包中带这个效果的物品可以开启 */
export const CRATE_EFFECT = 'open_crate';

/** 箱子中的一项, 开启一次抽中其中一项, 对应后端 crate.Prize; amount 为每次掉落的最小数量 */
export interface CratePrize extends RewardLine {
  /** 每次掉落的最大数量, 与 amount 相同时数量固定 */
  max_amount: number;
  /** 开启一次抽中这一项的概率 */
  probability: number;
}

/** 玩家看到的箱子, 对应后端 crate.View */
export interface CrateView {
  item_id: number;
  name: string;
  description: string;
  /** 品质等级, 见 useQualities */
  quality: number;
  image_url: string;
  /** 背包中的数量 */
  quantity: number;
  /** 按品质从高到低, 货币排在最后; 概率之和为 1 */
  prizes: CratePrize[];
}

/** 开启一次的结果, 对应后端 crate.Result */
export interface CrateOpenResult {
  /** 开启记录 ID */
  id: number;
  /** 抽中并已经发放的货币或物品, amount 为这次得到的数量 */
  prize: RewardLine;
  /** 开启后背包中剩余的箱子数 */
  quantity: number;
}
