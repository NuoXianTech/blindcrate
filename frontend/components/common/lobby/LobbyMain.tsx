'use client';

import {useCallback, useEffect, useState} from 'react';
import {useRouter} from 'next/navigation';
import {Map as MapIcon} from 'lucide-react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import {CheckinCard} from '@/components/common/checkin';
import {MissionsBadge} from '@/components/common/missions';
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
import {MatchmakingBar} from './MatchmakingBar';
import {RatingBadge} from './RatingBadge';
import {SeasonBadge} from './SeasonBadge';
import {VenueCard} from './VenueCard';
import {VenueDetailDialog} from './VenueDetailDialog';

/**
 * 游戏大厅: 任务入口、当前赛季和段位分、签到、地图列表、地图详情和匹配
 * 加入后凑够一局时直接进入对局; 排队时等待 match:updated 推送, 被拒绝或开局失败时提示, 队列人数变化时更新人数
 * 推送不保证送达, 连上 WebSocket 后重新拉取匹配状态
 */
export function LobbyMain() {
  const router = useRouter();
  const {user} = useAuth();
  const connected = useWsConnected();
  const [venues, setVenues] = useState<Venue[] | null>(null);
  const [decimals, setDecimals] = useState<Record<string, number>>({});
  const [status, setStatus] = useState<MatchmakingStatus | null>(null);
  const [joining, setJoining] = useState<{ venue: string; kind: ModeKind } | null>(null);
  const [leaving, setLeaving] = useState(false);
  const [viewing, setViewing] = useState<Venue | null>(null);
  // 签到后加一, 让任务入口重新拉取可领取的数量
  const [missionsVersion, setMissionsVersion] = useState(0);

  const loadStatus = useCallback(async () => {
    if (!user) return;
    try {
      setStatus(await services.match.getMatchmaking());
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user]);

  useEffect(() => {
    if (!user) return;
    services.venue.listVenues()
        .then((data) => setVenues(data.venues))
        .catch((err) => toast.error(toErrorMessage(err)));
    services.wallet.listBalances()
        .then((data) => setDecimals(Object.fromEntries(data.balances.map((b) => [b.currency, b.decimals]))))
        .catch(() => {
          // 只影响金额的小数位, 失败时按整数展示
        });
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 进入大厅和重连后拉取匹配状态
    void loadStatus();
  }, [connected, loadStatus]);

  useWsEvent<MatchView>(MATCH_MESSAGES.updated, (view) => {
    // 已释放的对局只剩开箱展示, 离开后不再拉回去
    if (view.status === MATCH_STATUS.playing && !view.released_at) router.push(matchHref(view.id));
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
    setJoining({venue: venue.code, kind});
    try {
      const next = await services.match.joinMatchmaking(modeCode(kind, venue.code));
      if (next.match_id) {
        router.push(matchHref(next.match_id));
        return;
      }
      setStatus(next);
    } catch (err) {
      toast.error(toErrorMessage(err));
      void loadStatus();
    } finally {
      setJoining(null);
    }
  };

  const leave = async () => {
    setLeaving(true);
    try {
      await services.match.leaveMatchmaking();
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLeaving(false);
      void loadStatus();
    }
  };

  // 排位人少时改打人机: 先离开队列再加入人机
  const switchToPve = async (venue: Venue) => {
    setLeaving(true);
    try {
      await services.match.leaveMatchmaking();
    } catch (err) {
      toast.error(toErrorMessage(err));
      void loadStatus();
      return;
    } finally {
      setLeaving(false);
    }
    await join(venue, 'pve');
  };

  const queued = status?.queue ? parseMode(status.queue.mode) : null;
  const queuedVenue = venues?.find((v) => v.code === queued?.venue);
  const busy = !!status && (status.queue !== null || status.match_id !== 0);

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
          onSwitchPve={queuedVenue?.pve_enabled ? () => void switchToPve(queuedVenue) : undefined}
        />
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
              disabled={busy || joining !== null}
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
