/** 设置项的值: 整数列表为数组 */
export type SettingValue = string | number | boolean | number[];

/** 下拉框的可选值 */
export interface SettingOption {
  value: string;
  label: string;
}

/** 设置项的描述, 后台据此生成表单, 对应后端 settings.Field */
export interface SettingField {
  key: string;
  label: string;
  help: string;
  type: 'string' | 'bool' | 'int' | 'int_list';
  input: 'text' | 'textarea' | 'password' | 'select' | 'switch' | 'number' | 'list';
  /** 下拉框的可选值, 引用字段为读取时的可选值 */
  options?: SettingOption[];
  /** 引用其他模块的数据, 如 currency 为启用的货币 */
  ref?: string;
  /** 整数的单位, bp 为基点, 按百分比输入 */
  unit?: 'bp' | 'second' | 'millisecond';
  /** 密钥, 读取时不返回明文 */
  secret: boolean;
  /** 会通过公开接口返回给前端 */
  public: boolean;
}

/** 设置组的分类 */
export interface SettingCategory {
  key: string;
  label: string;
}

/** 一个设置组, 对应后端 settings.GroupView */
export interface SettingGroup {
  key: string;
  label: string;
  description: string;
  category: string;
  fields: SettingField[];
  /** 密钥字段固定为空字符串 */
  value: Record<string, SettingValue>;
  /** 密钥字段是否已设置 */
  secrets_set: Record<string, boolean>;
  /** 无法解密的密钥字段, 需要重新填写或清空才能保存 */
  secrets_broken: string[] | null;
  updated_by: number;
  /** 未保存过为空 */
  updated_at: string | null;
}

export interface ListSettingsData {
  categories: SettingCategory[];
  groups: SettingGroup[];
}

export interface UpdateSettingRequest {
  /** 只需提交改动的字段, 密钥字段留空表示不修改 */
  value: Record<string, SettingValue>;
  /** 需要清空的密钥字段 */
  clear_secrets?: string[];
}
