import type {Condition, StatMetric} from '../admin';
import type {Bundle} from '../auction';
import type {CodeLabel, PageParams} from '../core';
import type {RewardLine} from '../reward';
import type {RequirementProgress} from '../venue';

/** 任务的周期, 对应后端 mission.CycleDaily 等 */
export const MISSION_CYCLE = {
  daily: 'daily',
  weekly: 'weekly',
  once: 'once',
} as const;

export type MissionCycle = (typeof MISSION_CYCLE)[keyof typeof MISSION_CYCLE];

/** 玩家看到的任务, 对应后端 mission.MissionView */
export interface Mission {
  code: string;
  name: string;
  description: string;
  cycle: MissionCycle;
  /** 当前周期的指标值, 净收益可以为负 */
  progress: number;
  target: number;
  /** 净收益指标的货币, 进度和目标按它的最小单位计; 计数指标为空 */
  currency: string;
  /** 货币的展示小数位, 计数指标为 0 */
  decimals: number;
  reward: RewardLine[];
  /** 每个任务条件的达成情况 */
  requirements: RequirementProgress[];
  /** 满足全部任务条件 */
  eligible: boolean;
  /** 当前周期已领取 */
  claimed: boolean;
  /** 已完成、满足条件且未领取 */
  claimable: boolean;
}

/** 一种周期的任务, 对应后端 mission.GroupView */
export interface MissionGroup {
  cycle: MissionCycle;
  /** 每日、每周或成就 */
  label: string;
  /** 下次重置的时间, 成就为 null */
  resets_at: string | null;
  missions: Mission[];
}

export interface MissionsData {
  /** 按每日、每周、成就排列, 没有任务的周期也返回 */
  groups: MissionGroup[];
  /** 可以领取的任务数 */
  claimable: number;
}

/** 领取的结果 */
export interface ClaimMissionResult {
  /** 实际得到的货币和物品, 已下架的物品按回收价折成了货币 */
  received: RewardLine[];
}

/** 后台看到的任务, 对应后端 mission.Mission */
export interface AdminMission {
  id: number;
  /** 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  cycle: MissionCycle;
  /** 统计范围, 如 all、mode:ranked、venue:dock */
  scope: string;
  /** 统计指标, 如 matches、profit:chip */
  metric: string;
  target: number;
  reward: Bundle;
  /** 领取条件, 需要全部满足 */
  requirements: Condition[];
  /** 同一周期内从小到大排列 */
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListMissionsParams extends PageParams {
  /** 按代码或名称搜索 */
  keyword?: string;
  cycle?: MissionCycle;
  is_enabled?: boolean;
}

export interface ListMissionsData {
  total: number;
  missions: AdminMission[];
}

/** 编辑任务时可以选择的周期和指标 */
export interface MissionOptions {
  cycles: CodeLabel[];
  /** 已注册的计数指标和每种货币的净收益 */
  metrics: StatMetric[];
}

/** 新建或修改任务的请求体, 修改时需提交全部字段 */
export type MissionRequest = Omit<AdminMission, 'id' | 'created_at' | 'updated_at'>;
