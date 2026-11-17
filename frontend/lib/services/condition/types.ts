import type {Bundle} from '../reward';

/** 参数的值: 文本、数字、开关, 多选的列表, 或结构化的一组奖励、一组条件 */
export type ParamValue = string | number | boolean | number[] | string[] | Bundle | Bundle[] | Condition[];

/** 一个条件, 对应后端 condition.Condition */
export interface Condition {
  type: string;
  /** 参数, 结构由条件类型决定 */
  params: Record<string, ParamValue>;
}

/**
 * 参数的描述, 后台据此生成表单, 对应后端 params.Field; 条件、物品的使用效果和人机策略共用
 * 下拉框和输入建议的可选值由后端按当前的数据填好, 后端新增一种可选值来源时不用改前端
 */
export interface ConditionField {
  key: string;
  label: string;
  help: string;
  /** 值的类型: string / bool / int / int_list / string_list, 或结构化的 bundle(一组奖励) / bundle_list(几组奖励) / conditions(一组条件) */
  type: string;
  /**
   * 输入方式: text / textarea / datetime / number / switch,
   * select 从可选值中选一个, suggest 带输入建议也可以手动填写, multiselect 多选,
   * 结构化的类型与 type 相同
   */
  input: string;
  /** select、suggest 和 multiselect 的可选值 */
  options?: ConditionOption[];
  /** 整数的单位, bp 为基点, 按百分比输入 */
  unit?: string;
  /** 新建时的默认值 */
  default?: string;
  /** 字符串不能为空, 列表不能为空 */
  required: boolean;
}

/** 参数的一个可选值, 对应后端 params.Option */
export interface ConditionOption {
  value: string;
  label: string;
  /** 次要的说明, 如代码 */
  hint?: string;
  /** 按这个品质的颜色展示 */
  quality?: number;
  /** 不能新选, 已经选了的照常展示, 如已下架的物品 */
  disabled?: boolean;
}

/** 条件类型, 对应后端 condition.TypeView */
export interface ConditionType {
  type: string;
  label: string;
  description: string;
  fields: ConditionField[];
}
