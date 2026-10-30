import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import type {RewardLine} from '@/lib/services/reward';
import {formatAmount} from '@/lib/utils';

/** 一行的 key, 货币和物品不会重复 */
function lineKey(line: RewardLine) {
  return line.currency ?? `item:${line.item_id}`;
}

/**
 * 价格: 货币显示图标和金额, 物品显示名称和数量, 为空时显示免费
 */
export function PriceText({lines, className}: { lines: RewardLine[]; className?: string }) {
  const {qualityOf} = useQualities();
  if (!lines.length) return <span className={cn('font-medium text-quality-green', className)}>免费</span>;
  return (
    <span className={cn('flex flex-wrap items-center gap-x-3 gap-y-1', className)}>
      {lines.map((line) => (
        <span key={lineKey(line)} className="flex items-center gap-1">
          {line.currency ? (
            <>
              <CurrencyIcon currency={line.currency} className="size-4" />
              <span className="font-mono tabular-nums">{formatAmount(line.amount, line.decimals)}</span>
              <span className="text-muted-foreground">{line.name}</span>
            </>
          ) : (
            <>
              <span className={qualityOf(line.quality).text}>{line.name}</span>
              <span className="font-mono tabular-nums">×{line.amount}</span>
            </>
          )}
        </span>
      ))}
    </span>
  );
}

/**
 * 奖励或商品的内容: 每行一种货币或物品, 物品带图片和品质色
 */
export function RewardList({lines, className}: { lines: RewardLine[]; className?: string }) {
  const {qualityOf} = useQualities();
  return (
    <ul className={cn('flex flex-col gap-1.5 text-sm', className)}>
      {lines.map((line) => (
        <li key={lineKey(line)} className="flex items-center gap-2">
          <span className="flex size-7 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-background p-0.5">
            {line.currency ? <CurrencyIcon currency={line.currency} className="size-4" /> : <ItemImage item={line} />}
          </span>
          <span className={cn('min-w-0 flex-1 truncate', line.item_id && qualityOf(line.quality).text)} title={line.name}>
            {line.name}
          </span>
          <span className="font-mono tabular-nums">
            {line.currency ? formatAmount(line.amount, line.decimals) : `×${line.amount}`}
          </span>
        </li>
      ))}
    </ul>
  );
}
