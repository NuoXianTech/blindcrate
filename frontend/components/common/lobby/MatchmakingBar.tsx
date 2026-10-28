'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {LoaderCircle, Swords} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {matchHref} from '@/lib/access';
import {modeLabel} from '@/lib/auction';
import type {MatchmakingStatus} from '@/lib/services/match';

/** 排队时长, 如 0:07 */
function formatElapsed(ms: number) {
  const seconds = Math.max(0, Math.floor(ms / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
}

/**
 * 大厅顶部的匹配状态: 有进行中的对局时提示回到对局, 排队时显示已等待的时间和取消按钮
 */
export function MatchmakingBar({status, venueName, leaving, onLeave}: {
  status: MatchmakingStatus;
  /** 排队的地图名称, 地图已不在列表中时为空 */
  venueName?: string;
  leaving: boolean;
  onLeave: () => void;
}) {
  const [now, setNow] = useState(() => Date.now());
  const queued = status.queue !== null;

  useEffect(() => {
    if (!queued) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [queued]);

  if (status.match_id) {
    return (
      <div className="flex items-center justify-between gap-3 rounded-xl border border-primary/40 bg-primary/10 px-4 py-3">
        <span className="flex items-center gap-2 text-sm">
          <Swords className="size-4 text-primary" />
          你有一局正在进行
        </span>
        <Button size="sm" asChild>
          <Link href={matchHref(status.match_id)}>回到对局</Link>
        </Button>
      </div>
    );
  }

  if (!status.queue) return null;
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border bg-card px-4 py-3">
      <span className="flex flex-wrap items-center gap-2 text-sm">
        <LoaderCircle className="size-4 animate-spin text-primary" />
        正在匹配 {venueName} · {modeLabel(status.queue.mode)}
        <span className="font-mono text-muted-foreground tabular-nums">
          {formatElapsed(now - new Date(status.queue.joined_at).getTime())}
        </span>
      </span>
      <Button size="sm" variant="outline" disabled={leaving} onClick={onLeave}>
        {leaving && <LoaderCircle className="animate-spin" />}
        取消
      </Button>
    </div>
  );
}
