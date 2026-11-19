import type {Bundle, RewardLine} from '@/lib/services/reward';
import type {ShopItem, ShopLimitType} from '@/lib/services/shop';

/** 购买记录每页条数 */
export const ORDER_PAGE_SIZE = 20;

/** 没有页签的商品归入的页签 */
export const DEFAULT_TAB = '其他';

export const LIMIT_LABELS: Record<ShopLimitType, string> = {
  total: '每人限购',
  daily: '每天限购',
  season: '每赛季限购',
};

/** 商品中的行转回 Bundle, 用于提交看到的价格 */
export function toBundle(lines: RewardLine[]): Bundle {
  return {
    currencies: lines.flatMap((l) => (l.currency ? [{currency: l.currency, amount: l.amount}] : [])),
    items: lines.flatMap((l) => (l.item_id ? [{item_id: l.item_id, quantity: l.amount}] : [])),
  };
}

/** 每一行乘以份数 */
export function scaleLines(lines: RewardLine[], quantity: number): RewardLine[] {
  return lines.map((l) => ({...l, amount: l.amount * quantity}));
}

/** 不能购买的原因, 可以购买时为 null */
export function unavailableReason(item: ShopItem): string | null {
  if (!item.eligible) return '未满足条件';
  if (item.stock === 0) return '已售罄';
  if (item.limits.some((l) => l.bought >= l.limit)) return '已达限购';
  if (item.max_quantity === 0) return '当前无法购买';
  return null;
}

/** 商品的图片, 没有设置时用第一件商品的图片 */
export function imageOf(item: ShopItem): string {
  return item.image_url || item.goods.find((l) => l.image_url)?.image_url || '';
}
