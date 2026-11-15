import type {CratePrize} from '@/lib/services/crate';
import type {RewardLine} from '@/lib/services/reward';
import {formatAmount} from '@/lib/utils';

/** 滚动条上一格的宽度和间距, 像素; 位移按它们计算, 不随屏幕缩放 */
export const TILE_WIDTH = 112;
export const TILE_GAP = 8;

/** 开启前指针对准第几格, 它左边的几格用来填满窗口 */
export const START_INDEX = 4;
/** 抽中的一项放在第几格 */
export const WIN_INDEX = 56;
/** 滚动条的总格数, 抽中的一项右边留几格填满窗口 */
export const REEL_LENGTH = WIN_INDEX + START_INDEX + 1;

/** 滚动的时长, 秒 */
export const SPIN_SECONDS = 6;
/** 滚动的缓动: 起步很快, 越来越慢地停下 */
export const SPIN_EASE = [0.12, 0.78, 0.14, 1] as const;

/** 停下时指针偏离抽中那一格中心的最大距离, 占半格宽度的比例; 小于 1, 不会停在两格之间 */
const MAX_JITTER = 0.8;

/** 滚动条的位移: 指针对准第 index 格的中心, jitter 为偏离中心的像素 */
export function reelOffset(index: number, jitter = 0): number {
  return -(index * (TILE_WIDTH + TILE_GAP) + TILE_WIDTH / 2 + jitter);
}

/** 停下时偏离中心的像素, 每次随机 */
export function randomJitter(): number {
  return (Math.random() * 2 - 1) * (TILE_WIDTH / 2) * MAX_JITTER;
}

/** 按公示的概率随机选一项, 只用于填充滚动条 */
function samplePrize(prizes: CratePrize[]): CratePrize {
  let n = Math.random();
  for (const prize of prizes) {
    n -= prize.probability;
    if (n < 0) return prize;
  }
  return prizes[prizes.length - 1];
}

/** 填充用的一格, 数量在这一项的范围内随机 */
function fillerLine(prize: CratePrize): RewardLine {
  return {...prize, amount: prize.amount + Math.floor(Math.random() * (prize.max_amount - prize.amount + 1))};
}

/**
 * 开启后的滚动条: 第 WIN_INDEX 格是抽中的一项, 其余按公示的概率随机填充
 * 结果在开启时已经由后端结算, 滚动条只负责揭晓
 */
export function buildReel(prizes: CratePrize[], won: RewardLine): RewardLine[] {
  return Array.from({length: REEL_LENGTH}, (_, i) => (i === WIN_INDEX ? won : fillerLine(samplePrize(prizes))));
}

/** 开启前的滚动条: 箱子中的内容按顺序循环排列, 填满窗口 */
export function idleReel(prizes: CratePrize[]): RewardLine[] {
  return Array.from({length: START_INDEX * 2 + 1}, (_, i) => prizes[i % prizes.length]);
}

/** 一项的数量: 货币为金额, 物品为 ×数量; max 大于数量时显示范围 */
export function amountText(line: RewardLine, max = line.amount): string {
  const format = (n: number) => (line.currency ? formatAmount(n, line.decimals) : String(n));
  const text = max > line.amount ? `${format(line.amount)}~${format(max)}` : format(line.amount);
  return line.currency ? text : `×${text}`;
}

/** 概率按百分比展示, 最多 2 位小数; 不到 0.01% 时显示 <0.01% */
export function formatProbability(probability: number): string {
  const percent = probability * 100;
  if (percent > 0 && percent < 0.01) return '<0.01%';
  return `${Number(percent.toFixed(2))}%`;
}
