'use client';

import {useEffect, useState} from 'react';
import {Trophy} from 'lucide-react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {Tabs, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import {LEADERBOARD_PERIOD, type Leaderboard, type LeaderboardDetail} from '@/lib/services/leaderboard';
import type {BalanceItem} from '@/lib/services/wallet';
import {BoardValue, LeaderboardTable, RankBadge} from './LeaderboardTable';

/** 榜单的统计说明, 如 "排位 · 今日 2026-10-28 · 至少完成 3 局才上榜" */
function boardSummary(board: Leaderboard, detail: LeaderboardDetail | null) {
  const parts = [board.scope_name || '全部对局'];
  if (board.period === LEADERBOARD_PERIOD.day) {
    const day = detail?.current_period.replace(/^day:/, '');
    parts.push(day ? `今日 ${day}` : '今日');
  } else if (board.period === LEADERBOARD_PERIOD.season) {
    if (!detail) parts.push('赛季');
    else if (!detail.season_name) parts.push('赛季还没有开始');
    else parts.push(detail.season_ended ? `${detail.season_name} (已结束)` : detail.season_name);
  } else {
    parts.push('累计');
  }
  if (board.min_matches > 0) parts.push(`至少完成 ${board.min_matches} 局才上榜`);
  return parts.join(' · ');
}

/**
 * 排行榜: 按榜单分页签, 前 N 名表格, 底部固定显示自己的名次
 */
export function LeaderboardMain() {
  const {user} = useAuth();
  const [boards, setBoards] = useState<Leaderboard[] | null>(null);
  const [code, setCode] = useState('');
  const [detail, setDetail] = useState<LeaderboardDetail | null>(null);
  // 货币的小数位, 用于显示净收益
  const [currencies, setCurrencies] = useState<BalanceItem[]>([]);

  useEffect(() => {
    if (!user) return;
    services.leaderboard.listBoards()
        .then((data) => {
          setBoards(data.boards);
          setCode((prev) => prev || (data.boards[0]?.code ?? ''));
        })
        .catch((err) => toast.error(toErrorMessage(err)));
    services.wallet.listBalances()
        .then((data) => setCurrencies(data.balances))
        .catch(() => {
        // 只影响净收益的小数位, 失败时按整数显示
        });
  }, [user]);

  useEffect(() => {
    if (!code) return;
    let cancelled = false;
    services.leaderboard.getBoard(code)
        .then((data) => {
          if (!cancelled) setDetail(data);
        })
        .catch((err) => toast.error(toErrorMessage(err)));
    return () => {
      cancelled = true;
    };
  }, [code]);

  const board = boards?.find((b) => b.code === code);
  // 切换页签后不再显示上一个榜单的数据
  const current = detail?.code === code ? detail : null;
  const decimals = currencies.find((c) => c.currency === board?.currency)?.decimals ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">排行榜</h1>

      {!boards ? (
        <Skeleton className="h-9 w-72" />
      ) : !boards.length ? (
        <div className="flex flex-col items-center gap-2 rounded-xl border bg-card py-16 text-muted-foreground">
          <Trophy className="size-8" />
          暂时没有排行榜
        </div>
      ) : (
        <Tabs value={code} onValueChange={setCode}>
          <TabsList className="h-auto max-w-full flex-wrap justify-start">
            {boards.map((b) => (
              <TabsTrigger key={b.code} value={b.code}>
                {b.name}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}

      {board && (
        <>
          <div className="flex flex-col gap-1">
            {board.description && <p className="text-sm">{board.description}</p>}
            <p className="text-xs text-muted-foreground">
              {boardSummary(board, current)} · 前 {board.size} 名, 每分钟更新
            </p>
          </div>

          <LeaderboardTable board={board} entries={current?.entries ?? null} decimals={decimals} />

          {current && (
            <div className="sticky bottom-24 z-10 flex items-center gap-3 rounded-xl border bg-card/95 px-4 py-3 shadow-lg backdrop-blur">
              {current.mine.rank > 0 ? (
                <RankBadge rank={current.mine.rank} />
              ) : (
                <span className="text-sm text-muted-foreground">未上榜</span>
              )}
              <span className="flex-1 text-sm font-medium">我的名次</span>
              <BoardValue board={board} value={current.mine.value} decimals={decimals} />
            </div>
          )}
        </>
      )}
    </div>
  );
}
