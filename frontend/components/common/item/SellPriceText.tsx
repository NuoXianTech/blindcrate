import {CurrencyIcon} from '@/components/common/wallet';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount} from '@/lib/utils';

/**
 * 物品的出售价格: 货币图标、金额和货币名称, 售价为 0 时显示不可出售
 */
export function SellPriceText({item, currencies}: {
  item: { sell_currency: string; sell_price: number };
  /** 货币列表, 用于展示货币名称和小数位; 找不到时显示货币代码 */
  currencies: BalanceItem[];
}) {
  if (item.sell_price <= 0) return <span className="text-muted-foreground">不可出售</span>;
  const currency = currencies.find((c) => c.currency === item.sell_currency);
  return (
    <span className="flex items-center gap-1.5">
      <CurrencyIcon currency={item.sell_currency} className="size-4" />
      <span className="font-mono tabular-nums">{formatAmount(item.sell_price, currency?.decimals ?? 0)}</span>
      <span className="text-muted-foreground">{currency?.name ?? item.sell_currency}</span>
    </span>
  );
}
