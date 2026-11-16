import type {SettingField, SettingGroup} from '@/lib/services/settings';

/** 表单里的值: 开关为布尔, 整数列表为每一项的输入, 其余都按字符串编辑 */
export type FormValue = string | boolean | string[];
export type FormValues = Record<string, FormValue>;

/** 整数单位的后缀 */
export const UNIT_SUFFIX: Record<NonNullable<SettingField['unit']>, string> = {
  bp: '%',
  second: '秒',
  millisecond: '毫秒',
};

/** 整数转为输入框的文字, 基点按百分比显示 */
export function formatInt(f: SettingField, n: number): string {
  return f.unit === 'bp' ? String(n / 100) : String(n);
}

/** 输入框的文字转为整数, 基点按百分比输入且最多两位小数; 格式不正确时返回 null */
export function parseIntInput(f: SettingField, s: string): number | null {
  if (f.unit === 'bp') {
    return /^-?\d+(\.\d{1,2})?$/.test(s) ? Math.round(Number(s) * 100) : null;
  }
  return /^-?\d+$/.test(s) ? Number(s) : null;
}

/** 已保存的值转为表单值, 密钥字段不回显 */
export function toFormValues(group: SettingGroup): FormValues {
  const values: FormValues = {};
  for (const f of group.fields) {
    const v = group.value[f.key];
    if (f.type === 'bool') {
      values[f.key] = !!v;
    } else if (f.type === 'int') {
      values[f.key] = formatInt(f, Number(v ?? 0));
    } else if (f.type === 'int_list') {
      values[f.key] = ((v as number[] | undefined) ?? []).map((n) => formatInt(f, n));
    } else {
      values[f.key] = f.secret ? '' : String(v ?? '');
    }
  }
  return values;
}

/** 与已保存的值比较, 只返回改动的字段; 密钥字段填了就算改动, 整数保留原始输入, 提交时再校验 */
export function diff(group: SettingGroup, values: FormValues): FormValues {
  const saved = toFormValues(group);
  const changes: FormValues = {};
  for (const f of group.fields) {
    const v = values[f.key];
    const old = saved[f.key];
    if (typeof v === 'boolean') {
      if (v !== old) changes[f.key] = v;
    } else if (Array.isArray(v)) {
      const items = v.map((s) => s.trim());
      if (items.join('\n') !== (old as string[]).join('\n')) changes[f.key] = items;
    } else if (f.secret) {
      if (v !== '') changes[f.key] = v;
    } else if (v.trim() !== old) {
      changes[f.key] = v.trim();
    }
  }
  return changes;
}

/** 整数格式不正确的提示 */
export function intError(f: SettingField, index?: number): string {
  const name = index === undefined ? f.label : `${f.label}第 ${index + 1} 项`;
  return f.unit === 'bp' ? `${name}需为数字, 最多两位小数` : `${name}需为整数`;
}
