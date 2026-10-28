'use client';

import {motion, useReducedMotion} from 'motion/react';
import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {qualityOf} from '@/lib/quality';
import type {AuctionView} from '@/lib/services/auction';
import {formatAmount} from '@/lib/utils';

/** 开箱时每件藏品出现的间隔, 秒 */
const REVEAL_STAGGER = 0.12;

/**
 * 仓库: 出价时格子被迷雾遮住, 只公开件数和已公开的品质; 开箱后按位置摆出全部藏品
 */
export function Warehouse({view, decimals}: { view: AuctionView; decimals: number }) {
  const reduced = useReducedMotion();
  const {grid} = view.venue;
  const pieces = view.result?.pieces;
  const revealed = new Map(view.revealed.map((p) => [p.order, p.quality]));

  return (
    <div className="flex flex-col gap-3">
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

        {pieces ? (
          pieces.map((p, i) => {
            const q = qualityOf(p.quality);
            return (
              <motion.div
                key={i}
                className="absolute p-0.5"
                style={{
                  left: `${(p.x / grid.width) * 100}%`,
                  top: `${(p.y / grid.height) * 100}%`,
                  width: `${(p.width / grid.width) * 100}%`,
                  height: `${(p.height / grid.height) * 100}%`,
                }}
                initial={reduced ? false : {opacity: 0, scale: 0.6}}
                animate={{opacity: 1, scale: 1}}
                transition={{delay: i * REVEAL_STAGGER, duration: 0.35, ease: [0.22, 1, 0.36, 1]}}
                title={`${p.name} · ${q.label} · ${formatAmount(p.sell_price, decimals)}`}
              >
                <div className="relative flex size-full items-center justify-center overflow-hidden rounded-md border bg-card">
                  <span className={cn('absolute inset-x-0 top-0 h-1', q.bar)} />
                  <ItemImage item={p} className="p-1" />
                </div>
              </motion.div>
            );
          })
        ) : (
          <div className="absolute inset-0 flex items-center justify-center bg-background/40 backdrop-blur-sm">
            <span className="rounded-full border bg-card px-4 py-1.5 text-sm">
              仓库里有 <span className="font-mono font-bold tabular-nums">{view.piece_count}</span> 件藏品
            </span>
          </div>
        )}
      </div>

      {/* 情报: 每轮公开一件藏品的品质 */}
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
    </div>
  );
}
