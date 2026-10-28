'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {SearchX} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {useCountdown} from '@/hooks/use-countdown';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import {ROUTES} from '@/lib/access';
import services from '@/lib/services';
import {AUCTION_MESSAGES, type AuctionView} from '@/lib/services/auction';
import {ApiError, toErrorMessage} from '@/lib/services/core';
import {MATCH_MESSAGES, MATCH_STATUS, type MatchView} from '@/lib/services/match';
import {AuctionHeader} from './AuctionHeader';
import {BidPanel} from './BidPanel';
import {FeedbackList} from './FeedbackList';
import {ResultPanel} from './ResultPanel';
import {SeatList} from './SeatList';
import {Warehouse} from './Warehouse';

/** 阶段到期后还没收到推送时, 每隔这么久重新拉取一次 */
const STALE_RELOAD_MS = 3000;

/** 只采用 version 不比现有的旧的数据, 推送可能乱序 */
function newer<T extends { version: number }>(prev: T | null, next: T): T {
  return prev && prev.version > next.version ? prev : next;
}

/**
 * 拍卖桌: 仓库、座位、出价和结算
 * 以 auction:updated 和 match:updated 推送为准; 推送不保证送达, 连上 WebSocket 后和阶段到期后迟迟没有推送时重新拉取
 */
export function AuctionMain({matchId}: { matchId: number }) {
  const {user} = useAuth();
  const connected = useWsConnected();
  const [view, setView] = useState<AuctionView | null>(null);
  const [match, setMatch] = useState<MatchView | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [decimals, setDecimals] = useState(0);

  const load = useCallback(async () => {
    if (!user) return;
    try {
      const [auction, matchView] = await Promise.all([services.auction.getAuction(matchId), services.match.getMatch(matchId)]);
      setView((prev) => newer(prev, auction));
      setMatch((prev) => newer(prev, matchView));
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setNotFound(true);
        return;
      }
      toast.error(toErrorMessage(err));
    }
  }, [matchId, user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入对局和重连后拉取最新状态
    void load();
  }, [connected, load]);

  const currency = view?.venue.bid_currency;
  useEffect(() => {
    if (!user || !currency) return;
    services.wallet.listBalances()
        .then((data) => setDecimals(data.balances.find((b) => b.currency === currency)?.decimals ?? 0))
        .catch(() => {
          // 只影响金额的小数位, 失败时按整数展示
        });
  }, [currency, user]);

  useWsEvent<AuctionView>(AUCTION_MESSAGES.updated, (data) => {
    if (data.match_id === matchId) setView((prev) => newer(prev, data));
  });

  useWsEvent<MatchView>(MATCH_MESSAGES.updated, (data) => {
    if (data.id === matchId) setMatch((prev) => newer(prev, data));
  });

  const finished = !!match && match.status !== MATCH_STATUS.playing;
  const remaining = useCountdown(finished ? null : view?.phase_ends_at ?? null);
  const expired = remaining === 0;

  useEffect(() => {
    if (!expired) return;
    const timer = setInterval(() => void load(), STALE_RELOAD_MS);
    return () => clearInterval(timer);
  }, [expired, load]);

  if (notFound) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 px-4 text-center">
        <SearchX className="size-10 text-muted-foreground" />
        <h1 className="text-xl font-bold">找不到这局对局</h1>
        <p className="text-sm text-muted-foreground">对局不存在, 或者你不是这局的玩家</p>
        <Button asChild>
          <Link href={ROUTES.gameHome}>返回大厅</Link>
        </Button>
      </div>
    );
  }

  if (!view || !match) {
    return (
      <div className="mx-auto grid w-full max-w-6xl gap-4 p-4 lg:grid-cols-[1fr_380px]">
        <Skeleton className="aspect-square rounded-xl" />
        <div className="flex flex-col gap-4">
          <Skeleton className="h-32 rounded-xl" />
          <Skeleton className="h-56 rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <>
      <AuctionHeader view={view} finished={finished} remaining={remaining} />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto grid w-full max-w-6xl gap-6 p-4 lg:grid-cols-[1fr_380px]">
          <Warehouse view={view} decimals={decimals} finished={finished} />
          <div className="flex flex-col gap-4">
            <SeatList view={view} players={match.players} decimals={decimals} />
            {view.result ? (
              <ResultPanel
                view={view}
                players={match.players}
                decimals={decimals}
                finished={finished}
                released={finished || !!match.released_at}
              />
            ) : (
              <BidPanel
                key={view.round}
                view={view}
                decimals={decimals}
                onBid={(next) => setView((prev) => newer(prev, next))}
              />
            )}
            <section className="flex flex-col gap-2">
              <h2 className="text-sm font-medium">每轮反馈</h2>
              <FeedbackList feedback={view.self.feedback} />
            </section>
          </div>
        </div>
      </main>
    </>
  );
}
