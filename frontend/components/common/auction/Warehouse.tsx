'use client';

import {useEffect, useState} from 'react';
import {Eye, Megaphone} from 'lucide-react';
import {motion, useReducedMotion} from 'motion/react';
import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {useQualities} from '@/components/common/site';
import type {Scan} from '@/hooks/use-scan';
import {cellKey, formatRate} from '@/lib/auction';
import {AUCTION_PHASE, HOST_HINT, type AuctionPublicView, type PieceIntel} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 首轮扫出轮廓时扫描线走完整个仓库的时长, 秒 */
const OUTLINE_SWEEP_S = 1.6;
/** 道具扫出轮廓时扫描线走完一件的时长, 秒 */
const CELL_SWEEP_S = 0.6;
/** 道具扫出名称和外观时金色扫描线走完一件的时长, 秒 */
const GOLD_SWEEP_S = 0.9;
/** 播放过首轮扫描的对局, 存在 sessionStorage, 刷新和重连后不再播放 */
const SWEPT_KEY = 'auction:outline-swept';

/** 格子在仓库中的位置, 按百分比 */
function cellStyle(c: { x: number; y: number; width: number; height: number }, grid: { width: number; height: number }) {
  return {
    left: `${(c.x / grid.width) * 100}%`,
    top: `${(c.y / grid.height) * 100}%`,
    width: `${(c.width / grid.width) * 100}%`,
    height: `${(c.height / grid.height) * 100}%`,
  };
}

/** 这局是否已经播放过首轮扫描 */
function swept(matchId: number) {
  try {
    return (sessionStorage.getItem(SWEPT_KEY) ?? '').split(',').includes(String(matchId));
  } catch {
    return true;
  }
}

/** 记下这局已经播放过首轮扫描, 只保留最近几局 */
function markSwept(matchId: number) {
  try {
    const ids = (sessionStorage.getItem(SWEPT_KEY) ?? '').split(',').filter(Boolean);
    sessionStorage.setItem(SWEPT_KEY, [...ids.slice(-9), String(matchId)].join(','));
  } catch {
    // 无痕模式等不能写 sessionStorage 时每次进入都播放
  }
}

/**
 * 迷雾中的一件藏品: 只看得到轮廓(占的格子), 知道品质时顶部显示品质色
 */
function FogCell({quality, className, children}: { quality?: number; className?: string; children?: React.ReactNode }) {
  const {qualityOf} = useQualities();
  return (
    <div className={cn('relative size-full overflow-hidden rounded-md border border-dashed border-muted-foreground/40 bg-muted/70', className)}>
      {!!quality && <span className={cn('absolute inset-x-0 top-0 h-1', qualityOf(quality).bar)} />}
      {children}
    </div>
  );
}

/** 主持人指认这件藏品时的标记, 品质色为主持人说的品质, 可能说错 */
function HintMark({quality}: { quality: number }) {
  const q = useQualities().qualityOf(quality);
  return (
    <span className="absolute top-1 right-1 flex items-center" title={`主持人说这件是${q.name}, 可能说错`}>
      <Megaphone className={cn('size-3.5', q.text)} aria-label="主持人指认" />
    </span>
  );
}

/**
 * 道具扫出名称和外观的一件藏品: 金色扫描线扫过后显示外观、名称和品质, 只有自己看得到
 */
function NamedCell({intel, animate}: { intel: PieceIntel; animate: boolean }) {
  const q = useQualities().qualityOf(intel.quality);
  return (
    <div
      title={`仅自己可见: ${intel.name} · ${q.name}`}
      className="relative flex size-full items-center justify-center overflow-hidden rounded-md border border-primary/60 bg-card glow"
    >
      <span className={cn('absolute inset-x-0 top-0 z-10 h-1', q.bar)} />
      <motion.div
        className="absolute inset-0 flex flex-col items-center justify-center gap-0.5 p-1 text-center"
        initial={animate ? {opacity: 0} : false}
        animate={{opacity: 1}}
        transition={{delay: GOLD_SWEEP_S * 0.6, duration: 0.4}}
      >
        <ItemImage item={{name: intel.name ?? '', image_url: intel.image_url ?? ''}} className="min-h-0 flex-1 p-0.5" />
        <span className={cn('line-clamp-1 text-[10px] leading-tight', q.text)}>{intel.name}</span>
      </motion.div>
      <Eye className="absolute bottom-0.5 left-1 size-3 text-primary" aria-label="仅自己可见" />
      {animate && (
        <>
          <motion.span
            className="pointer-events-none absolute inset-x-0 top-0 bg-primary/20"
            initial={{height: '0%', opacity: 1}}
            animate={{height: '100%', opacity: 0}}
            transition={{duration: GOLD_SWEEP_S, ease: 'easeInOut'}}
          />
          <motion.span
            className="pointer-events-none absolute inset-x-0 h-0.5 bg-primary shadow-[0_0_8px_var(--color-primary)]"
            initial={{top: '0%', opacity: 1}}
            animate={{top: '100%', opacity: 0}}
            transition={{duration: GOLD_SWEEP_S, ease: 'easeInOut'}}
          />
        </>
      )}
    </div>
  );
}

