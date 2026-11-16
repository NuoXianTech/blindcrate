'use client';

import {cn} from 'cn';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities} from '@/components/common/site';
import {cellKey} from '@/lib/auction';
import {FACT_KIND, type Fact, type Intel, type PieceIntel, type UseView} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 品质色点和品质名称, 品质为 0(不知道品质)时显示 "轮廓" */
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

/** 情报中的一个条目: "[品质] 标签 内容" */
function FactRow({fact, currency, decimals}: { fact: Fact; currency: string; decimals: number }) {
  const {qualityOf} = useQualities();
  const counts = fact.counts ?? [];
  return (
    <span className="flex flex-wrap items-center gap-1 text-xs">
      <span className="text-muted-foreground">
        {fact.quality ? qualityOf(fact.quality).name : ''}{fact.label}
      </span>
      {fact.kind === FACT_KIND.text && <span>{fact.text}</span>}
      {fact.kind === FACT_KIND.range && (
        <span className="font-mono tabular-nums">
          {fact.min === fact.max ? fact.min : `${fact.min} - ${fact.max}`}{fact.unit && ` ${fact.unit}`}
        </span>
      )}
      {fact.kind === FACT_KIND.amount && (
        <>
          <CurrencyIcon currency={currency} className="size-3.5" />
          <span className="font-mono tabular-nums">
            {formatAmount(fact.min, decimals)} - {formatAmount(fact.max, decimals)}
          </span>
        </>
      )}
      {fact.kind === FACT_KIND.counts && (counts.length > 0 ?
        counts.map((c) => (
          <QualityTag key={c.quality} quality={c.quality}>
            <span className="font-mono tabular-nums">×{c.count}</span>
          </QualityTag>
        )) :
        <span>没有</span>)}
    </span>
  );
}

/**
 * 一个道具得到的情报: 格子和条目, 按内容的种类展示, 不依赖效果代码, 后端新增的效果不用改这里
 * 格子同时画在货仓里, 这里不编号, 只标出占的格子大小
 */
export function IntelResult({intel, currency, decimals}: {
  intel: Intel;
  /** 出价货币, 展示金额区间 */
  currency: string;
  decimals: number;
}) {
  const pieces = (intel.pieces ?? []).filter((p) => !p.empty);
  const empty = (intel.pieces ?? []).filter((p) => p.empty);
  const facts = intel.facts ?? [];
  // 看的是其他拍品时标出是哪个
  const lotNo = intel.pieces?.find((p) => p.lot_no)?.lot_no;

  if (pieces.length === 0 && empty.length === 0 && facts.length === 0) {
    return <span className="text-xs text-muted-foreground">没有新的情报</span>;
  }

  return (
    <div className="flex flex-col gap-1.5">
      {(pieces.length > 0 || empty.length > 0) && (
        <div className="flex flex-wrap items-center gap-1">
          {!!lotNo && <span className="text-xs text-muted-foreground">拍品 {lotNo}</span>}
          {pieces.map((p) => (
            <QualityTag key={cellKey(p)} quality={p.quality}>
              <span className="font-mono text-muted-foreground tabular-nums">{p.width}×{p.height}</span>
              {p.name && <span>{p.name}</span>}
              {p.note && <span className="text-primary">{p.note}</span>}
            </QualityTag>
          ))}
          {empty.length > 0 && (
            <span className="inline-flex h-6 items-center gap-1.5 rounded-md border bg-card px-2 text-xs">
              <span className="text-muted-foreground">空格子</span>
              <span className="font-mono tabular-nums">×{empty.length}</span>
            </span>
          )}
        </div>
      )}
      {facts.map((f, i) => <FactRow key={i} fact={f} currency={currency} decimals={decimals} />)}
    </div>
  );
}

/** 一个格子情报的详细程度: 名称 > 品质 > 只有轮廓 */
function detail(p: PieceIntel) {
  return p.name ? 2 : p.quality ? 1 : 0;
}

/**
 * 合并道具在一个拍品里看到的格子, 按位置索引, 见 cellKey; 同一格看到过多次时保留最详细的, 标注不丢
 * 格子属于它自己给出的拍品, 没给出时属于使用道具时的拍品
 */
export function intelPieces(uses: UseView[], lotNo: number): Map<string, PieceIntel> {
  const known = new Map<string, PieceIntel>();
  for (const use of uses) {
    for (const p of use.result.pieces ?? []) {
      if ((p.lot_no || use.lot_no) !== lotNo) continue;
      const prev = known.get(cellKey(p));
      if (!prev) {
        known.set(cellKey(p), p);
      } else if (!p.empty && !prev.empty) {
        const [more, less] = detail(p) > detail(prev) ? [p, prev] : [prev, p];
        known.set(cellKey(p), more.note || !less.note ? more : {...more, note: less.note});
      }
    }
  }
  return known;
}
