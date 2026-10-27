'use client';

import {motion, useReducedMotion} from 'motion/react';
import {cn} from 'cn';

/** 7 级品质, 从低到高, 与 README 一致 */
const QUALITIES = [
  {label: '灰', className: 'bg-quality-gray'},
  {label: '绿', className: 'bg-quality-green'},
  {label: '蓝', className: 'bg-quality-blue'},
  {label: '紫', className: 'bg-quality-purple'},
  {label: '红', className: 'bg-quality-red'},
  {label: '金', className: 'bg-quality-gold'},
  {label: '炫彩', className: 'quality-rainbow'},
];

const LID_DELAY = 0.4;

/** 首页主视觉: 箱盖打开一次, 品质阶梯从箱口升起 */
export function CrateHero() {
  const reduced = useReducedMotion();
  // 减少动态效果时直接显示打开后的状态
  const t = (delay: number, duration: number) => (reduced ? {duration: 0} : {delay, duration, ease: [0.22, 1, 0.36, 1] as const});

  return (
    <div className="relative flex h-56 w-72 flex-col items-center justify-end" aria-hidden>
      {/* 箱口的光 */}
      <motion.div
        className="absolute bottom-16 h-40 w-56 rounded-full bg-quality-gold/25 blur-3xl"
        initial={{opacity: 0, scale: 0.6}}
        animate={{opacity: 1, scale: 1}}
        transition={t(LID_DELAY + 0.1, 0.9)}
      />

      {/* 品质阶梯: 越高品质越高 */}
      <div className="absolute bottom-[4.5rem] flex items-end gap-1.5">
        {QUALITIES.map((q, i) => (
          <motion.div
            key={q.label}
            title={`${q.label}色品质`}
            className={cn('w-5 rounded-t-sm', q.className)}
            style={{height: 24 + i * 12}}
            initial={{y: 40, opacity: 0}}
            animate={{y: 0, opacity: 1}}
            transition={t(LID_DELAY + 0.25 + i * 0.07, 0.6)}
          />
        ))}
      </div>

      {/* 箱盖 */}
      <motion.div
        className="absolute bottom-[4.25rem] left-4 h-5 w-64 origin-bottom-left rounded-md border border-primary/40 bg-secondary"
        initial={{rotate: 0}}
        animate={{rotate: -24}}
        transition={t(LID_DELAY, 0.7)}
      />

      {/* 箱体 */}
      <div className="relative z-10 flex h-[4.5rem] w-64 items-center justify-center rounded-b-lg border border-primary/40 bg-card">
        <div className="absolute inset-x-0 top-1/2 h-px bg-border" />
        <div className="relative rounded-sm border border-primary/50 bg-secondary px-2 py-1 text-primary">
          <div className="size-2 rounded-full bg-primary" />
        </div>
      </div>
    </div>
  );
}
