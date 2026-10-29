import {SHOP_ORDER_STATUS, type ShopOrderStatus} from '@/lib/services/shop';

/** 订单每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 商品筛选下拉框中最多列出的商品数 */
export const SHOP_ITEM_LIMIT = 100;

/** 订单状态的名称 */
export const STATUS_LABELS: Record<ShopOrderStatus, string> = {
  [SHOP_ORDER_STATUS.paid]: '已支付',
  [SHOP_ORDER_STATUS.refunded]: '已退款',
};

/** 日期输入框的值转为当天 0 点的 RFC3339, 按本地时区; days 为往后偏移的天数, 用于不包含的结束日期 */
export function dayStart(value: string, days = 0): string | undefined {
  if (!value) return undefined;
  const d = new Date(`${value}T00:00`);
  if (Number.isNaN(d.getTime())) return undefined;
  d.setDate(d.getDate() + days);
  return d.toISOString();
}
