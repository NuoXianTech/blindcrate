/** 一个条件, 对应后端 condition.Condition */
export interface Condition {
  type: string;
  /** 参数, 结构由条件类型决定 */
  params: Record<string, string | number | boolean>;
}

/** 条件参数的描述, 后台据此生成表单, 对应后端 condition.Field */
export interface ConditionField {
  key: string;
  label: string;
  help: string;
  type: 'string' | 'bool' | 'int';
  /** text / select / switch / number, 或选择器 currency / item / quality / scope / metric */
  input: 'text' | 'select' | 'switch' | 'number' | 'currency' | 'item' | 'quality' | 'scope' | 'metric';
  /** 下拉框 select 和选择器 metric 的可选值 */
  options?: ConditionOption[];
  /** 整数的单位, bp 为基点, 按百分比输入 */
  unit?: 'bp';
}

/** 条件参数选择器的一个可选值 */
export interface ConditionOption {
  value: string;
  label: string;
}

/** 条件类型, 对应后端 condition.TypeView */
export interface ConditionType {
  type: string;
  label: string;
  description: string;
  fields: ConditionField[];
}
