import {nanoid} from 'nanoid';
import {check} from '@/hooks/use-submit';
import type {Condition, ConditionField, ConditionType, ParamValue} from '@/lib/services/condition';
import type {Bundle} from '@/lib/services/reward';
import type {Currency} from '@/lib/services/wallet';
import {amountInput, fromLocalInput, parseAmount, toLocalInput} from '@/lib/utils';

/** 物品的数量上限, 与后端 reward 的校验一致 */
const MAX_ITEM_QUANTITY = 1_000_000;
/** 货币的金额上限, 最小单位 */
const MAX_CURRENCY_AMOUNT = 1_000_000_000_000;
/** 基点按百分比输入, 保留 2 位小数 */
export const PERCENT_DECIMALS = 2;

/** 编辑中的一行货币或物品, 数量保留输入框里的原文 */
export interface BundleDraft {
  key: string;
  /** 物品行的物品 ID, 货币行为 null */
  itemID: number | null;
  /** 货币行的货币代码, 物品行为空 */
  currency: string;
  amount: string;
}

/** 编辑中的一个参数: 文本和数值保留输入框里的原文, 多选为选中的值, 结构化的参数为各自的草稿 */
export type ParamDraft = string | boolean | string[] | BundleDraft[] | BundleDraft[][] | ConditionDraft[];

/** 编辑中的一组参数 */
export type ParamsDraft = Record<string, ParamDraft>;

/** 编辑中的一个条件 */
export interface ConditionDraft {
  key: string;
  type: string;
  params: ParamsDraft;
}

/** 转换结构化的参数时需要的数据: 货币的小数位, 以及嵌套条件的类型 */
export interface DraftContext {
  currencies: Currency[];
  /** 条件类型, 没有 conditions 参数时可以不传 */
  types?: ConditionType[];
}

/** 货币的小数位, 找不到时按 0 */
export function decimalsOf(currencies: Currency[], code: string): number {
  return currencies.find((c) => c.code === code)?.decimals ?? 0;
}

export function toBundleDrafts(bundle: Bundle, currencies: Currency[]): BundleDraft[] {
  return [
    ...(bundle.currencies ?? []).map((line) => ({
      key: nanoid(),
      itemID: null,
      currency: line.currency,
      amount: amountInput(line.amount, decimalsOf(currencies, line.currency)),
    })),
    ...(bundle.items ?? []).map((line) => ({key: nanoid(), itemID: line.item_id, currency: '', amount: String(line.quantity)})),
  ];
}

/** 转为请求中的 Bundle, label 为校验提示的前缀, 如 "入场代价" */
export function fromBundleDrafts(drafts: BundleDraft[], currencies: Currency[], label: string): Bundle {
  const bundle: Required<Bundle> = {currencies: [], items: []};
  drafts.forEach((d, i) => {
    const row = `${label}第 ${i + 1} 行`;
    if (d.currency) {
      const amount = parseAmount(d.amount, decimalsOf(currencies, d.currency)) ?? 0;
      check(amount < 1 || amount > MAX_CURRENCY_AMOUNT ? `${row}: 金额格式不正确` : null);
      bundle.currencies.push({currency: d.currency, amount});
      return;
    }
    check(d.itemID === null ? `${row}: 请选择物品` : null);
    const quantity = parseAmount(d.amount) ?? 0;
    check(quantity < 1 || quantity > MAX_ITEM_QUANTITY ? `${row}: 数量需为 1-${MAX_ITEM_QUANTITY} 的整数` : null);
    bundle.items.push({item_id: d.itemID ?? 0, quantity});
  });
  return bundle;
}

/** 一组奖励最多的行数, 与后端 reward 的校验一致 */
export const MAX_BUNDLE_LINES = 20;
/** 结构化参数中几组奖励、嵌套条件最多的个数 */
export const MAX_PARAM_ITEMS = 8;

/** 按字段类型给参数默认值, 条件、物品的使用效果和人机策略共用; 有默认值的用默认值, 下拉框留空等管理员选 */
export function newParamsDraft(fields: ConditionField[]): ParamsDraft {
  const params: ParamsDraft = {};
  for (const field of fields) {
    if (field.type === 'bool') params[field.key] = false;
    else if (field.input === 'multiselect' || field.type === 'bundle' || field.type === 'bundle_list' || field.type === 'conditions') params[field.key] = [];
    else if (field.default) params[field.key] = field.default;
    else if (field.input === 'select' || field.input === 'suggest' || field.input === 'datetime') params[field.key] = '';
    else params[field.key] = field.type === 'int' ? '1' : '';
  }
  return params;
}

/** 基点或倍数转为输入框的原文, 去掉多余的 0 */
export function ratioInput(value: number, decimals: number): string {
  return String(value / 10 ** decimals);
}

