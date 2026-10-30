'use client';

import {useEffect} from 'react';
import {Clock} from 'lucide-react';
import {useCountdown} from '@/hooks/use-countdown';
import {resetText} from './constants';

/** 倒计时的刷新间隔, 文字最细到秒 */
const TICK_MS = 1000;

/** 到点后服务器还没进入新周期(客户端时钟偏快或请求失败)时, 每隔这么久再拉一次 */
const RETRY_MS = 5000;

/**
 * 周期重置的倒计时, 到点后通知刷新任务, 直到拿到新的重置时间
 */
export function ResetCountdown({at, onReset}: { at: string; onReset: () => void }) {
  const remaining = useCountdown(at, TICK_MS);
  const reached = remaining === 0;

  // 拿到新的重置时间后 at 变化, reached 变回 false, 停止重试
  useEffect(() => {
    if (!reached) return;
    onReset();
    const timer = setInterval(onReset, RETRY_MS);
    return () => clearInterval(timer);
  }, [reached, onReset]);

  return (
    <span className="flex items-center gap-1.5 text-sm text-muted-foreground tabular-nums">
      <Clock className="size-4" />
      {reached || remaining === null ? '正在重置' : resetText(remaining)}
    </span>
  );
}
