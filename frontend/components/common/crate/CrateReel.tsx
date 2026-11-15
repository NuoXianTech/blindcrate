'use client';

import {motion, type MotionValue} from 'motion/react';
import {cn} from 'cn';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import type {RewardLine} from '@/lib/services/reward';
import {amountText, TILE_GAP, TILE_WIDTH} from './reel';

/**
 * 开箱滚动条的Props接口
 */
interface CrateReelProps {
  /** 从左到右的每一格 */
  tiles: RewardLine[];
  /** 滚动条的位移, 见 reelOffset */
  x: MotionValue<number>;
  /** 已经揭晓时抽中的是第几格, 这一格高亮, 其余变暗; 还没揭晓时为 null */
  wonIndex: number | null;
}

/**
 * 开箱滚动条: 一排物品从右向左滚过中间的指针, 停下时指针指着的就是抽中的一项
 * 只负责展示, 结果另有文字说明, 对读屏隐藏
 */
export function CrateReel({tiles, x, wonIndex}: CrateReelProps) {
  return (
    <div className="relative h-36 overflow-hidden rounded-xl border bg-background" aria-hidden>
      {/* 第一格的左边缘在窗口正中, 位移为负时向左滚动 */}
      <motion.div className="absolute top-3 bottom-3 left-1/2 flex" style={{x, gap: TILE_GAP}}>
        {tiles.map((line, i) => (
          <PrizeTile key={i} line={line} won={wonIndex === i} dimmed={wonIndex !== null && wonIndex !== i} />
        ))}
      </motion.div>

      {/* 两侧渐隐 */}
      <div className="pointer-events-none absolute inset-y-0 left-0 w-16 bg-gradient-to-r from-background to-transparent" />
      <div className="pointer-events-none absolute inset-y-0 right-0 w-16 bg-gradient-to-l from-background to-transparent" />

      {/* 指针 */}
      <div className="pointer-events-none absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 bg-primary shadow-[0_0_10px_var(--color-primary)]" />
      <div className="pointer-events-none absolute top-0 left-1/2 -translate-x-1/2 border-x-[6px] border-t-8 border-x-transparent border-t-primary" />
      <div className="pointer-events-none absolute bottom-0 left-1/2 -translate-x-1/2 border-x-[6px] border-b-8 border-x-transparent border-b-primary" />
    </div>
  );
}

/**
 * 滚动条上的一格: 图片、名称和底部的品质色条, 货币用强调色
 */
function PrizeTile({line, won, dimmed}: { line: RewardLine; won: boolean; dimmed: boolean }) {
  const q = useQualities().qualityOf(line.quality);
  const bar = line.currency ? 'bg-primary' : q.bar;
  // 物品只有 1 个时不标数量
  const amount = line.currency || line.amount > 1 ? amountText(line) : null;
  return (
    <div
      style={{width: TILE_WIDTH}}
      className={cn(
          'relative flex h-full shrink-0 flex-col overflow-hidden rounded-lg border bg-card transition-[opacity,scale,box-shadow] duration-300',
          dimmed && 'opacity-40',
          won && 'z-10 scale-105 border-primary glow',
      )}
    >
      {/* 品质色从底部向上晕开 */}
      <span className={cn('pointer-events-none absolute inset-0 opacity-20 [mask-image:linear-gradient(to_top,black,transparent_75%)]', bar)} />
      <span className="relative flex min-h-0 flex-1 items-center justify-center p-2">
        {line.currency ? <CurrencyIcon currency={line.currency} className="size-10" /> : <ItemImage item={line} />}
      </span>
      {amount && (
        <span className="absolute top-1 right-1 rounded bg-background/80 px-1 font-mono text-[10px] tabular-nums">{amount}</span>
      )}
      <span className={cn('relative truncate px-1.5 pb-1.5 text-center text-xs', line.item_id && q.text)}>{line.name}</span>
      <span className={cn('relative h-1 shrink-0', bar)} />
    </div>
  );
}