/** 出价时看到的一件藏品: 首轮公开的轮廓, 或道具和情报私下扫出的 */
interface SeenCell {
  key: string;
  x: number;
  y: number;
  width: number;
  height: number;
  /** 首轮扫出的轮廓, 所有人都看得到 */
  public: boolean;
  /** 自己用道具或情报看到的, 只有自己看得到 */
  intel?: PieceIntel;
  /** 主持人指认这件时说的品质 */
  hint?: number;
}

/** 出价时仓库里看得到的藏品: 首轮扫出的轮廓加上自己的情报, 按位置从上到下、从左到右 */
function seenCells(view: AuctionPublicView, intel?: Map<string, PieceIntel>): SeenCell[] {
  const cells = new Map<string, SeenCell>();
  for (const f of view.footprints) {
    cells.set(cellKey(f), {key: cellKey(f), ...f, public: true});
  }
  for (const [key, p] of intel ?? []) {
    const prev = cells.get(key);
    cells.set(key, {key, x: p.x, y: p.y, width: p.width, height: p.height, public: !!prev?.public, intel: p});
  }
  for (const h of view.host?.hints ?? []) {
    const cell = h.kind === HOST_HINT.piece ? cells.get(cellKey(h)) : undefined;
    if (cell) cell.hint = h.quality;
  }
  return [...cells.values()].sort((a, b) => a.y - b.y || a.x - b.x);
}

/**
 * 仓库: 只公开仓库大小和第 1 轮扫出的几件藏品的轮廓, 件数、品质和名称直到开箱都不公开;
 * 道具和情报扫出的藏品只有自己看得到, 扫出名称和外观时播放金色扫描;
 * 开箱后按扫描顺序逐件扫描, 扫到时才显示轮廓、外观和回收价, 仓库价值随扫描累加
 */
