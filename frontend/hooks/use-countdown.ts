'use client';

import {useEffect, useState} from 'react';

/** 距离截止时间的剩余毫秒, 每 250 毫秒刷新; 没有截止时间时为 null */
export function useCountdown(endsAt: string | null): number | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!endsAt) return;
    const timer = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(timer);
  }, [endsAt]);

  if (!endsAt) return null;
  return Math.max(0, new Date(endsAt).getTime() - now);
}
