'use client';

import {cn} from 'cn';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities} from '@/components/common/site';
import type {Intel, PieceIntel} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 一件藏品的品质色点和品质名称 */
function QualityTag({quality, children}: { quality: number; children?: React.ReactNode }) {
  const q = useQualities().qualityOf(quality);
  return (
    <span className="inline-flex h-6 items-center gap-1.5 rounded-md border bg-card px-2 text-xs">
      <span className={cn('size-2 rounded-full', q.bar)} />
      <span className={q.text}>{q.name}</span>
      {children}
    </span>
  );
}

/**
 * 一个道具得到的情报, 按结果中有的字段展示, 不依赖效果代码, 新增的效果复用同样的字段即可
 * 藏品以扫描顺序编号, 与仓库中的格子对应
 */
export function IntelResult({intel, currency, decimals}: {
  intel: Intel;
  /** 出价货币, 展示价值区间 */
  currency: string;
  decimals: number;
}) {
  const {qualityOf} = useQualities();
  const pieces = intel.pieces ?? [];
  const counts = intel.counts ?? [];
  const hasRange = intel.value_max !== undefined;
  const min = intel.min_quality ? qualityOf(intel.min_quality).name : '';

  if (pieces.length === 0 && counts.length === 0 && !hasRange) {
    return <span className="text-xs text-muted-foreground">{min ? `没有${min}及以上的藏品` : '没有新的情报'}</span>;
  }

  return (
    <div className="flex flex-col gap-1.5">
      {pieces.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {pieces.map((p) => (
            <QualityTag key={p.scan_order} quality={p.quality}>
              <span className="font-mono text-muted-foreground tabular-nums">#{p.scan_order}</span>
              {p.name && <span>{p.name}</span>}
            </QualityTag>
          ))}
        </div>
      )}
      {counts.length > 0 && (
        <div className="flex flex-wrap items-center gap-1">
          {min && <span className="text-xs text-muted-foreground">{min}及以上</span>}
          {counts.map((c) => (
            <QualityTag key={c.quality} quality={c.quality}>
              <span className="font-mono tabular-nums">×{c.count}</span>
            </QualityTag>
          ))}
        </div>
      )}
      {hasRange && (
        <span className="flex items-center gap-1 text-xs">
          <span className="text-muted-foreground">仓库价值</span>
          <CurrencyIcon currency={currency} className="size-3.5" />
          <span className="font-mono tabular-nums">
            {formatAmount(intel.value_min ?? 0, decimals)} - {formatAmount(intel.value_max ?? 0, decimals)}
          </span>
        </span>
      )}
    </div>
  );
}

/** 合并道具看到的藏品, 按扫描顺序索引; 同一件看到过多次时保留带名称的 */
export function intelPieces(intels: Intel[]): Map<number, PieceIntel> {
  const known = new Map<number, PieceIntel>();
  for (const intel of intels) {
    for (const p of intel.pieces ?? []) {
      const prev = known.get(p.scan_order);
      if (!prev || (!prev.name && p.name)) known.set(p.scan_order, p);
    }
  }
  return known;
}
