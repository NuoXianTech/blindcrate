'use client';

import {useEffect, useState} from 'react';

/** 距离截止时间的剩余毫秒, 默认每 250 毫秒刷新; 没有截止时间时为 null */
export function useCountdown(endsAt: string | null, interval = 250): number | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!endsAt) return;
    const timer = setInterval(() => setNow(Date.now()), interval);
    return () => clearInterval(timer);
  }, [endsAt, interval]);

  if (!endsAt) return null;
  return Math.max(0, new Date(endsAt).getTime() - now);
}
