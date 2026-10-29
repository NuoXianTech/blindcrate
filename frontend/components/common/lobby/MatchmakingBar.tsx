'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {Bot, LoaderCircle, Swords, Users} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {matchHref} from '@/lib/access';
import {modeLabel, parseMode} from '@/lib/auction';
import type {MatchmakingStatus} from '@/lib/services/match';

/** 排队时长, 如 0:07 */
function formatElapsed(ms: number) {
  const seconds = Math.max(0, Math.floor(ms / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
}

/**
 * 大厅顶部的匹配状态: 有进行中的对局时提示回到对局, 排队时显示已等待的时间、队列人数和取消按钮
 * 排位只有自己一个人在等时不会补人机开局, 提示可以先去人机模式
 */
export function MatchmakingBar({status, venueName, leaving, onLeave, onSwitchPve}: {
  status: MatchmakingStatus;
  /** 排队的地图名称, 地图已不在列表中时为空 */
  venueName?: string;
  leaving: boolean;
  onLeave: () => void;
  /** 改打这张地图的人机, 地图没有开放人机时为空 */
  onSwitchPve?: () => void;
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
  const {queue} = status;
  const alone = parseMode(queue.mode)?.kind === 'ranked' && queue.count <= 1;
  return (
    <div className="flex flex-col gap-2 rounded-xl border bg-card px-4 py-3">
      <div className="flex items-center justify-between gap-3">
        <span className="flex flex-wrap items-center gap-2 text-sm">
          <LoaderCircle className="size-4 animate-spin text-primary" />
          正在匹配 {venueName} · {modeLabel(queue.mode)}
          <span className="font-mono text-muted-foreground tabular-nums">
            {formatElapsed(now - new Date(queue.joined_at).getTime())}
          </span>
          <span className="flex items-center gap-1 text-muted-foreground" title="队列中的人数, 凑满时立即开局">
            <Users className="size-3.5" />
            <span className="font-mono tabular-nums">{queue.capacity ? `${queue.count}/${queue.capacity}` : queue.count}</span>
          </span>
        </span>
        <Button size="sm" variant="outline" disabled={leaving} onClick={onLeave}>
          {leaving && <LoaderCircle className="animate-spin" />}
          取消
        </Button>
      </div>
      {alone && (
        <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span>当前排位人少, 至少 2 人才能开局, 可以先去人机模式</span>
          {onSwitchPve && (
            <Button size="sm" variant="ghost" className="h-7" disabled={leaving} onClick={onSwitchPve}>
              <Bot />
              改打人机
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
