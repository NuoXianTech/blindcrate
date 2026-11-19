'use client';

import {useCallback, useEffect, useRef, useState} from 'react';
import {useRouter} from 'next/navigation';
import {Map as MapIcon} from 'lucide-react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {Button} from '@/components/ui/button';
import {useAuth} from '@/components/common/auth';
import {CheckinCard} from '@/components/common/checkin';
import {MissionsBadge} from '@/components/common/missions';
import {useLockNavigation} from '@/hooks/use-navigation-lock';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import {matchHref} from '@/lib/access';
import {modeCode, parseMode, type ModeKind} from '@/lib/auction';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import {
  MATCH_MESSAGES,
  MATCH_STATUS,
  type MatchView,
  type MatchmakingFailedMessage,
  type MatchmakingRejectedMessage,
  type MatchmakingStatus,
  type QueueUpdatedMessage,
} from '@/lib/services/match';
import type {Venue} from '@/lib/services/venue';
import {isDismissed} from './CurrentMatchBanner';
import {MatchmakingBar} from './MatchmakingBar';
import {RatingBadge} from './RatingBadge';
import {SeasonBadge} from './SeasonBadge';
import {VenueCard} from './VenueCard';
import {VenueDetailDialog} from './VenueDetailDialog';

/**
 * 游戏大厅: 任务入口、当前赛季和段位分、签到、地图列表、地图详情和匹配
 * 加入后凑够一局时直接进入对局; 排队时锁定导航并等待 match:updated 推送, 被拒绝或开局失败时提示, 队列人数变化时更新人数
 * 推送不保证送达, 连上 WebSocket 后重新拉取匹配状态
 */
export function LobbyMain() {
  const {user} = useAuth();
  return user ? <LobbyContent key={user.id} /> : null;
}

