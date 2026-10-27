import type {SettingGroup} from '@/lib/services/admin';

/** 表单里的值: 开关为布尔, 其余都按字符串编辑 */
export type FormValues = Record<string, string | boolean>;

/** 已保存的值转为表单值, 密钥字段不回显 */
export function toFormValues(group: SettingGroup): FormValues {
  const values: FormValues = {};
  for (const f of group.fields) {
    const v = group.value[f.key];
    values[f.key] = f.type === 'bool' ? !!v : f.secret ? '' : String(v ?? '');
  }
  return values;
}

/** 与已保存的值比较, 只返回改动的字段; 密钥字段填了就算改动, 整数字段保留原始输入, 提交时再校验 */
export function diff(group: SettingGroup, values: FormValues): FormValues {
  const changes: FormValues = {};
  for (const f of group.fields) {
    const v = values[f.key];
    if (typeof v === 'boolean') {
      if (v !== !!group.value[f.key]) changes[f.key] = v;
    } else if (f.secret) {
      if (v !== '') changes[f.key] = v;
    } else if (v.trim() !== String(group.value[f.key] ?? '')) {
      changes[f.key] = v.trim();
    }
  }
  return changes;
}