export function Warehouse({view, decimals, scan, intel}: {
  view: AuctionPublicView;
  decimals: number;
  /** 开箱扫描的进度, 见 useScan */
  scan: Scan;
  /** 自己用道具和情报看到的藏品, 按位置索引, 见 cellKey */
  intel?: Map<string, PieceIntel>;
}) {
  const reduced = useReducedMotion();
  const {qualityOf} = useQualities();
  const {grid, bid_currency: currency} = view.venue;
  const result = view.result;
  const cells = seenCells(view, intel);
  // 第 1 轮刚开局时播放一次扫描线, 轮廓随扫描线依次出现
  const [sweep] = useState(() => !reduced && view.phase === AUCTION_PHASE.bidding && view.round === 1 && !swept(view.match_id));
  // 进入时已经扫出的轮廓和名称不再播放扫描, 之后用道具新扫出的才播放
  const [outlined] = useState(() => new Set(cells.map((c) => c.key)));
  const [named] = useState(() => new Set(cells.filter((c) => c.intel?.name).map((c) => c.key)));

  useEffect(() => {
    if (sweep) markSwept(view.match_id);
  }, [sweep, view.match_id]);

  const pieces = result?.pieces ?? [];
  const known = new Set(cells.map((c) => c.key));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="text-muted-foreground">
          仓库 <span className="font-mono font-bold tabular-nums text-foreground">{grid.width}×{grid.height}</span>
          {result && scan.complete && (
            <>
              {' '}· 共 <span className="font-mono font-bold tabular-nums text-foreground">{pieces.length}</span> 件藏品
            </>
          )}
        </span>
        {result && (
          <span className="flex items-center gap-3">
            {!scan.complete && (
              <span className="text-muted-foreground">
                已扫描 <span className="font-mono tabular-nums text-foreground">{scan.done}</span> 件
              </span>
            )}
            <span className="flex items-center gap-1 font-mono font-bold tabular-nums">
              <CurrencyIcon currency={currency} className="size-4" />
              {formatAmount(scan.value, decimals)}
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
            // 还没扫到的藏品不显示, 出价时看到过轮廓的仍显示轮廓
            if (i > scan.done) {
              return known.has(cellKey(p)) ? (
                <div key={p.scan_order} className="absolute p-0.5" style={style}>
                  <FogCell />
                </div>
              ) : null;
            }
            if (i === scan.done) {
              const percent = `${scan.fraction * 100}%`;
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
                initial={reduced || !scan.scanning ? false : {opacity: 0, scale: 0.6}}
                animate={{opacity: 1, scale: 1}}
                transition={{duration: 0.35, ease: [0.22, 1, 0.36, 1]}}
                title={`${p.name} · ${q.name} · ${formatAmount(p.sell_price, decimals)}`}
              >
                <div className="relative flex size-full items-center justify-center overflow-hidden rounded-md border bg-card">
                  <span className={cn('absolute inset-x-0 top-0 h-1', q.bar)} />
                  <ItemImage item={p} className="p-1" />
                </div>
              </motion.div>
            );
          }) :
          cells.map((c) => {
            const style = cellStyle(c, grid);
            if (c.intel?.name) {
              return (
                <div key={`${c.key}:named`} className="absolute p-0.5" style={style}>
                  <NamedCell intel={c.intel} animate={!reduced && !named.has(c.key)} />
                  {c.hint !== undefined && <HintMark quality={c.hint} />}
                </div>
              );
            }
            const quality = c.intel?.quality;
            const title = quality ?
              `仅自己可见: ${qualityOf(quality).name}` :
              c.public ? '首轮扫出的轮廓, 品质和名称开箱前不公开' : '仅自己可见的轮廓, 品质和名称未知';
            const scanned = !reduced && !c.public && !outlined.has(c.key);
            return (
              <motion.div
                key={c.key}
                className="absolute p-0.5"
                style={style}
                title={title}
                initial={sweep && c.public ? {opacity: 0, scale: 0.92} : !reduced && !c.public ? {opacity: 0} : false}
                animate={{opacity: 1, scale: 1}}
                transition={{delay: sweep && c.public ? (c.y / grid.height) * OUTLINE_SWEEP_S : 0, duration: 0.3}}
              >
                <FogCell quality={quality} className={cn(!c.public && 'border-primary/50')}>
                  {!c.public && <Eye className="absolute bottom-0.5 left-1 size-3 text-muted-foreground" aria-label="仅自己可见" />}
                  {scanned && (
                    <motion.span
                      className="pointer-events-none absolute inset-x-0 h-0.5 bg-foreground/60 shadow-[0_0_8px_var(--color-foreground)]"
                      initial={{top: '0%', opacity: 1}}
                      animate={{top: '100%', opacity: 0}}
                      transition={{duration: CELL_SWEEP_S, ease: 'easeInOut'}}
                    />
                  )}
                </FogCell>
                {c.hint !== undefined && <HintMark quality={c.hint} />}
              </motion.div>
            );
          })}

        {/* 首轮扫描线: 从上到下扫过整个仓库, 扫到的轮廓依次出现 */}
        {!result && sweep && (
          <motion.span
            className="pointer-events-none absolute inset-x-0 h-0.5 bg-foreground/60 shadow-[0_0_10px_var(--color-foreground)]"
            initial={{top: '0%', opacity: 1}}
            animate={{top: '100%', opacity: [1, 1, 0]}}
            transition={{duration: OUTLINE_SWEEP_S, ease: 'linear'}}
            aria-hidden
          />
        )}
      </div>

      {!result && (
        <span className="text-xs text-muted-foreground">
          {view.footprints.length > 0 ?
            `首轮扫出 ${view.footprints.length} 件藏品的轮廓, 每件有 ${formatRate(view.venue.preview_rate)} 的概率被扫出; ` :
            '首轮没有扫描; '}
          件数、品质和名称开箱前都不公开, 道具和情报扫出的只有你看得到
        </span>
      )}
    </div>
  );
}