/** 已保存的一个参数转为编辑中的草稿, 按字段的类型转换; 没有对应字段的原样转为文本 */
function toParamDraft(field: ConditionField | undefined, value: ParamValue, ctx: DraftContext): ParamDraft {
  if (typeof value === 'boolean') return value;
  switch (field?.type) {
    case 'int_list':
    case 'string_list':
      return Array.isArray(value) ? value.map(String) : [];
    case 'bundle':
      return toBundleDrafts(value as Bundle, ctx.currencies);
    case 'bundle_list':
      return Array.isArray(value) ? (value as Bundle[]).map((b) => toBundleDrafts(b, ctx.currencies)) : [];
    case 'conditions':
      return Array.isArray(value) ? toConditionDrafts(value as Condition[], ctx.types ?? [], ctx.currencies) : [];
  }
  if (typeof value === 'number' && field?.unit === 'bp') return ratioInput(value, PERCENT_DECIMALS);
  if (typeof value === 'string' && value && field?.input === 'datetime') return toLocalInput(value);
  return typeof value === 'object' ? '' : String(value);
}

/** 已保存的参数转为编辑中的草稿, 基点参数转为百分比, 时间转为本地时间; 没有保存或留空的参数有默认值时按默认值展示, 如之后才新增的统计周期 */
export function toParamsDraft(fields: ConditionField[], params: Record<string, ParamValue> | undefined, ctx: DraftContext = {currencies: []}): ParamsDraft {
  const draft: ParamsDraft = Object.fromEntries(
      Object.entries(params ?? {}).map(([key, value]) => [key, toParamDraft(fields.find((f) => f.key === key), value, ctx)]),
  );
  for (const field of fields) {
    if (field.default && (draft[field.key] === undefined || draft[field.key] === '')) draft[field.key] = field.default;
  }
  return draft;
}

/** 按字段转换参数, 校验不通过时中断提交; row 为提示的前缀, 如 "入场条件第 1 条" */
export function fromParamsDraft(fields: ConditionField[], draft: ParamsDraft, row: string, ctx: DraftContext = {currencies: []}): Condition['params'] {
  const params: Condition['params'] = {};
  for (const field of fields) {
    const value = draft[field.key];
    const name = `${row}: ${field.label}`;
    if (field.type === 'bool') {
      params[field.key] = value === true;
      continue;
    }
    if (field.type === 'int_list' || field.type === 'string_list') {
      const picked = Array.isArray(value) ? (value as string[]) : [];
      check(field.required && !picked.length ? `${row}: 请选择${field.label}` : null);
      params[field.key] = field.type === 'int_list' ? picked.map(Number) : picked;
      continue;
    }
    if (field.type === 'bundle') {
      params[field.key] = fromBundleDrafts(Array.isArray(value) ? (value as BundleDraft[]) : [], ctx.currencies, name);
      continue;
    }
    if (field.type === 'bundle_list') {
      const groups = Array.isArray(value) ? (value as BundleDraft[][]) : [];
      params[field.key] = groups.map((drafts, i) => fromBundleDrafts(drafts, ctx.currencies, `${name}第 ${i + 1} 组`));
      continue;
    }
    if (field.type === 'conditions') {
      params[field.key] = fromConditionDrafts(Array.isArray(value) ? (value as ConditionDraft[]) : [], ctx.types ?? [], name, ctx.currencies);
      continue;
    }
    const text = typeof value === 'string' ? value.trim() : '';
    if (field.type === 'int') {
      check(field.input === 'select' && !text ? `${row}: 请选择${field.label}` : null);
      const percent = field.unit === 'bp';
      const number = parseAmount(text, percent ? PERCENT_DECIMALS : 0);
      check(number === null ? `${name}${percent ? `需为百分比, 最多 ${PERCENT_DECIMALS} 位小数` : '需为整数'}` : null);
      params[field.key] = number ?? 0;
    } else if (field.input === 'datetime') {
      const time = text ? fromLocalInput(text) : '';
      check(time === null ? `${name}不是有效的时间` : null);
      check(field.required && !time ? `${row}: 请填写${field.label}` : null);
      params[field.key] = time ?? '';
    } else {
      check(field.required && !text ? `${row}: 请${field.input === 'select' ? '选择' : '填写'}${field.label}` : null);
      params[field.key] = text;
    }
  }
  return params;
}

/** 新建一个条件, 参数按字段类型给默认值 */
export function newConditionDraft(type: ConditionType): ConditionDraft {
  return {key: nanoid(), type: type.type, params: newParamsDraft(type.fields)};
}

/** 已保存的条件转为编辑中的条件, 按条件类型的字段转换参数 */
export function toConditionDrafts(conditions: Condition[], types: ConditionType[], currencies: Currency[]): ConditionDraft[] {
  return conditions.map((c) => ({
    key: nanoid(),
    type: c.type,
    params: toParamsDraft(types.find((t) => t.type === c.type)?.fields ?? [], c.params, {currencies, types}),
  }));
}

/** 转为请求中的条件, 按条件类型的字段转换参数; 未知类型的条件不能保存, 需要先删掉或改成已有的类型 */
export function fromConditionDrafts(drafts: ConditionDraft[], types: ConditionType[], label: string, currencies: Currency[]): Condition[] {
  return drafts.map((d, i) => {
    const type = types.find((t) => t.type === d.type);
    const row = `${label}第 ${i + 1} 条`;
    check(!d.type ? `${row}: 请选择条件类型` : null);
    check(!type ? `${row}: 未知的条件类型 ${d.type}` : null);
    return {type: d.type, params: fromParamsDraft(type?.fields ?? [], d.params, row, {currencies, types})};
  });
}
