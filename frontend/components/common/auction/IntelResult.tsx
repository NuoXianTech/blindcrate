'use client';

import {cn} from 'cn';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities} from '@/components/common/site';
import {cellKey} from '@/lib/auction';
import type {Intel, PieceIntel} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 一件藏品的品质色点和品质名称, 品质为 0(只扫出轮廓)时显示 "轮廓" */
function QualityTag({quality, children}: { quality: number; children?: React.ReactNode }) {
  const q = useQualities().qualityOf(quality);
  return (
    <span className="inline-flex h-6 items-center gap-1.5 rounded-md border bg-card px-2 text-xs">
      {quality ? (
        <>
          <span className={cn('size-2 rounded-full', q.bar)} />
          <span className={q.text}>{q.name}</span>
        </>
      ) : (
        <span className="text-muted-foreground">轮廓</span>
      )}
      {children}
    </span>
  );
}

/**
 * 一个道具得到的情报, 按结果中有的字段展示, 不依赖效果代码, 新增的效果复用同样的字段即可
 * 看到的藏品同时在仓库中扫出轮廓, 这里不编号, 只标出占的格子大小
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
            <QualityTag key={cellKey(p)} quality={p.quality}>
              <span className="font-mono text-muted-foreground tabular-nums">{p.width}×{p.height}</span>
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

/** 一件藏品情报的详细程度: 名称 > 品质 > 只有轮廓 */
function detail(p: PieceIntel) {
  return p.name ? 2 : p.quality ? 1 : 0;
}

/** 合并道具看到的藏品, 按位置索引, 见 cellKey; 同一件看到过多次时保留最详细的 */
export function intelPieces(intels: Intel[]): Map<string, PieceIntel> {
  const known = new Map<string, PieceIntel>();
  for (const intel of intels) {
    for (const p of intel.pieces ?? []) {
      const prev = known.get(cellKey(p));
      if (!prev || detail(p) > detail(prev)) known.set(cellKey(p), p);
    }
  }
  return known;
}