function LobbyContent() {
  const router = useRouter();
  const connected = useWsConnected();
  const [venues, setVenues] = useState<Venue[] | null>(null);
  const [decimals, setDecimals] = useState<Record<string, number>>({});
  const [status, setStatus] = useState<MatchmakingStatus | null>(null);
  const [statusError, setStatusError] = useState('');
  const latestStatus = useRef<MatchmakingStatus | null>(null);
  const changing = useRef(false);
  const refreshRequested = useRef(false);
  const statusRead = useRef(0);
  const navigated = useRef<number | null>(null);
  const latestUpdate = useRef<{id: number; version: number} | null>(null);
  const [joining, setJoining] = useState<{ venue: string; kind: ModeKind } | null>(null);
  const [leaving, setLeaving] = useState(false);
  const [viewing, setViewing] = useState<Venue | null>(null);
  // 各地图选的带入金额, 没选过的地图没有
  const [carries, setCarries] = useState<Record<string, number>>({});
  // 签到后加一, 让任务入口重新拉取可领取的数量
  const [missionsVersion, setMissionsVersion] = useState(0);

  const invalidateStatus = useCallback(() => {
    statusRead.current++;
  }, []);

  const applyStatus = useCallback((next: MatchmakingStatus | null) => {
    invalidateStatus();
    latestStatus.current = next;
    setStatus(next);
    if (next) setStatusError('');
  }, [invalidateStatus]);

  const enterMatch = useCallback((id: number) => {
    // 已拿到开局结果, 不再补做等待期间的旧刷新请求。
    refreshRequested.current = false;
    applyStatus({queue: null, match_id: id});
    if (navigated.current === id) return;
    navigated.current = id;
    router.push(matchHref(id));
  }, [applyStatus, router]);

  const loadStatus = useCallback(async (afterMutation = false) => {
    if (changing.current && !afterMutation) {
      refreshRequested.current = true;
      return;
    }
    refreshRequested.current = false;
    const id = ++statusRead.current;
    try {
      const next = await services.match.getMatchmaking();
      if (id !== statusRead.current) return;
      applyStatus(next);
    } catch (err) {
      if (id !== statusRead.current) return;
      applyStatus(null);
      setStatusError(toErrorMessage(err));
      toast.error(toErrorMessage(err));
    }
  }, [applyStatus]);

  useEffect(() => {
    services.venue.listVenues()
        .then((data) => setVenues(data.venues))
        .catch((err) => toast.error(toErrorMessage(err)));
    services.auction.listCarry()
        .then((data) => setCarries(Object.fromEntries(data.preferences.map((p) => [p.venue, p.amount]))))
        .catch(() => {
          // 只影响带入金额的展示, 失败时显示上限, 准入仍按保存的金额
        });
    services.wallet.listBalances()
        .then((data) => setDecimals(Object.fromEntries(data.balances.map((b) => [b.currency, b.decimals]))))
        .catch(() => {
          // 只影响金额的小数位, 失败时按整数展示
        });
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入大厅和重连后拉取匹配状态
    void loadStatus();
  }, [connected, loadStatus]);

  useEffect(() => invalidateStatus, [invalidateStatus]);

  useWsEvent<MatchView>(MATCH_MESSAGES.updated, (view) => {
    const previous = latestUpdate.current;
    if (previous && (view.id < previous.id || (view.id === previous.id && view.version <= previous.version))) return;
    if (latestStatus.current?.match_id && latestStatus.current.match_id > view.id) return;
    latestUpdate.current = {id: view.id, version: view.version};
    // 已释放的对局只剩开箱展示, 离开后不再拉回去
    if (view.status !== MATCH_STATUS.playing || view.released_at) {
      if (latestStatus.current?.match_id === view.id) applyStatus({queue: null, match_id: 0});
      return;
    }
    // 断线回来后选择了暂不回去的对局也不再拉回去, 状态栏仍有回到对局的入口
    if (isDismissed(view.id)) {
      applyStatus({queue: null, match_id: view.id});
      return;
    }
    // 开局后已出队, 先更新状态, 免得回到大厅时按旧的排队状态锁定导航
    enterMatch(view.id);
  });

  useWsEvent<MatchmakingRejectedMessage>(MATCH_MESSAGES.matchmakingRejected, (msg) => {
    toast.error(`无法开局: ${msg.reason}`);
    void loadStatus();
  });

  useWsEvent<QueueUpdatedMessage>(MATCH_MESSAGES.queueUpdated, (msg) => {
    setStatus((prev) => prev?.queue?.mode === msg.mode ? {...prev, queue: {...prev.queue, count: msg.count}} : prev);
  });

  useWsEvent<MatchmakingFailedMessage>(MATCH_MESSAGES.matchmakingFailed, () => {
    toast.error('开局失败, 请重新匹配');
    void loadStatus();
  });

  const join = async (venue: Venue, kind: ModeKind) => {
    // ref 同步拦住同一帧内的人机/排位连点, 成功后先保存状态再跳转。
    if (changing.current || !latestStatus.current || latestStatus.current.queue || latestStatus.current.match_id) return;
    changing.current = true;
    const id = ++statusRead.current;
    setJoining({venue: venue.code, kind});
    try {
      const next = await services.match.joinMatchmaking(modeCode(kind, venue.code));
      if (id !== statusRead.current) return;
      if (next.match_id) {
        enterMatch(next.match_id);
        return;
      }
      applyStatus(next);
    } catch (err) {
      if (id !== statusRead.current) return;
      toast.error(toErrorMessage(err));
      // 请求失败也可能已经入队, 在确认状态之前不允许再次发起匹配。
      await loadStatus(true);
    } finally {
      changing.current = false;
      if (refreshRequested.current) await loadStatus();
      setJoining(null);
    }
  };

  const leave = async () => {
    if (changing.current || !latestStatus.current?.queue) return;
    changing.current = true;
    invalidateStatus();
    setLeaving(true);
    try {
      await services.match.leaveMatchmaking();
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      await loadStatus(true);
      changing.current = false;
      if (refreshRequested.current) await loadStatus();
      setLeaving(false);
    }
  };

  const queued = status?.queue ? parseMode(status.queue.mode) : null;
  const queuedVenue = venues?.find((v) => v.code === queued?.venue);
  const busy = !!status && (status.queue !== null || status.match_id !== 0);

  // 开局时只有大厅会进入对局, 加入和排队期间不能离开大厅
  useLockNavigation(joining !== null || !!status?.queue);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-bold">游戏大厅</h1>
        <div className="flex flex-wrap items-center gap-2">
          <MissionsBadge version={missionsVersion} />
          <RatingBadge />
          <SeasonBadge />
        </div>
      </div>

      {status && (
        <MatchmakingBar
          status={status}
          venueName={queuedVenue?.name}
          leaving={leaving}
          onLeave={leave}
        />
      )}

      {statusError && (
        <div className="flex items-center justify-between gap-2 text-sm">
          <p role="alert" className="text-destructive">{statusError}</p>
          <Button variant="outline" size="sm" onClick={() => void loadStatus()}>重新读取匹配状态</Button>
        </div>
      )}

      <CheckinCard onCheckedIn={() => setMissionsVersion((v) => v + 1)} />

      {venues === null ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({length: 3}, (_, i) => (
            <Skeleton key={i} className="h-80 rounded-xl" />
          ))}
        </div>
      ) : venues.length ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {venues.map((v) => (
            <VenueCard
              key={v.code}
              venue={v}
              decimals={decimals[v.bid_currency] ?? 0}
              busy={joining?.venue === v.code ? joining.kind : null}
              queued={queued?.venue === v.code ? queued.kind : null}
              disabled={status === null || busy || joining !== null || leaving}
              carry={Math.min(Math.max(carries[v.code] ?? v.auction.max_carry, v.auction.start_price), v.auction.max_carry)}
              onCarry={(amount) => setCarries((prev) => ({...prev, [v.code]: amount}))}
              onJoin={(kind) => join(v, kind)}
              onDetail={() => setViewing(v)}
            />
          ))}
        </div>
      ) : (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border bg-card text-muted-foreground">
          <MapIcon className="size-8" />
          <span className="text-sm">还没有开放的地图, 请稍后再来</span>
        </div>
      )}

      <VenueDetailDialog
        venue={viewing}
        decimals={viewing ? decimals[viewing.bid_currency] ?? 0 : 0}
        onClose={() => setViewing(null)}
      />
    </div>
  );
}
