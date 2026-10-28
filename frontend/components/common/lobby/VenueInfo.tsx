import {Check, X} from 'lucide-react';
import {cn} from 'cn';
import {CurrencyIcon} from '@/components/common/wallet';
import type {Venue} from '@/lib/services/venue';
import {formatAmount} from '@/lib/utils';

/**
 * 入场代价: 货币显示金额, 物品显示数量, 没有代价时显示免费
 */
export function EntryCost({venue: v, decimals}: { venue: Venue; decimals: number }) {
  if (!v.entry_cost.length) return <span className="text-muted-foreground">免费</span>;
  return (
    <span className="flex flex-wrap gap-x-3 gap-y-1">
      {v.entry_cost.map((line) => (
        <span key={line.currency ?? `item:${line.item_id}`} className="flex items-center gap-1">
          {line.currency && <CurrencyIcon currency={line.currency} className="size-4" />}
          <span className="font-mono tabular-nums">
            {line.currency ? formatAmount(line.amount, line.currency === v.bid_currency ? decimals : 0) : `×${line.amount}`}
          </span>
          <span className="text-muted-foreground">{line.name}</span>
        </span>
      ))}
    </span>
  );
}

/**
 * 入场条件: 逐条显示是否满足, 未满足的标红
 */
export function RequirementList({venue: v}: { venue: Venue }) {
  return (
    <ul className="flex flex-col gap-1 text-sm">
      {v.requirements.map((r, i) => (
        <li key={i} className={cn('flex items-center gap-1.5', r.met ? 'text-muted-foreground' : 'text-destructive')}>
          {r.met ? <Check className="size-4 shrink-0" /> : <X className="size-4 shrink-0" />}
          {r.text}
        </li>
      ))}
    </ul>
  );
}
