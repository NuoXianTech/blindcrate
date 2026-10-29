import {nanoid} from 'nanoid';
import {check} from '@/hooks/use-submit';
import type {Condition, ConditionField, ConditionType, Currency} from '@/lib/services/admin';
import type {Bundle} from '@/lib/services/auction';
import {amountInput, parseAmount} from '@/lib/utils';

/** 物品的数量上限, 与后端 reward 的校验一致 */
const MAX_ITEM_QUANTITY = 1_000_000;
/** 货币的金额上限, 最小单位 */
const MAX_CURRENCY_AMOUNT = 1_000_000_000_000;

/** 编辑中的一行货币或物品, 数量保留输入框里的原文 */
export interface BundleDraft {
  key: string;
  /** 物品行的物品 ID, 货币行为 null */
  itemID: number | null;
  /** 货币行的货币代码, 物品行为空 */
  currency: string;
  amount: string;
}

/** 编辑中的一组参数, 数值保留输入框里的原文 */
export type ParamsDraft = Record<string, string | boolean>;

/** 编辑中的一个条件, 数值参数保留输入框里的原文 */
export interface ConditionDraft {
  key: string;
  type: string;
  params: ParamsDraft;
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

/** 按字段类型给参数默认值, 条件和物品的使用效果共用 */
export function newParamsDraft(fields: ConditionField[], currencies: Currency[]): ParamsDraft {
  const params: ParamsDraft = {};
  for (const field of fields) {
    if (field.type === 'bool') params[field.key] = false;
    else if (field.input === 'currency') params[field.key] = currencies[0]?.code ?? '';
    else if (field.input === 'scope') params[field.key] = 'all';
    else if (field.input === 'metric') params[field.key] = field.options?.[0]?.value ?? '';
    else if (field.input === 'item' || field.input === 'quality') params[field.key] = '';
    else params[field.key] = field.type === 'int' ? '1' : '';
  }
  return params;
}

/** 已保存的参数转为输入框里的原文 */
export function toParamsDraft(params: Record<string, string | number | boolean> | undefined): ParamsDraft {
  return Object.fromEntries(
      Object.entries(params ?? {}).map(([key, value]) => [key, typeof value === 'boolean' ? value : String(value)]),
  );
}

/** 按字段转换参数, 校验不通过时中断提交; row 为提示的前缀, 如 "入场条件第 1 条" */
export function fromParamsDraft(fields: ConditionField[], draft: ParamsDraft, row: string): Condition['params'] {
  const params: Condition['params'] = {};
  for (const field of fields) {
    const value = draft[field.key];
    if (field.type === 'bool') {
      params[field.key] = value === true;
      continue;
    }
    const text = typeof value === 'string' ? value.trim() : '';
    if (field.type === 'int') {
      check((field.input === 'item' || field.input === 'quality') && !text ? `${row}: 请选择${field.label}` : null);
      const number = parseAmount(text);
      check(number === null ? `${row}: ${field.label}需为整数` : null);
      params[field.key] = number ?? 0;
    } else {
      check(!text ? `${row}: 请填写${field.label}` : null);
      params[field.key] = text;
    }
  }
  return params;
}

/** 新建一个条件, 参数按字段类型给默认值 */
export function newConditionDraft(type: ConditionType, currencies: Currency[]): ConditionDraft {
  return {key: nanoid(), type: type.type, params: newParamsDraft(type.fields, currencies)};
}

export function toConditionDrafts(conditions: Condition[]): ConditionDraft[] {
  return conditions.map((c) => ({key: nanoid(), type: c.type, params: toParamsDraft(c.params)}));
}

/** 转为请求中的条件, 按条件类型的字段转换参数; 未知类型的参数原样提交, 由后端校验 */
export function fromConditionDrafts(drafts: ConditionDraft[], types: ConditionType[], label: string): Condition[] {
  return drafts.map((d, i) => {
    const type = types.find((t) => t.type === d.type);
    const row = `${label}第 ${i + 1} 条`;
    check(!d.type ? `${row}: 请选择条件类型` : null);
    if (!type) return {type: d.type, params: d.params};
    return {type: d.type, params: fromParamsDraft(type.fields, d.params, row)};
  });
}
