'use client';

import {useEffect, useState} from 'react';
import {motion, useReducedMotion} from 'motion/react';
import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {scanProgress} from '@/lib/auction';
import {qualityOf} from '@/lib/quality';
import {AUCTION_OUTCOME, type AuctionPublicView} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 扫描进度的刷新间隔, 毫秒 */
const SCAN_TICK_MS = 100;

/** 格子在仓库中的位置, 按百分比 */
function cellStyle(c: { x: number; y: number; width: number; height: number }, grid: { width: number; height: number }) {
  return {
    left: `${(c.x / grid.width) * 100}%`,
    top: `${(c.y / grid.height) * 100}%`,
    width: `${(c.width / grid.width) * 100}%`,
    height: `${(c.height / grid.height) * 100}%`,
  };
}

/** 当前时间, 每隔 interval 毫秒刷新; interval 为 null 时不刷新 */
function useNow(interval: number | null) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (interval === null) return;
    const timer = setInterval(() => setNow(Date.now()), interval);
    return () => clearInterval(timer);
  }, [interval]);

  return now;
}

/**
 * 迷雾中的一件藏品: 只看得到占的格子, 已公开品质时顶部显示品质色
 */
function FogCell({quality, children}: { quality?: number; children?: React.ReactNode }) {
  return (
    <div className="relative size-full overflow-hidden rounded-md border border-dashed border-muted-foreground/40 bg-muted/70">
      {!!quality && <span className={cn('absolute inset-x-0 top-0 h-1', qualityOf(quality).bar)} />}
      {children}
    </div>
  );
}

/**
 * 仓库: 出价时只看得到每件藏品占的格子和已公开的品质;
 * 开箱后按扫描顺序逐件扫描, 扫完才显示品质、名称和回收价, 中途进入或重连时按已过去的时间接着扫
 */
export function Warehouse({view, decimals, finished}: {
  view: AuctionPublicView;
  decimals: number;
  /** 对局已结束, 直接显示全部藏品 */
  finished: boolean;
}) {
  const reduced = useReducedMotion();
  const {grid, bid_currency: currency} = view.venue;
  const result = view.result;
  const scanFrom = result?.scan_from ? new Date(result.scan_from).getTime() : null;
  const scanning = !!result && !finished && result.outcome !== AUCTION_OUTCOME.cancelled && scanFrom !== null;
  const now = useNow(scanning ? SCAN_TICK_MS : null);
  const revealed = new Map(view.revealed.map((p) => [p.order, p.quality]));

  const pieces = result?.pieces ?? [];
  const progress = scanning ?
    scanProgress(pieces.map((p) => p.scan_ms), now - (scanFrom ?? now)) :
    {done: pieces.length, fraction: 1};
  const scannedValue = pieces
      .slice(0, progress.done)
      .reduce((sum, p) => sum + (p.sell_currency === currency ? p.sell_price : 0), 0);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="text-muted-foreground">
          仓库里有 <span className="font-mono font-bold tabular-nums text-foreground">{view.piece_count}</span> 件藏品
        </span>
        {result && (
          <span className="flex items-center gap-3">
            <span className="text-muted-foreground">
              已扫描{' '}
              <span className="font-mono tabular-nums text-foreground">
                {progress.done}/{pieces.length}
              </span>
            </span>
            <span className="flex items-center gap-1 font-mono font-bold tabular-nums">
              <CurrencyIcon currency={currency} className="size-4" />
              {formatAmount(scannedValue, decimals)}
            </span>
          </span>
        )}
      </div>

      <div
        className="relative w-full overflow-hidden rounded-xl border bg-muted/30"
        style={{aspectRatio: `${grid.width} / ${grid.height}`}}
      >
        {/* 格子线 */}
        <div
          className="absolute inset-0 grid"
          style={{gridTemplateColumns: `repeat(${grid.width}, 1fr)`, gridTemplateRows: `repeat(${grid.height}, 1fr)`}}
          aria-hidden
        >
          {Array.from({length: grid.width * grid.height}, (_, i) => (
            <div key={i} className="border-[0.5px] border-border/40" />
          ))}
        </div>

        {result ?
          pieces.map((p, i) => {
            const style = cellStyle(p, grid);
            // 还没扫到的藏品不显示品质, 留到扫描时揭晓
            if (i > progress.done) {
              return (
                <div key={p.scan_order} className="absolute p-0.5" style={style}>
                  <FogCell />
                </div>
              );
            }
            if (i === progress.done) {
              const percent = `${progress.fraction * 100}%`;
              return (
                <div key={p.scan_order} className="absolute p-0.5" style={style}>
                  <FogCell>
                    <span className="absolute inset-0 rounded-md ring-2 ring-primary ring-inset" />
                    {!reduced && (
                      <>
                        <span className="absolute inset-x-0 top-0 bg-primary/15" style={{height: percent}} />
                        <span
                          className="absolute inset-x-0 h-0.5 bg-primary shadow-[0_0_8px_var(--color-primary)]"
                          style={{top: percent}}
                        />
                      </>
                    )}
                  </FogCell>
                </div>
              );
            }
            const q = qualityOf(p.quality);
            return (
              <motion.div
                key={p.scan_order}
                className="absolute p-0.5"
                style={style}
                initial={reduced || !scanning ? false : {opacity: 0, scale: 0.6}}
                animate={{opacity: 1, scale: 1}}
                transition={{duration: 0.35, ease: [0.22, 1, 0.36, 1]}}
                title={`${p.name} · ${q.label} · ${formatAmount(p.sell_price, decimals)}`}
              >
                <div className="relative flex size-full items-center justify-center overflow-hidden rounded-md border bg-card">
                  <span className={cn('absolute inset-x-0 top-0 h-1', q.bar)} />
                  <ItemImage item={p} className="p-1" />
                </div>
              </motion.div>
            );
          }) :
          view.footprints.map((f) => (
            <div key={f.scan_order} className="absolute p-0.5" style={cellStyle(f, grid)}>
              <FogCell quality={f.quality} />
            </div>
          ))}
      </div>

      {/* 情报: 每轮公开一件藏品的品质 */}
      {!result && (
        <div className="flex flex-col gap-1.5">
          <span className="text-xs text-muted-foreground">
            已公开 {view.revealed.length}/{view.piece_count} 件的品质
          </span>
          <div className="flex flex-wrap gap-1.5">
            {Array.from({length: view.piece_count}, (_, i) => {
              const quality = revealed.get(i + 1);
              if (quality === undefined) {
                return (
                  <span key={i} className="flex h-7 min-w-10 items-center justify-center rounded-md border border-dashed text-xs text-muted-foreground">
                    ?
                  </span>
                );
              }
              const q = qualityOf(quality);
              return (
                <span key={i} className="flex h-7 min-w-10 items-center gap-1.5 rounded-md border bg-card px-2 text-xs">
                  <span className={cn('size-2.5 rounded-full', q.bar)} />
                  <span className={q.text}>{q.label}</span>
                </span>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
