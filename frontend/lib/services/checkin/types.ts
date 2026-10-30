import type {Bundle} from '../auction';
import type {RewardLine} from '../reward';

/** 签到周期中的一天, 对应后端 checkin.DayView */
export interface CheckinDay {
  day: number;
  reward: RewardLine[];
}

/** 玩家的签到情况, 对应后端 checkin.StatusView */
export interface CheckinStatus {
  /** 关闭时其他字段为空 */
  enabled: boolean;
  /** 今天已签到 */
  checked_in: boolean;
  /** 连续签到的天数, 今天还没签时为到昨天为止的天数, 已断开时为 0 */
  streak: number;
  /** 今天签到领(或已领)第几天的奖励, 没有配置奖励时为 0 */
  today: number;
  /** 累计签到天数 */
  total: number;
  /** 整个周期的奖励 */
  days: CheckinDay[];
}

/** 签到的结果, 对应后端 checkin.CheckinResult */
export interface CheckinResult {
  /** 含今天在内的连续签到天数 */
  streak: number;
  /** 领到的是第几天的奖励 */
  day: number;
  /** 实际得到的货币和物品, 已下架的物品按回收价折成了货币 */
  received: RewardLine[];
}

/** 后台看到的一天, 对应后端 checkin.Day */
export interface AdminCheckinDay {
  day: number;
  reward: Bundle;
  created_at: string;
  updated_at: string;
}

export interface CheckinDaysData {
  /** 按天排列, 从第 1 天开始连续 */
  days: AdminCheckinDay[];
  /** 最多的天数 */
  max_days: number;
}

/** 整体保存签到奖励的请求体, 第 1 项为第 1 天 */
export interface UpdateCheckinDaysRequest {
  days: { reward: Bundle }[];
}
