import {Snowflake} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount} from '@/lib/utils';
import {CurrencyIcon} from './CurrencyIcon';

/**
 * 单个货币的余额卡片
 */
export function BalanceCard({balance: b}: { balance: BalanceItem }) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <CurrencyIcon currency={b.currency} className="size-5" />
        <span>{b.name}</span>
        {!b.is_enabled && <Badge variant="outline">已停用</Badge>}
      </div>
      <span className="font-mono text-3xl font-bold tabular-nums">{formatAmount(b.balance, b.decimals)}</span>
      {b.frozen > 0 && (
        <span className="flex items-center gap-1 text-xs text-muted-foreground" title="冻结中的余额(如竞拍押金), 暂不可用">
          <Snowflake className="size-3" />
          冻结 {formatAmount(b.frozen, b.decimals)}
        </span>
      )}
    </div>
  );
}
