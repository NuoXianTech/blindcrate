'use client';

import {useEffect, useState} from 'react';
import {scanProgress, type ScanProgress} from '@/lib/auction';
import {AUCTION_OUTCOME, type LotView} from '@/lib/services/auction';

/** 扫描进度的刷新间隔, 毫秒 */
const SCAN_TICK_MS = 100;

/** 开箱扫描的状态 */
export interface Scan extends ScanProgress {
  /** 正在播放开箱扫描 */
  scanning: boolean;
  /** 全部扫完, 可以公开仓库价值、盈亏和名次; 还没结算时为 false */
  complete: boolean;
  /** 已扫完的藏品中出价货币的回收价之和 */
  value: number;
}

/**
 * 一个拍品的开箱扫描进度: 结算后按扫描顺序逐件扫描, 仓库价值随扫描累加, 全部扫完才算揭晓
 * 中途进入或重连时按结算后已过去的时间接着扫; 对局已结束或拍品已取消时视为已扫完
 */
export function useScan(lot: LotView, currency: string, finished: boolean): Scan {
  const settled = lot.outcome !== '';
  const scanFrom = lot.scan_from ? new Date(lot.scan_from).getTime() : null;
  const scanning = settled && !finished && lot.outcome !== AUCTION_OUTCOME.cancelled && scanFrom !== null;
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!scanning) return;
    const timer = setInterval(() => setNow(Date.now()), SCAN_TICK_MS);
    return () => clearInterval(timer);
  }, [scanning]);

  const pieces = lot.pieces ?? [];
  const progress = scanning ?
    scanProgress(pieces.map((p) => p.scan_ms), now - (scanFrom ?? now)) :
    {done: pieces.length, fraction: 1};
  const value = pieces
      .slice(0, progress.done)
      .reduce((sum, p) => sum + (p.sell_currency === currency ? p.sell_price : 0), 0);
  const complete = settled && progress.done >= pieces.length;
  return {...progress, scanning: scanning && !complete, complete, value};
}
