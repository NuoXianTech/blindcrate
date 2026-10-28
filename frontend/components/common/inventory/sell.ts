import type {UserItem} from '@/lib/services/inventory';

/** 出售确认框中的一种物品 */
export interface SellLine {
  item: UserItem;
  quantity: number;
}

/** 一种货币上的出售汇总, 实得为 amount - fee */
export interface SellTotal {
  currency: string;
  /** 卖出金额, 未扣手续费 */
  amount: number;
  fee: number;
}

/**
 * 按货币汇总卖出金额和手续费, 按货币代码排序
 * 与后端一致: 同一货币的金额先求和, 再乘以手续费率向下取整
 */
export function sellTotals(lines: SellLine[], feeRate: number): SellTotal[] {
  const amounts = new Map<string, number>();
  for (const {item, quantity} of lines) {
    amounts.set(item.sell_currency, (amounts.get(item.sell_currency) ?? 0) + item.sell_price * quantity);
  }
  return [...amounts.keys()].sort().map((currency) => {
    const amount = amounts.get(currency) ?? 0;
    return {currency, amount, fee: Math.floor((amount * feeRate) / 10000)};
  });
}

/** 手续费率(基点)的展示, 如 5% */
export function formatFeeRate(feeRate: number): string {
  return `${feeRate / 100}%`;
}
