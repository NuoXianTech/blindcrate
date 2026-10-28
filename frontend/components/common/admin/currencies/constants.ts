import type {Currency, CurrencyRequest} from '@/lib/services/admin';

/** 与后端 wallet.ValidateCurrency 和请求的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const MAX_DECIMALS = 6;
export const MAX_SORT_ORDER = 1_000_000;

/** 货币转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(c: Currency): CurrencyRequest {
  return {name: c.name, decimals: c.decimals, sort_order: c.sort_order, is_enabled: c.is_enabled};
}
