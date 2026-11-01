import type {ConditionField} from '../admin';
import type {PageParams} from '../core';

/** 策略参数, 结构由决策策略决定; 基点参数为整数 */
export type BotParams = Record<string, string | number | boolean>;

/** 人机昵称, 对应后端 bot.Name */
export interface BotName {
  id: number;
  name: string;
  /** 停用后开局时不再选取 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListBotNamesParams extends PageParams {
  /** 按昵称搜索, 不区分大小写 */
  keyword?: string;
  is_enabled?: boolean;
}

export interface ListBotNamesData {
  total: number;
  names: BotName[];
}

/** 批量新建昵称 */
export interface CreateBotNamesRequest {
  /** 1-200 个, 去掉首尾空白后为 1-32 个字符; 已存在的和重复的跳过 */
  names: string[];
  is_enabled: boolean;
}

export interface CreateBotNamesData {
  created: BotName[];
  /** 已存在或重复而跳过的昵称 */
  skipped: string[];
}

/** 修改昵称 */
export type BotNameRequest = Pick<BotName, 'name' | 'is_enabled'>;

/** 人机风格: 一种决策策略和它的参数, 对应后端 bot.Persona */
export interface BotPersona {
  id: number;
  /** 地图按它引用, 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  /** 决策策略的代码 */
  strategy: string;
  /** 保存时按策略规范化, 缺少的参数取默认值 */
  params: BotParams;
  created_at: string;
  updated_at: string;
}

export interface ListBotPersonasParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
}

export interface ListBotPersonasData {
  total: number;
  personas: BotPersona[];
}

/** 新建或修改风格, 修改时需提交全部字段 */
export type BotPersonaRequest = Pick<BotPersona, 'code' | 'name' | 'description' | 'strategy' | 'params'>;

/** 决策策略, 由玩法模块注册, 对应后端 bot.StrategyView */
export interface BotStrategy {
  code: string;
  label: string;
  description: string;
  /** 参数的表单描述, 与条件类型相同 */
  fields: ConditionField[];
  /** 新建风格时的默认参数 */
  defaults: BotParams;
}

/** 开局时人机风格的快照, 对应后端 bot.Snapshot; 真人为空对象 */
export interface BotSnapshot {
  code?: string;
  name?: string;
  strategy?: string;
  params?: BotParams;
}
