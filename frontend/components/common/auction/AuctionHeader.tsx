'use client';

import Link from 'next/link';
import {ChevronLeft, Timer} from 'lucide-react';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {ROUTES} from '@/lib/access';
import {modeLabel} from '@/lib/auction';
import {AUCTION_PHASE, type AuctionView} from '@/lib/services/auction';
import {MATCH_STATUS} from '@/lib/services/match';

/** 剩余这么多毫秒时倒计时变红 */
const URGENT_MS = 5000;

/**
 * 拍卖桌顶部: 返回大厅、地图和模式、当前轮次、阶段倒计时
 */
export function AuctionHeader({view, finished, remaining}: {
  view: AuctionView;
  /** 对局已结束 */
  finished: boolean;
  /** 当前阶段剩余的毫秒, 不限时为 null */
  remaining: number | null;
}) {
  const {venue} = view;
  const extra = view.round - venue.rounds;

  let stage: string;
  if (finished) {
    stage = view.status === MATCH_STATUS.cancelled ? '已取消' : '已结束';
  } else if (view.phase === AUCTION_PHASE.reveal) {
    stage = '开箱';
  } else if (extra > 0) {
    stage = `加轮 ${extra}`;
  } else {
    stage = `第 ${view.round}/${venue.rounds} 轮`;
  }

  const showTimer = !finished && remaining !== null;
  const urgent = showTimer && view.phase === AUCTION_PHASE.bidding && remaining <= URGENT_MS;

  return (
    <header className="flex items-center gap-3 border-b px-4 py-3">
      <Button variant="ghost" size="icon-sm" asChild>
        <Link href={ROUTES.gameHome} aria-label="返回大厅">
          <ChevronLeft />
        </Link>
      </Button>
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="flex items-center gap-2">
          <span className="truncate font-bold">{venue.name}</span>
          <Badge variant="secondary">{modeLabel(view.mode)}</Badge>
          {view.self.is_practice && <Badge variant="outline" title="不扣入场代价, 不冻结资金, 也不发放藏品">练习局</Badge>}
        </span>
        <span className="text-xs text-muted-foreground">{stage}</span>
      </div>
      {showTimer && (
        <span
          className={cn(
              'flex items-center gap-1.5 rounded-full border px-3 py-1 font-mono text-lg tabular-nums',
              urgent && 'border-destructive text-destructive',
          )}
          aria-label="剩余时间"
        >
          <Timer className="size-4" />
          {Math.ceil(remaining / 1000)}
        </span>
      )}
    </header>
  );
}
